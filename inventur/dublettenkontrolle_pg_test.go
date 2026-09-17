package inventur

import (
	"context"
	"errors"
	"testing"

	"bibliothek/internal/pgtest"
)

// Dublettenkontrolle beim Anlegen (OFFEN.md 4.18, Stufe 2) — nach dem Vorbild von
// Littera: Dort wird bei der Medienaufnahme über die ISBN geprüft (ohne ISBN über
// Verfasser und Haupttitel), und statt eines zweiten Titels bietet das Programm an,
// ein weiteres Exemplar anzulegen.
//
// Der UNIQUE-Index auf isbn fängt nur ZEICHENGLEICHE Dubletten: „978-3-12-345678-9" und
// „9783123456789" sind dieselbe Nummer und standen bis zum 17.09.2026 als zwei Titel
// nebeneinander — zwei Bestände, zwei Meldebestände, zwei Zeilen in der Nachbestellung
// (OFFEN.md 5.5, 5.12).
//
// Ohne ISBN entscheidet das Paar aus Titel und Autor. Eine ZWEITE AUFLAGE ist erlaubt:
// Seit Stufe 1 gibt es dafür ein Feld, und wer es füllt, sagt damit, dass er ein anderes
// Buch meint.
func TestDublettenkontrolle_BeimAnlegen(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	repo := NewBookRepository(pool)

	aufraeumen := func(t *testing.T, titel string) {
		t.Helper()
		t.Cleanup(func() {
			if _, err := pool.Exec(context.Background(), `DELETE FROM buecher_titel WHERE titel = $1`, titel); err != nil {
				t.Logf("Aufräumen: %v", err)
			}
		})
	}

	t.Run("dieselbe ISBN in anderer Schreibweise", func(t *testing.T) {
		const titel = "Dubletten-Probe ISBN"
		aufraeumen(t, titel)
		if _, err := pool.Exec(ctx, `DELETE FROM buecher_titel WHERE titel = $1`, titel); err != nil {
			t.Fatal(err)
		}
		if _, err := repo.CreateBook(ctx, Book{ISBN: "978-3-12-345678-9", Title: titel}); err != nil {
			t.Fatalf("erster Titel: %v", err)
		}
		_, err := repo.CreateBook(ctx, Book{ISBN: "9783123456789", Title: titel + " (zweite Schreibweise)"})
		if !errors.Is(err, ErrDuplicateISBN) {
			t.Errorf("zweite Schreibweise: Fehler %v, erwartet ErrDuplicateISBN — es ist dieselbe Nummer", err)
		}
	})

	t.Run("ohne ISBN entscheidet Titel und Autor", func(t *testing.T) {
		const titel = "Dubletten-Probe ohne Nummer"
		aufraeumen(t, titel)
		if _, err := pool.Exec(ctx, `DELETE FROM buecher_titel WHERE titel = $1`, titel); err != nil {
			t.Fatal(err)
		}
		if _, err := repo.CreateBook(ctx, Book{Title: titel, Author: "Meyer"}); err != nil {
			t.Fatalf("erster Titel: %v", err)
		}
		_, err := repo.CreateBook(ctx, Book{Title: titel, Author: "Meyer"})
		if !errors.Is(err, ErrDubletteTitel) {
			t.Errorf("zweiter gleicher Titel: Fehler %v, erwartet ErrDubletteTitel", err)
		}

		// Eine andere Auflage ist ein anderes Buch und darf angelegt werden.
		if _, err := repo.CreateBook(ctx, Book{Title: titel, Author: "Meyer", Auflage: "2. Aufl."}); err != nil {
			t.Errorf("zweite Auflage: %v — mit eigener Auflage ist es ein anderes Buch", err)
		}
	})
}
