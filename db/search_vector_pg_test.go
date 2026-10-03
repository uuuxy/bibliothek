package db

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
)

// TestSearchVector_UmfasstTitelangabenUndISBN sichert den Volltext-Suchvektor nach Migration
// 156 ab: Titel, Untertitel, Autor, Verlag und ISBN sind über ihn auffindbar.
func TestSearchVector_UmfasstTitelangabenUndISBN(t *testing.T) {
	pool := pgTestPool(t)
	ctx := context.Background()

	inTx(t, pool, func(tx pgx.Tx) {
		if _, err := tx.Exec(ctx,
			`INSERT INTO buecher_titel (titel, untertitel, autor, verlag, isbn)
			 VALUES ('Quarzwaldgeflecht', 'Nebelzinnenpfad', 'Federkielmann', 'Lindenhofverlag', '9783161484100')`); err != nil {
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

		for spalte, wort := range map[string]string{
			"Titel": "Quarzwaldgeflecht", "Untertitel": "Nebelzinnenpfad", "Autor": "Federkielmann",
			"Verlag": "Lindenhofverlag", "ISBN": "9783161484100",
		} {
			if treffer(wort) != 1 {
				t.Errorf("%s ist nicht über den search_vector auffindbar (%q)", spalte, wort)
			}
		}
	})
}
