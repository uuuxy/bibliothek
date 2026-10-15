package inventur

import (
	"context"
	"errors"
	"testing"

	"bibliothek/internal/pgtest"
)

// Eine doppelte ISBN ist eine Auskunft („gibt es schon"), keine Störung — und zwar am
// ECHTEN Constraint, nicht an einem nachgespielten.
//
// handleDbError erkannte die Verletzung bis zum 13.09.2026 am Namen `books_isbn_key`.
// So hieß der Constraint, als die Tabelle noch `books` hieß; heute heißt sie
// `buecher_titel` und der Constraint `buecher_titel_isbn_key` (auf dem Server
// nachgesehen). Die Übersetzung griff damit nie: Wer ein Buch mit vorhandener ISBN
// anlegte, las „buch konnte nicht erstellt werden" statt „existiert bereits". Der
// Unit-Test daneben blieb grün, weil er denselben alten Namen nachspielte.
func TestISBNDublette_WirdAlsDubletteErkannt(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	repo := NewTitelAbfragen(pool)

	const isbn = "9780000131313"
	const andere = "9780000131320"
	t.Cleanup(func() {
		if _, err := pool.Exec(context.Background(),
			`DELETE FROM buecher_titel WHERE isbn IN ($1, $2)`, isbn, andere); err != nil {
			t.Errorf("aufräumen: %v", err)
		}
	})

	if _, err := repo.LegeTitelAn(ctx, Titel{ISBN: isbn, Titel: "Erstes Exemplar der ISBN", Autor: "Test"}); err != nil {
		t.Fatalf("erstes Anlegen: %v", err)
	}

	t.Run("Anlegen", func(t *testing.T) {
		_, err := repo.LegeTitelAn(ctx, Titel{ISBN: isbn, Titel: "Dublette", Autor: "Test"})
		if !errors.Is(err, ErrISBNDoppelt) {
			t.Fatalf("erwartet ErrDuplicateISBN, bekam %v", err)
		}
	})

	t.Run("Bearbeiten", func(t *testing.T) {
		zweiter, err := repo.LegeTitelAn(ctx, Titel{ISBN: andere, Titel: "Zweiter Titel", Autor: "Test"})
		if err != nil {
			t.Fatalf("zweiten Titel anlegen: %v", err)
		}
		err = repo.AendereTitel(ctx, zweiter, Titel{ISBN: isbn, Titel: "Zweiter Titel", Autor: "Test"}, []string{"isbn", "title", "author"}, nil)
		if !errors.Is(err, ErrISBNDoppelt) {
			t.Fatalf("erwartet ErrDuplicateISBN, bekam %v", err)
		}
	})
}
