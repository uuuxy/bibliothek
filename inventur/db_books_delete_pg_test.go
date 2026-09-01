package inventur

import (
	"context"
	"strings"
	"testing"

	"bibliothek/internal/pgtest"
	"bibliothek/repository"
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
	pool := pgtest.Pool(t)
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

	// pgtest teilt das Schema mit allen Tests des Binaries — die eigenen Zeilen werden
	// deshalb abgeräumt. Die IDs stehen VOR dem Cleanup, damit die Closure sie sieht;
	// die Guards greifen, wenn ein früher t.Fatal einen Teil nie angelegt hat.
	// Reihenfolge wegen ON DELETE RESTRICT (ausleihen/schadensfaelle → exemplare):
	// erst die abhängigen Zeilen, dann der Titel (Exemplare kaskadieren), dann der
	// Schüler; die Protokollzeilen aus DeleteBooks zuletzt scoped über datensatz_id.
	// Im Erfolgsfall hat DeleteBooks das meiste schon entfernt — die DELETEs treffen
	// dann 0 Zeilen, und genau das ist in Ordnung.
	var titelID, imRegal, verliehen, schuelerID string
	t.Cleanup(func() {
		ctx := context.Background()
		exemplare := []string{}
		for _, id := range []string{imRegal, verliehen} {
			if id != "" {
				exemplare = append(exemplare, id)
			}
		}
		if len(exemplare) > 0 {
			for _, tabelle := range []string{"schadensfaelle", "ausleihen"} {
				if _, err := pool.Exec(ctx,
					`DELETE FROM `+tabelle+` WHERE exemplar_id = ANY($1)`, exemplare); err != nil {
					t.Errorf("Aufräumen %s: %v", tabelle, err)
				}
			}
			if _, err := pool.Exec(ctx,
				`DELETE FROM audit_log
				 WHERE tabelle IN ('ausleihen', 'buecher_exemplare') AND datensatz_id = ANY($1)`,
				exemplare); err != nil {
				t.Errorf("Aufräumen audit_log: %v", err)
			}
		}
		if titelID != "" {
			if _, err := pool.Exec(ctx, `DELETE FROM buecher_titel WHERE id = $1`, titelID); err != nil {
				t.Errorf("Aufräumen Titel: %v", err)
			}
		}
		if schuelerID != "" {
			if _, err := pool.Exec(ctx, `DELETE FROM schueler WHERE id = $1`, schuelerID); err != nil {
				t.Errorf("Aufräumen Schüler: %v", err)
			}
		}
	})

	titelID = eins("Titel", `INSERT INTO buecher_titel (titel) VALUES ('Der Zauberberg') RETURNING id`)
	imRegal = eins("Exemplar im Regal",
		`INSERT INTO buecher_exemplare (titel_id, barcode_id) VALUES ($1, 'ZB-REGAL') RETURNING id`, titelID)
	verliehen = eins("Exemplar verliehen",
		`INSERT INTO buecher_exemplare (titel_id, barcode_id) VALUES ($1, 'ZB-DRAUSSEN') RETURNING id`, titelID)
	schuelerID = eins("Schüler", `
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
	// schueler_id ist kein Beiwerk, sondern der Schlüssel, an dem die
	// Lesehistorie-Befristung diese Zeile findet (`details ? 'schueler_id'`). Fehlt er,
	// steht der Klarname 24 Monate statt 90 Tage — und der Wächter der Selbstprüfung
	// sieht das nicht, weil er dieselbe Frage stellt wie der Job.
	if !strings.Contains(details, `"schueler_id"`) {
		t.Errorf("Protokollzeile ohne schueler_id — die Befristung erreicht den Klarnamen nie: %s", details)
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

	// 4. BEIDE Exemplare bleiben am Tresen auffindbar — nicht nur das verliehene.
	//    Die Tresen-Auskunft findet gelöschte Exemplare ausschließlich über
	//    audit_log-Zeilen mit tabelle='buecher_exemplare' und details->>'barcode_id'
	//    (repository.SucheTresenExemplare); die Ausleihen-Spur aus Schritt 2 sieht sie
	//    nicht. Bis zum 01.09.2026 fehlte dieser Snapshot hier wie beim
	//    Geschwister-Pfad DeleteTitle (Befund-Register): Ein per Titel-Löschung
	//    verschwundenes Buch war beim Scannen „nie gesehen" statt „gelöscht am …".
	//    Geprüft über den echten Leseweg, nicht über count(*) — eine Zeile im falschen
	//    Format bestünde die Zählung und bliebe trotzdem unsichtbar.
	for _, fall := range []struct{ barcode, exemplarID string }{
		{"ZB-REGAL", imRegal},
		{"ZB-DRAUSSEN", verliehen},
	} {
		zeilen, err := repository.SucheTresenExemplare(ctx, pool, fall.barcode)
		if err != nil {
			t.Fatalf("Tresen-Auskunft für %s: %v", fall.barcode, err)
		}
		if len(zeilen) != 1 {
			t.Fatalf("Barcode %s: %d Treffer in der Tresen-Auskunft, erwartet genau 1", fall.barcode, len(zeilen))
		}
		if zeilen[0].Status != "geloescht" || zeilen[0].ExemplarID != fall.exemplarID {
			t.Errorf("Barcode %s: Status %q / Exemplar %s, erwartet geloescht / %s",
				fall.barcode, zeilen[0].Status, zeilen[0].ExemplarID, fall.exemplarID)
		}
		if zeilen[0].Titel != "Der Zauberberg" {
			t.Errorf("Barcode %s: Auskunft nennt Titel %q, erwartet Der Zauberberg", fall.barcode, zeilen[0].Titel)
		}
	}
}
