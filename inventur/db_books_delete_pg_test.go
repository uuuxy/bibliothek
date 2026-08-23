package inventur

import (
	"context"
	"strings"
	"testing"
)

// „Titel löschen" gegen ein echtes Postgres — an der Regel, die der Betreiber am
// 23.08.2026 gesetzt hat: Es geht ALLES, auch ein Exemplar, das gerade bei einem Kind
// zu Hause liegt. Vorher brach der Lauf dann ab; ein versehentlich importierter Titel
// liess sich nicht aufräumen, bis das letzte Buch zurück war.
//
// Der Test prüft deshalb zwei Dinge zusammen, denn nur zusammen sind sie vertretbar:
// dass wirklich alles verschwindet — UND dass das verliehene Buch eine Spur hinterlässt.
// Ein spurlos verschwundenes Buch ist ein verlorenes Buch: Wer es zurückbringt, findet
// beim Scannen nichts mehr vor, und ohne Protokollzeile weiss niemand mehr, wer es hatte.
func TestDeleteBooks_LoeschtAuchVerlieheneUndHinterlaesstSpur(t *testing.T) {
	pool := ladeSchemaPool(t)
	ctx := context.Background()
	repo := NewBookRepository(pool)

	eins := func(was, sql string, args ...any) string {
		t.Helper()
		var id string
		if err := pool.QueryRow(ctx, sql, args...).Scan(&id); err != nil {
			t.Fatalf("%s: %v", was, err)
		}
		return id
	}

	titelID := eins("Titel", `INSERT INTO buecher_titel (titel) VALUES ('Der Zauberberg') RETURNING id`)
	imRegal := eins("Exemplar im Regal",
		`INSERT INTO buecher_exemplare (titel_id, barcode_id) VALUES ($1, 'ZB-REGAL') RETURNING id`, titelID)
	verliehen := eins("Exemplar verliehen",
		`INSERT INTO buecher_exemplare (titel_id, barcode_id) VALUES ($1, 'ZB-DRAUSSEN') RETURNING id`, titelID)
	schuelerID := eins("Schüler", `
		INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr)
		VALUES ('ZB-S1', 'Hans', 'Castorp', '9a', 2030) RETURNING id`)

	// Eine LAUFENDE Ausleihe (rueckgabe_am IS NULL) — das Buch ist bei ihm zu Hause.
	if _, err := pool.Exec(ctx, `
		INSERT INTO ausleihen (exemplar_id, schueler_id, ausgeliehen_am, rueckgabe_frist)
		VALUES ($1, $2, now() - interval '5 days', now() + interval '9 days')`,
		verliehen, schuelerID); err != nil {
		t.Fatalf("laufende Ausleihe: %v", err)
	}
	// Dazu eine abgeschlossene mit unbezahlter Gebühr am anderen Exemplar: Beim
	// Titel-Löschen geht auch die mit — bewusste Entscheidung des Betreibers
	// („ist das Buch wohl so alt, dass es egal ist").
	altAusleihe := eins("alte Ausleihe", `
		INSERT INTO ausleihen (exemplar_id, schueler_id, ausgeliehen_am, rueckgabe_frist, rueckgabe_am)
		VALUES ($1, $2, now() - interval '60 days', now() - interval '46 days', now() - interval '40 days')
		RETURNING id`, imRegal, schuelerID)
	if _, err := pool.Exec(ctx, `
		INSERT INTO schadensfaelle (exemplar_id, ausleihe_id, schueler_id, beschreibung, betrag, ist_bezahlt)
		VALUES ($1, $2, $3, 'Fleck', 3.00, false)`, imRegal, altAusleihe, schuelerID); err != nil {
		t.Fatalf("Gebühr: %v", err)
	}

	if err := repo.DeleteBooks(ctx, []string{titelID}); err != nil {
		t.Fatalf("Titel mit verliehenem Exemplar muss löschbar sein: %v", err)
	}

	// 1. Wirklich alles weg — Titel, BEIDE Exemplare, alle Ausleihen, die Gebühr.
	for _, fall := range []struct {
		name, sql string
	}{
		{"Titel", `SELECT count(*) FROM buecher_titel WHERE id = $1`},
		{"Exemplare", `SELECT count(*) FROM buecher_exemplare WHERE titel_id = $1`},
	} {
		var n int
		if err := pool.QueryRow(ctx, fall.sql, titelID).Scan(&n); err != nil {
			t.Fatal(err)
		}
		if n != 0 {
			t.Errorf("%s: %d Zeilen überlebten", fall.name, n)
		}
	}
	for _, fall := range []struct {
		name, sql string
	}{
		{"Ausleihen", `SELECT count(*) FROM ausleihen WHERE exemplar_id = ANY($1)`},
		{"Schadensfälle", `SELECT count(*) FROM schadensfaelle WHERE exemplar_id = ANY($1)`},
	} {
		var n int
		if err := pool.QueryRow(ctx, fall.sql, []string{imRegal, verliehen}).Scan(&n); err != nil {
			t.Fatal(err)
		}
		if n != 0 {
			t.Errorf("%s: %d Zeilen überlebten", fall.name, n)
		}
	}

	// 2. Die Spur des verliehenen Buchs. Ohne sie wäre die Löschung lautlos, und ein
	//    zurückgebrachtes Buch hätte niemanden mehr, dem es zuzuordnen wäre.
	var kontext, details string
	if err := pool.QueryRow(ctx, `
		SELECT kontext, details::text FROM audit_log
		WHERE tabelle = 'ausleihen' AND aktion = 'DELETE' AND datensatz_id = $1`,
		verliehen).Scan(&kontext, &details); err != nil {
		t.Fatalf("keine Protokollzeile für das verliehene Buch: %v", err)
	}
	if !strings.Contains(kontext, "verliehen") {
		t.Errorf("Kontext sagt nicht, was los war: %q", kontext)
	}
	for _, muss := range []string{"ZB-DRAUSSEN", "Zauberberg", "Hans Castorp"} {
		if !strings.Contains(details, muss) {
			t.Errorf("Protokollzeile ohne %q — nicht nachschlagbar: %s", muss, details)
		}
	}

	// 3. Kein Rauschen: Das Exemplar aus dem Regal war nicht verliehen und darf keine
	//    solche Zeile haben — sonst stünde bei jedem Aufräumen eine Warnung ohne Anlass.
	var rauschen int
	if err := pool.QueryRow(ctx, `
		SELECT count(*) FROM audit_log
		WHERE tabelle = 'ausleihen' AND aktion = 'DELETE' AND datensatz_id = $1`,
		imRegal).Scan(&rauschen); err != nil {
		t.Fatal(err)
	}
	if rauschen != 0 {
		t.Errorf("%d Protokollzeilen für ein Exemplar, das gar nicht verliehen war", rauschen)
	}
}
