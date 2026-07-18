package db

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
)

// TestSearchVector_IncludesIsbnAndBeschreibung sichert Migration 050 ab: ISBN und
// Beschreibung müssen über den Volltext-Suchvektor auffindbar sein — vorher deckte er nur
// titel/untertitel/autor/verlag ab.
func TestSearchVector_IncludesIsbnAndBeschreibung(t *testing.T) {
	pool := pgTestPool(t)
	ctx := context.Background()

	inTx(t, pool, func(tx pgx.Tx) {
		if _, err := tx.Exec(ctx,
			`INSERT INTO buecher_titel (titel, isbn, beschreibung)
			 VALUES ('Belangloser Titel', '9783161484100', 'Ein Zauberwald voller Drachen')`); err != nil {
			t.Fatalf("Titel anlegen: %v", err)
		}

		treffer := func(term string) int {
			t.Helper()
			var n int
			if err := tx.QueryRow(ctx,
				`SELECT count(*) FROM buecher_titel WHERE search_vector @@ plainto_tsquery('german', $1)`,
				term).Scan(&n); err != nil {
				t.Fatalf("Suche %q: %v", term, err)
			}
			return n
		}

		if treffer("Zauberwald") != 1 {
			t.Error("Stichwort aus der Beschreibung ist nicht über den search_vector auffindbar")
		}
		if treffer("9783161484100") != 1 {
			t.Error("ISBN ist nicht über den search_vector auffindbar")
		}
	})
}
