package inventur

import (
	"context"
	"testing"

	"bibliothek/internal/pgtest"
)

// Eine Liste ohne Klasse schreibt „unbekannt" (NULL), nicht die Vorgabe 5 — dieselbe Regel
// wie Littera-Übernahme und Sammelimport. Und weil das Upsert eine vorhandene Klasse
// ungleich 0 behält, muss eine spätere Liste MIT Klasse die Lücke noch füllen können: Mit
// der Vorgabe 5 bis zum 22.09.2026 kam sie gegen die geratene 5 nicht mehr an.
func TestListenimport_UnbekannteKlasseBleibtNullUndWirdNachgetragen(t *testing.T) {
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

		klasse := func() *int16 {
			t.Helper()
			var k *int16
			if err := pool.QueryRow(ctx, `SELECT grade_level FROM buecher_titel WHERE isbn = isbn_normalform($1)`, isbn).Scan(&k); err != nil {
				t.Fatalf("%s: %v", weg, err)
			}
			return k
		}

		ohneSpalte, err := verarbeiteImportZeile(ImportConfig{
			Ctx:       ctx,
			Row:       []string{isbn, "Die 13½ Leben des Käpt'n Blaubär", "Moers"},
			ColIdx:    map[string]int{"isbn": 0, "titel": 1, "autor": 2, "fach": -1, "klasse": -1, "bestand": -1},
			Metadaten: offlineMetadatenClient(),
		})
		if err != nil || ohneSpalte == nil {
			t.Fatalf("%s: Zeile: %v", weg, err)
		}
		if err := importiere(*ohneSpalte); err != nil {
			t.Fatalf("%s: erster Import: %v", weg, err)
		}
		if k := klasse(); k != nil {
			t.Fatalf("%s: Klasse %d nach einer Liste ohne Klasse, erwartet NULL", weg, *k)
		}

		mitSpalte := *ohneSpalte
		mitSpalte.GradeLevel = parseKlassenStufe("8")
		if err := importiere(mitSpalte); err != nil {
			t.Fatalf("%s: zweiter Import: %v", weg, err)
		}
		if k := klasse(); k == nil {
			t.Errorf("%s: Klasse NULL nach einer Liste mit Klasse 8, erwartet 8", weg)
		} else if *k != 8 {
			t.Errorf("%s: Klasse %d nach einer Liste mit Klasse 8, erwartet 8", weg, *k)
		}
	}
}
