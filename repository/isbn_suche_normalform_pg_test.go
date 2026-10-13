package repository

import (
	"context"
	"testing"
)

// Die Suche findet eine ISBN in jeder Schreibweise (Migration 133).
//
// Die Datenbank speichert eine ISBN ohne Bindestriche; wer sie vom Buchrücken abtippt,
// tippt sie mit. Ein zeichengenauer Vergleich fände mit „978-3-16"
// einen Titel, der als 9783161484100 gespeichert ist, nicht. Deshalb vergleichen beide
// Seiten ohne Trenner — auch für Altbestand, der noch mit Bindestrichen gespeichert ist.
func TestSuche_FindetISBNInJederSchreibweise(t *testing.T) {
	pool := pgTestPool(t)
	ctx := context.Background()
	t.Cleanup(func() {
		if _, err := pool.Exec(ctx, `DELETE FROM buecher_titel WHERE titel = 'Suchprobe Normalform'`); err != nil {
			t.Errorf("aufräumen: %v", err)
		}
	})
	// Mit Exemplar: ohne eines zeigt die Suche keinen Titel.
	if _, err := pool.Exec(ctx, `WITH t AS (
			INSERT INTO buecher_titel (titel, isbn) VALUES ('Suchprobe Normalform', '978-3-16-148410-0') RETURNING id)
		INSERT INTO buecher_exemplare (titel_id, barcode_id) SELECT id, 'B-NORMALFORM-1' FROM t`); err != nil {
		t.Fatal(err)
	}
	repo := NewKatalogAbfragen(pool)
	for _, suchtext := range []string{"978-3-16-148", "9783161484", "978 3 16 148410 0"} {
		treffer, err := repo.FindeTitel(ctx, suchtext)
		if err != nil {
			t.Fatalf("suchen %q: %v", suchtext, err)
		}
		gefunden := false
		for _, tr := range treffer {
			gefunden = gefunden || tr.Titel == "Suchprobe Normalform"
		}
		if !gefunden {
			t.Errorf("Suche nach %q findet den Titel mit ISBN 9783161484100 nicht", suchtext)
		}
	}
}
