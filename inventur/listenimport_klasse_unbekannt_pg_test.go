package inventur

import (
	"context"
	"testing"

	"bibliothek/internal/pgtest"
)

// Eine Liste ohne die Spalte „klasse" lässt den Jahrgang unbekannt (NULL), auch wenn der Titel
// eine Zahl trägt. Eine spätere Liste mit der Spalte füllt die Lücke: Das Upsert behält nur
// eine eingetragene Spanne, und eine geratene stünde der echten im Weg.
func TestListenimport_OhneKlasseUnbekanntUndSpaeterNachgetragen(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	repo := NewBookRepository(pool)

	wege := map[string]func(Book) error{
		"UpsertBooksBatch": func(b Book) error {
			_, err := repo.UpsertBooksBatch(ctx, []Book{b})
			return err
		},
		"UpsertBook (Einzel-Rückfall)": func(b Book) error {
			_, err := repo.UpsertBook(ctx, b)
			return err
		},
	}
	isbns := map[string]string{"UpsertBooksBatch": "978-9-99-300001-1", "UpsertBook (Einzel-Rückfall)": "978-9-99-300001-2"}

	for weg, importiere := range wege {
		isbn := isbns[weg]
		t.Cleanup(func() {
			for _, sql := range []string{
				`DELETE FROM buecher_exemplare WHERE titel_id IN (SELECT id FROM buecher_titel WHERE isbn = isbn_normalform($1))`,
				`DELETE FROM buecher_titel WHERE isbn = isbn_normalform($1)`,
			} {
				if _, err := pool.Exec(context.Background(), sql, isbn); err != nil {
					t.Logf("Aufräumen: %v", err)
				}
			}
		})

		spanne := func() string {
			t.Helper()
			var s string
			if err := pool.QueryRow(ctx, `
				SELECT coalesce(jahrgang_von::text, 'leer') || ' bis ' || coalesce(jahrgang_bis::text, 'leer')
				FROM buecher_titel WHERE isbn = isbn_normalform($1)`, isbn).Scan(&s); err != nil {
				t.Fatalf("%s: %v", weg, err)
			}
			return s
		}
		zeile := func(spalten map[string]int, werte ...string) Book {
			t.Helper()
			buch, err := verarbeiteImportZeile(ImportConfig{
				Ctx:       ctx,
				Row:       append([]string{isbn, "Die 13½ Leben des Käpt'n Blaubär", "Moers"}, werte...),
				ColIdx:    spalten,
				Metadaten: offlineMetadatenClient(),
			})
			if err != nil || buch == nil {
				t.Fatalf("%s: Zeile: %v", weg, err)
			}
			return *buch
		}

		ohneSpalte := map[string]int{"isbn": 0, "titel": 1, "autor": 2, "fach": -1, "klasse": -1, "bestand": -1}
		if err := importiere(zeile(ohneSpalte)); err != nil {
			t.Fatalf("%s: erster Import: %v", weg, err)
		}
		if ist := spanne(); ist != "leer bis leer" {
			t.Fatalf("%s: nach einer Liste ohne Klasse steht %s, erwartet leer bis leer", weg, ist)
		}

		mitSpalte := map[string]int{"isbn": 0, "titel": 1, "autor": 2, "fach": -1, "klasse": 3, "bestand": -1}
		if err := importiere(zeile(mitSpalte, "8")); err != nil {
			t.Fatalf("%s: zweiter Import: %v", weg, err)
		}
		if ist := spanne(); ist != "8 bis 8" {
			t.Errorf("%s: nach einer Liste mit Klasse 8 steht %s, erwartet 8 bis 8", weg, ist)
		}
	}
}
