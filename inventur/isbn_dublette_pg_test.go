package inventur

import (
	"context"
	"errors"
	"testing"

	"bibliothek/internal/pgtest"
)

// Eine doppelte ISBN ist eine Auskunft („gibt es schon"), keine Störung, und zwar am echten
// Constraint, nicht an einem nachgespielten.
//
// uebersetzeDatenbankfehler erkennt die Verletzung am Namen `buecher_titel_isbn_key` (auf dem
// Server nachgesehen). Spielte ein Test einen veralteten Namen nach (`books_isbn_key` aus der Zeit,
// als die Tabelle `books` hieß), bliebe er grün, während die Übersetzung nie griffe und, wer einen
// Titel mit vorhandener ISBN anlegt, „Der Titel ließ sich nicht anlegen" läse statt des Titels,
// der die ISBN schon trägt.
func TestISBNDublette_WirdAlsDubletteErkannt(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	abfragen := NewTitelAbfragen(pool)

	const isbn = "9780000131313"
	const andere = "9780000131320"
	t.Cleanup(func() {
		if _, err := pool.Exec(context.Background(),
			`DELETE FROM buecher_titel WHERE isbn IN ($1, $2)`, isbn, andere); err != nil {
			t.Errorf("aufräumen: %v", err)
		}
	})

	if _, err := abfragen.LegeTitelAn(ctx, Titel{ISBN: isbn, Titel: "Erstes Exemplar der ISBN", Autor: "Test"}); err != nil {
		t.Fatalf("erstes Anlegen: %v", err)
	}

	t.Run("Anlegen", func(t *testing.T) {
		_, err := abfragen.LegeTitelAn(ctx, Titel{ISBN: isbn, Titel: "Dublette", Autor: "Test"})
		if !errors.Is(err, ErrISBNDoppelt) {
			t.Fatalf("erwartet ErrDuplicateISBN, bekam %v", err)
		}
	})

	t.Run("Bearbeiten", func(t *testing.T) {
		zweiter, err := abfragen.LegeTitelAn(ctx, Titel{ISBN: andere, Titel: "Zweiter Titel", Autor: "Test"})
		if err != nil {
			t.Fatalf("zweiten Titel anlegen: %v", err)
		}
		err = abfragen.AendereTitel(ctx, zweiter, Titel{ISBN: isbn, Titel: "Zweiter Titel", Autor: "Test"}, []string{"isbn", "title", "author"}, nil)
		if !errors.Is(err, ErrISBNDoppelt) {
			t.Fatalf("erwartet ErrDuplicateISBN, bekam %v", err)
		}
	})
}
