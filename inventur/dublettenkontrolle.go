package inventur

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"bibliothek/repository"
)

// istKeineZeile: „nichts gefunden" ist bei einer Suche nach Dubletten das erwünschte
// Ergebnis, nicht der Fehlerfall.
func istKeineZeile(err error) bool { return errors.Is(err, pgx.ErrNoRows) }

// Dublettenkontrolle beim Anlegen und Ändern eines Titels (OFFEN.md 4.18, Stufe 2).
//
// Der UNIQUE-Index auf isbn fängt nur die ZEICHENGLEICHE Dublette. „978-3-12-345678-9"
// und „9783123456789" sind aber dieselbe Nummer; als zwei Titel bedeuten sie zwei
// Bestände, zwei Meldebestände und zwei Zeilen in der Nachbestellung für ein Buch.
//
// Littera prüft an derselben Stelle (Medienaufnahme) und bietet statt eines zweiten
// Titels an, ein weiteres Exemplar anzulegen. Genau das ist hier die Botschaft der
// Fehlermeldung — die Entscheidung trifft ein Mensch, nicht das Programm.
//
// Die Prüfung läuft VOR dem Schreiben in derselben Transaktion. Sie ersetzt keinen
// Constraint: Zwei gleichzeitige Anfragen können beide durchkommen. Der Constraint
// dafür wäre ein UNIQUE-Index auf replace(isbn, '-', ”); er lässt sich erst anlegen,
// wenn am Server gemessen ist, dass es dort keine Altdublette gibt (OFFEN.md 4.18).
func pruefeDublette(ctx context.Context, q repository.DBQueryer, b Book, eigeneID string) error {
	if b.ISBN != "" {
		var vorhandenerTitel string
		err := q.QueryRow(ctx, `
			SELECT titel FROM buecher_titel
			WHERE replace(replace(lower(isbn), '-', ''), ' ', '') = replace(replace(lower($1), '-', ''), ' ', '')
			  AND ($2 = '' OR id <> $2::uuid)
			LIMIT 1`, b.ISBN, eigeneID).Scan(&vorhandenerTitel)
		switch {
		case err == nil:
			return fmt.Errorf("%w: %q trägt dieselbe Nummer in anderer Schreibweise", ErrDuplicateISBN, vorhandenerTitel)
		case istKeineZeile(err):
			// kein Treffer — weiter
		default:
			return fmt.Errorf("dublettenkontrolle (isbn): %w", err)
		}
		return nil
	}

	// Ohne ISBN entscheidet das Paar aus Titel und Autor — wie bei Littera. Die Auflage
	// hebt den Verdacht auf: Wer sie füllt, sagt damit, dass er ein anderes Buch meint
	// (dieselbe Reihe, andere Seitenzahlen).
	if b.Title == "" {
		return nil
	}
	var vorhandeneID string
	err := q.QueryRow(ctx, `
		SELECT id::text FROM buecher_titel
		WHERE isbn IS NULL
		  AND lower(btrim(titel)) = lower(btrim($1))
		  AND lower(btrim(coalesce(autor, ''))) = lower(btrim($2))
		  AND lower(btrim(coalesce(auflage, ''))) = lower(btrim($3))
		  AND ($4 = '' OR id <> $4::uuid)
		LIMIT 1`, b.Title, b.Author, b.Auflage, eigeneID).Scan(&vorhandeneID)
	switch {
	case err == nil:
		return fmt.Errorf("%w: derselbe Titel steht schon ohne Nummer im Katalog", ErrDubletteTitel)
	case istKeineZeile(err):
		return nil
	default:
		return fmt.Errorf("dublettenkontrolle (titel): %w", err)
	}
}
