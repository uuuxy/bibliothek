package repository

// Die Theke findet JEDEN aktiven Leser über seinen Ausweis — Schüler wie Kollegium
// (Migration 125, 16.09.2026). Vorher entschied das eine eigene Abfrage über die
// Kontentabelle und die Personenart; eine Lehrkraft ohne dieses Feld fand die Theke nicht.
//
// Die Gegenprobe steht mit im selben Test und ist der eigentliche Punkt: Dieselbe Person
// darf über die SICHT `schueler` NICHT auffindbar sein. Liefe GetByBarcode auf die Tabelle,
// stünde ein Kollege in Klassenlisten, Mahnläufen und der Abgänger-Versetzung.

import (
	"context"
	"testing"

	"bibliothek/internal/pgtest"
)

func TestGetLeserByBarcode_FindetSchuelerUndKollegium(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	t.Cleanup(func() {
		if _, err := pool.Exec(ctx, `DELETE FROM leser WHERE nachname = 'Ausweisprobe'`); err != nil {
			t.Errorf("aufräumen: %v", err)
		}
	})

	anlegen := func(barcode, art string) {
		t.Helper()
		klasse, jahr := "05F1", 2030
		if art != "schueler" {
			if _, err := pool.Exec(ctx, `INSERT INTO leser (barcode_id, vorname, nachname, art)
				VALUES ($1, 'Probe', 'Ausweisprobe', $2)`, barcode, art); err != nil {
				t.Fatalf("Leser %s anlegen: %v", barcode, err)
			}
			return
		}
		if _, err := pool.Exec(ctx, `INSERT INTO leser (barcode_id, vorname, nachname, art, klasse, abgaenger_jahr)
			VALUES ($1, 'Probe', 'Ausweisprobe', $2, $3, $4)`, barcode, art, klasse, jahr); err != nil {
			t.Fatalf("Leser %s anlegen: %v", barcode, err)
		}
	}
	anlegen("LP-SCH", "schueler")
	anlegen("LP-LK", "lehrkraft")
	anlegen("LP-LIV", "liv")

	repo := NewStudentRepository(pool)

	for barcode, art := range map[string]string{"LP-SCH": "schueler", "LP-LK": "lehrkraft", "LP-LIV": "liv"} {
		leser, err := repo.GetLeserByBarcode(ctx, barcode)
		if err != nil {
			t.Fatalf("%s: %v", barcode, err)
		}
		if leser == nil {
			t.Errorf("%s: die Theke findet diesen Ausweis nicht", barcode)
			continue
		}
		if leser.Art != art {
			t.Errorf("%s: Art %q, erwartet %q", barcode, leser.Art, art)
		}
	}

	// Gegenprobe über die Sicht: nur der Schüler.
	for barcode, sichtbar := range map[string]bool{"LP-SCH": true, "LP-LK": false, "LP-LIV": false} {
		s, err := repo.GetByBarcode(ctx, barcode)
		if err != nil {
			t.Fatalf("%s über die Sicht: %v", barcode, err)
		}
		if (s != nil) != sichtbar {
			t.Errorf("%s: über die Sicht schueler gefunden=%v, erwartet %v", barcode, s != nil, sichtbar)
		}
	}
}
