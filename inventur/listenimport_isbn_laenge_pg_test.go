package inventur

import (
	"context"
	"testing"

	"bibliothek/internal/pgtest"
)

// Der Listenimport gleicht über ON CONFLICT (isbn) ab. Trägt der Katalog ein Buch und nennt
// die Liste seine ISBN in der anderen Länge, kommen die Exemplare an den vorhandenen Titel:
// Die Datenbank führt beide Längen als eine Nummer (Migration 157). Als zwei Titel hätte das
// Buch zwei Bestände und zwei Zeilen in der Nachbestellung.
func TestListenimport_BeideLaengenDerISBNSindEinTitel(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	handler := &APIHandler{repo: NewBookRepository(pool)}

	// stand zählt Titel und Exemplare zu einer ISBN, in welcher Länge sie auch gespeichert ist.
	stand := func(isbn string) (titel, exemplare int) {
		t.Helper()
		if err := pool.QueryRow(ctx, `
			SELECT count(DISTINCT t.id)::int, count(e.id)::int
			FROM buecher_titel t LEFT JOIN buecher_exemplare e ON e.titel_id = t.id
			WHERE isbn_normalform(t.isbn) = isbn_normalform($1)`, isbn).Scan(&titel, &exemplare); err != nil {
			t.Fatal(err)
		}
		return titel, exemplare
	}
	loesche := func(isbns ...string) {
		for _, sql := range []string{
			`DELETE FROM buecher_exemplare WHERE titel_id IN (SELECT id FROM buecher_titel WHERE isbn = ANY($1))`,
			`DELETE FROM buecher_titel WHERE isbn = ANY($1)`,
		} {
			if _, err := pool.Exec(ctx, sql, isbns); err != nil {
				t.Logf("Aufräumen: %v", err)
			}
		}
	}

	t.Run("der Katalog trägt die dreizehnstellige, die Liste nennt die zehnstellige", func(t *testing.T) {
		const zehn, dreizehn = "3551551677", "9783551551672"
		loesche(zehn, dreizehn)
		t.Cleanup(func() { loesche(zehn, dreizehn) })
		if _, err := pool.Exec(ctx,
			`INSERT INTO buecher_titel (titel, autor, isbn) VALUES ('Harry Potter und der Stein der Weisen', 'Rowling', $1)`,
			dreizehn); err != nil {
			t.Fatalf("Titel anlegen: %v", err)
		}

		importiert, gescheitert, err := handler.persistImportedBooks(ctx,
			[]Book{{ISBN: "3-551-55167-7", Title: "Harry Potter 1", Stock: 2}}, 0, nil)
		if err != nil || importiert != 1 || gescheitert != 0 {
			t.Fatalf("Listenimport: %d importiert, %d gescheitert, %v", importiert, gescheitert, err)
		}
		if titel, exemplare := stand(dreizehn); titel != 1 || exemplare != 2 {
			t.Errorf("nach dem Listenimport: %d Titel mit %d Exemplaren, erwartet 1 Titel mit 2", titel, exemplare)
		}
	})

	// Zwei Zeilen einer Liste, die auf denselben Titel führen, lehnt Postgres im Batch ab
	// (21000). Der Import fällt dann auf den Einzelweg zurück, Zeile für Zeile.
	t.Run("die Liste nennt dasselbe Buch in beiden Längen", func(t *testing.T) {
		const zehn, dreizehn = "0306406152", "9780306406157"
		loesche(zehn, dreizehn)
		t.Cleanup(func() { loesche(zehn, dreizehn) })

		importiert, gescheitert, err := handler.persistImportedBooks(ctx, []Book{
			{ISBN: dreizehn, Title: "Advanced Organic Chemistry", Stock: 2},
			{ISBN: zehn, Title: "Advanced Organic Chemistry", Stock: 3},
		}, 0, nil)
		if err != nil || importiert != 2 || gescheitert != 0 {
			t.Fatalf("Listenimport: %d importiert, %d gescheitert, %v", importiert, gescheitert, err)
		}
		if titel, exemplare := stand(dreizehn); titel != 1 || exemplare != 5 {
			t.Errorf("nach dem Listenimport: %d Titel mit %d Exemplaren, erwartet 1 Titel mit 5", titel, exemplare)
		}
	})

	// Eine zehnstellige Nummer mit falschem Prüfzeichen ist keine andere Schreibweise der
	// dreizehnstelligen: Am Testserver steht unter 3499500252 ein anderes Buch als unter
	// 9783499500251.
	t.Run("falsches Prüfzeichen bleibt ein eigener Titel", func(t *testing.T) {
		const falsch, dreizehn = "3499500252", "9783499500251"
		loesche(falsch, dreizehn)
		t.Cleanup(func() { loesche(falsch, dreizehn) })

		importiert, gescheitert, err := handler.persistImportedBooks(ctx, []Book{
			{ISBN: dreizehn, Title: "Das eine Buch", Stock: 1},
			{ISBN: falsch, Title: "Das andere Buch", Stock: 1},
		}, 0, nil)
		if err != nil || importiert != 2 || gescheitert != 0 {
			t.Fatalf("Listenimport: %d importiert, %d gescheitert, %v", importiert, gescheitert, err)
		}
		var titel int
		if err := pool.QueryRow(ctx, `SELECT count(*)::int FROM buecher_titel WHERE isbn = ANY($1)`,
			[]string{falsch, dreizehn}).Scan(&titel); err != nil {
			t.Fatal(err)
		}
		if titel != 2 {
			t.Errorf("%d Titel, erwartet 2: je einer unter %s und %s", titel, falsch, dreizehn)
		}
	})
}
