package repository

import (
	"context"
	"encoding/json"
	"os"
	"slices"
	"testing"
)

// Texte ordnen an drei Stellen: in der Datenbank mit der Kollation deutsch (Migration 165), in
// der Titelliste mit sortiereBuecherNachTitel (Go) und in den Listen der Oberfläche mit
// Intl.Collator('de', {numeric: true}). Alle drei lesen dieselben Fälle; ordnete eine Seite
// anders, stünde derselbe Titel in einer Liste des Servers woanders als im Browser.
func TestSortierung_DatenbankOrdnetWieDieGemeinsamenFaelle(t *testing.T) {
	pool := pgTestPool(t)
	ctx := context.Background()

	roh, err := os.ReadFile("../frontend/src/lib/utils/titelReihenfolge.faelle.json")
	if err != nil {
		t.Fatal(err)
	}
	var pruefung struct {
		Faelle []struct {
			Fall     string   `json:"fall"`
			Erwartet []string `json:"erwartet"`
		} `json:"faelle"`
	}
	if err := json.Unmarshal(roh, &pruefung); err != nil {
		t.Fatal(err)
	}
	if len(pruefung.Faelle) < 10 {
		t.Fatalf("len(pruefung.Faelle) = %d, erwartet mindestens 10: liest der Test noch titelReihenfolge.faelle.json?", len(pruefung.Faelle))
	}

	for _, f := range pruefung.Faelle {
		// Umgekehrt hinein: Eine Datenbank, die nicht ordnet, gäbe die Eingabe zurück.
		eingabe := slices.Clone(f.Erwartet)
		slices.Reverse(eingabe)
		var geordnet []string
		if err := pool.QueryRow(ctx,
			`SELECT array_agg(w ORDER BY w COLLATE deutsch) FROM unnest($1::text[]) w`, eingabe).
			Scan(&geordnet); err != nil {
			t.Fatalf("%s: %v", f.Fall, err)
		}
		if !slices.Equal(geordnet, f.Erwartet) {
			t.Errorf("%s: ORDER BY … COLLATE deutsch = %q, erwartet %q", f.Fall, geordnet, f.Erwartet)
		}
	}

	// Kein Index benutzt die Kollation. Bringt ein neues Image eine andere ICU-Fassung, ändert
	// sich so nur die Reihenfolge einer Ausgabe und nie die Ordnung eines gespeicherten Index.
	var indizes int
	if err := pool.QueryRow(ctx, `
		SELECT count(*) FROM pg_index i JOIN pg_collation c ON c.oid = ANY(i.indcollation::oid[])
		WHERE c.collname = 'deutsch'`).Scan(&indizes); err != nil {
		t.Fatal(err)
	}
	if indizes != 0 {
		t.Errorf("Indizes mit der Kollation deutsch: %d, erwartet 0", indizes)
	}
}
