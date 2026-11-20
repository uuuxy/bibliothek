package repository

import (
	"context"
	"testing"

	"bibliothek/internal/pgtest"
)

// Ein Exemplar im Abholfach zählt nicht als verfügbar, solange die Abholfrist läuft: Die Theke
// gibt es keinem anderen (FindeVormerkungAmExemplar). Katalog, Suche, Klassensätze und Lernmittel
// zeigten es trotzdem als „1 verfügbar“, und wer danach im Regal suchte, fand nichts. Nach Ablauf
// der Frist und bei einer bloß wartenden Vormerkung ist es wieder verfügbar.
func TestVerfuegbar_OhneExemplarImAbholfach(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	tx := beginne(t, pool)
	defer func() {
		if err := tx.Rollback(ctx); err != nil {
			t.Errorf("zurückrollen: %v", err)
		}
	}()

	var schueler string
	if err := tx.QueryRow(ctx, `INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr)
		VALUES ('A-VERF-1', 'Vorname', 'Nachname', '06C', 2032) RETURNING id::text`).Scan(&schueler); err != nil {
		t.Fatalf("Schüler anlegen: %v", err)
	}
	titelMitExemplar := func(name, status, frist string) string {
		t.Helper()
		var id string
		if err := tx.QueryRow(ctx, `WITH t AS (INSERT INTO buecher_titel (titel) VALUES ($1) RETURNING id),
			     e AS (INSERT INTO buecher_exemplare (titel_id, barcode_id) SELECT id, $2 FROM t RETURNING id, titel_id),
			     v AS (INSERT INTO vormerkungen (titel_id, schueler_id, status, bereitgestellt_exemplar_id, bereitgestellt_bis)
			           SELECT titel_id, $3, $4::text, id, now() + $5::interval FROM e WHERE $4::text <> '')
			SELECT titel_id::text FROM e`, "Verfügbar "+name, "VERF-"+name, schueler, status, frist).Scan(&id); err != nil {
			t.Fatalf("%s anlegen: %v", name, err)
		}
		return id
	}

	je := []struct {
		name, status, frist string
		verfuegbar          int
	}{
		{"im Abholfach", "abholbereit", "2 days", 0},
		{"Abholfrist abgelaufen", "abholbereit", "-1 hour", 1},
		{"Vormerkung wartet", "wartend", "2 days", 1},
		{"ohne Vormerkung", "", "0", 1},
	}
	for _, f := range je {
		titelID := titelMitExemplar(f.name, f.status, f.frist)
		var jeTitel, gejoint int
		if err := tx.QueryRow(ctx, `SELECT `+SQLBestandVerfuegbar+` FROM buecher_titel b WHERE b.id = $1`, titelID).Scan(&jeTitel); err != nil {
			t.Fatalf("%s: SQLBestandVerfuegbar: %v", f.name, err)
		}
		if err := tx.QueryRow(ctx, `SELECT `+sqlFilterVerfuegbar+`
			FROM buecher_titel b
			LEFT JOIN buecher_exemplare e ON e.titel_id = b.id
			LEFT JOIN ausleihen a ON a.exemplar_id = e.id AND a.rueckgabe_am IS NULL
			WHERE b.id = $1`, titelID).Scan(&gejoint); err != nil {
			t.Fatalf("%s: sqlFilterVerfuegbar: %v", f.name, err)
		}
		if jeTitel != f.verfuegbar || gejoint != f.verfuegbar {
			t.Errorf("%s: verfügbar %d (je Titel) und %d (gejoint), erwartet %d", f.name, jeTitel, gejoint, f.verfuegbar)
		}
	}
}
