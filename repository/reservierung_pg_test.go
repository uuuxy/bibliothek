package repository

import (
	"bibliothek/internal/testdaten"
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Die Warteschlange der Reservierungen am echten Postgres: Reservieren sperrt nichts; die
// Reihenfolge (die älteste zuerst) und die je Zeile mitgelieferte Verfügbarkeit sind die ganze
// Steuerung. Beides ist Verhalten der Abfrage (ORDER BY über zwei CASE-Zweige, die Zählung wie
// im Katalog).
func TestReservierungen_WarteschlangeUndVerfuegbarkeit(t *testing.T) {
	pool := pgTestPool(t)
	ctx := context.Background()
	repo := NewReservierungsAbfragen(pool)

	// Ein Titel mit drei Exemplaren. legeSignaturMitExemplarenAn taugt hier nicht: Es legt je
	// Exemplar einen eigenen Titel an, und die Verfügbarkeit wäre immer 1.
	titel := testdaten.LegeTitelMitSignaturAn(t, pool, "KSQ-Klassensatz", "KSQ", 5)
	exemplare := make([]string, 3)
	for i := range exemplare {
		if err := pool.QueryRow(ctx,
			`INSERT INTO buecher_exemplare (titel_id, barcode_id) VALUES ($1, $2) RETURNING id`,
			titel, "BC-KSQ-Q-"+string(rune('0'+i))).Scan(&exemplare[i]); err != nil {
			t.Fatalf("Exemplar anlegen: %v", err)
		}
	}
	bearbeiter := legeEigenenBearbeiterAn(t, pool, "KSQ-B")

	// Zwei Reservierungen nacheinander: 8a zuerst, 9b stellt sich an.
	ersteID, _, err := repo.LegeAn(ctx, titel, "8a", 3, nil, bearbeiter, nil)
	if err != nil {
		t.Fatalf("Reservierung 8a: %v", err)
	}
	if _, _, err := repo.LegeAn(ctx, titel, "9b", 2, nil, bearbeiter, nil); err != nil {
		t.Fatalf("Reservierung 9b: %v", err)
	}

	// (1) Die Liste der Bibliothek: Reihenfolge der Warteschlange (die älteste zuerst),
	// verfügbar = 3.
	liste, err := repo.LadeAlle(ctx)
	if err != nil {
		t.Fatalf("Liste: %v", err)
	}
	meine := nurTitel(liste, titel)
	if len(meine) != 2 || meine[0].Klasse != "8a" || meine[1].Klasse != "9b" {
		t.Fatalf("Warteschlange = %+v, erwartet [8a, 9b]", meine)
	}
	if meine[0].Verfuegbar != 3 {
		t.Errorf("verfügbar = %d, erwartet 3 (alle im Regal)", meine[0].Verfuegbar)
	}
	// Der Anfragende steht mit Namen in der Liste: Die Bibliothek muss wissen, welche Lehrkraft
	// sie anrufen soll.
	if meine[0].AngefordertVon == nil || *meine[0].AngefordertVon != "Klassensatz Kraft" {
		t.Errorf("angefordert_von = %v, erwartet den Klarnamen der Lehrkraft", meine[0].AngefordertVon)
	}

	// (2) Ein Exemplar wird verliehen, verfügbar sinkt auf 2. Diese Zahl erspart der Bibliothek
	// das Zählen am Regal.
	schueler := legeSchuelerAn(t, pool, "S-KSQ-1", "Kim", "8a")
	legeAusleiheAn(t, pool, exemplare[0], schueler, bearbeiter)

	liste, err = repo.LadeAlle(ctx)
	if err != nil {
		t.Fatalf("Liste nach Ausleihe: %v", err)
	}
	if v := nurTitel(liste, titel)[0].Verfuegbar; v != 2 {
		t.Errorf("verfügbar nach Ausleihe = %d, erwartet 2", v)
	}

	// (3) Die Sicht des Portals führt beide in derselben Reihenfolge, ohne Personendaten.
	offene, err := repo.LadeOffene(ctx)
	if err != nil {
		t.Fatalf("Offene: %v", err)
	}
	offeneMeine := []OffeneReservierung{}
	for _, o := range offene {
		if o.TitelID == titel {
			offeneMeine = append(offeneMeine, o)
		}
	}
	if len(offeneMeine) != 2 || offeneMeine[0].Klasse != "8a" || offeneMeine[1].Klasse != "9b" {
		t.Fatalf("Portal-Warteschlange = %+v, erwartet [8a, 9b]", offeneMeine)
	}
	// Der Titel kommt mit: Auf der Startfläche des Portals steht die Warteschlange ohne ein
	// Buch daneben, und „Klasse 8a · 3 Stück" allein sagt niemandem, worum es geht.
	if offeneMeine[0].Titel == "" || offeneMeine[1].Titel == "" {
		t.Errorf("Titel fehlt in der Portal-Warteschlange: %+v", offeneMeine)
	}

	// (4) 8a wird erledigt: Die 9b rückt in der offenen Liste nach vorn, die erledigte 8a
	// bleibt dahinter stehen. Das Abschließen liefert die Angaben für die Mail, mit der Adresse
	// des Kontos der Lehrkraft.
	erledigt, err := repo.Erledige(ctx, ersteID, "")
	if err != nil {
		t.Fatalf("Erledigen: %v", err)
	}
	if erledigt == nil || erledigt.Klasse != "8a" || erledigt.Anzahl != 3 ||
		erledigt.AnfragendeMail == nil || *erledigt.AnfragendeMail != "KSQ-B@example.org" {
		t.Fatalf("Bereit-Mail-Angaben falsch: %+v", erledigt)
	}
	// Der zweite Klick schließt nichts erneut ab und löst keine zweite Mail aus.
	if nochmal, err := repo.Erledige(ctx, ersteID, ""); err != nil || nochmal != nil {
		t.Fatalf("Doppel-Erledigen = %+v / %v, erwartet nil ohne Fehler", nochmal, err)
	}
	liste, err = repo.LadeAlle(ctx)
	if err != nil {
		t.Fatalf("Liste nach Erledigen: %v", err)
	}
	meine = nurTitel(liste, titel)
	if len(meine) != 2 || meine[0].Klasse != "9b" || meine[0].Erledigt || !meine[1].Erledigt {
		t.Fatalf("Warteschlange nach Erledigen = %+v, erwartet [9b offen, 8a erledigt]", meine)
	}
}

// nurTitel filtert die Liste auf einen Titel; andere Tests des Pakets hinterlassen eigene
// Reservierungen.
func nurTitel(liste []Reservierung, titelID string) []Reservierung {
	out := []Reservierung{}
	for _, r := range liste {
		if r.TitelID == titelID {
			out = append(out, r)
		}
	}
	return out
}

// legeEigenenBearbeiterAn legt ein Konto mit eigenem Barcode an: Der feste Barcode von
// legeBearbeiterAn gehört den Tests in schaden_melden_ausleihe_pg_test.go, und mit einem
// geteilten Konto hinge das Ergebnis an der Reihenfolge der Tests.
func legeEigenenBearbeiterAn(t *testing.T, pool *pgxpool.Pool, barcode string) string {
	t.Helper()
	id := testdaten.LegeKontoAn(t, pool, "Klassensatz", "Kraft", barcode+"@example.org", "mitarbeiter", true)
	return id
}

// Der Schutz vor dem Doppelklick: Zwei Anfragen mit demselben Idempotenz-Schlüssel erzeugen
// eine Reservierung; die zweite liefert dieselbe ID und neu = false. Ein anderer Schlüssel legt
// eine weitere an.
func TestReservierungIdempotenz(t *testing.T) {
	pool := pgTestPool(t)
	resetInventurDaten(t, pool)
	ctx := context.Background()
	repo := NewReservierungsAbfragen(pool)

	ex := legeSignaturMitExemplarenAn(t, pool, "KsIdem", 3)
	titel := titelIDVonExemplar(t, pool, ex[0])
	bearbeiter := legeBearbeiterAn(t, pool)
	key := "11111111-1111-1111-1111-111111111111"

	id1, neu1, err := repo.LegeAn(ctx, titel, "8a", 2, nil, bearbeiter, &key)
	if err != nil || !neu1 {
		t.Fatalf("erste Reservierung: id=%q neu=%v err=%v", id1, neu1, err)
	}
	// Doppelklick: gleicher Schlüssel, dieselbe ID, neu = false, keine zweite Zeile.
	id2, neu2, err := repo.LegeAn(ctx, titel, "8a", 2, nil, bearbeiter, &key)
	if err != nil {
		t.Fatalf("zweiter Klick: %v", err)
	}
	if neu2 {
		t.Error("zweiter Klick mit gleichem Schlüssel muss No-op sein (neu=false)")
	}
	if id2 != id1 {
		t.Errorf("zweiter Klick muss dieselbe ID liefern: %q vs %q", id2, id1)
	}

	var anzahl int
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM klassensatz_reservierungen WHERE idempotenz_schluessel = $1`, key).Scan(&anzahl); err != nil {
		t.Fatal(err)
	}
	if anzahl != 1 {
		t.Fatalf("anzahl = %d, erwartet genau 1 Reservierung", anzahl)
	}

	// Ein anderer Schlüssel (eine gewollte zweite Reservierung) legt an.
	key2 := "22222222-2222-2222-2222-222222222222"
	if _, neu3, err := repo.LegeAn(ctx, titel, "8a", 1, nil, bearbeiter, &key2); err != nil || !neu3 {
		t.Fatalf("bewusste zweite Reservierung muss durchgehen: neu=%v err=%v", neu3, err)
	}
}
