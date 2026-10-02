package inventur

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"bibliothek/pkg/isbnutil"
	"bibliothek/repository"
)

// istKeineZeile: „nichts gefunden" ist bei einer Suche nach Dubletten das erwünschte
// Ergebnis, nicht der Fehlerfall.
func istKeineZeile(err error) bool { return errors.Is(err, pgx.ErrNoRows) }

// DubletteISBN ist ErrDuplicateISBN mit dem Titel, der die Nummer schon trägt. Ein Titel
// ohne Exemplar steht in keinem Katalog (repository.SQLTitelHatExemplar): Ohne Kennung und
// Ort führte die Ablehnung zu einem Buch, das sich nicht finden lässt.
type DubletteISBN struct {
	ID          string
	Titel       string
	HatExemplar bool
}

func (d *DubletteISBN) Error() string {
	return fmt.Sprintf("%v: %q (%s) trägt dieselbe Nummer", ErrDuplicateISBN, d.Titel, d.ID)
}

// Unwrap hält errors.Is(err, ErrDuplicateISBN) für alle Aufrufer gültig.
func (d *DubletteISBN) Unwrap() error { return ErrDuplicateISBN }

// Meldung ist der Satz für die Oberfläche: der Titel und, hat er kein Exemplar, die Sicht,
// in der er steht. Die Ablehnung beim Speichern und die Auskunft vorab sagen dasselbe.
func (d *DubletteISBN) Meldung() string {
	meldung := "Diese ISBN trägt schon der Titel „" + d.Titel + "“."
	if !d.HatExemplar {
		meldung += " Er hat kein Exemplar und steht deshalb in der Titelliste nur unter „Ohne Exemplare“."
	}
	return meldung
}

// alsAntwort ist der Titel in der Form, in der beide Türen ihn unter „vorhanden" nennen.
func (d *DubletteISBN) alsAntwort() map[string]any {
	return map[string]any{"id": d.ID, "title": d.Titel, "ohneExemplar": !d.HatExemplar}
}

// titelMitISBN sucht den Titel, der die ISBN schon trägt, ohne Rücksicht auf Bindestriche,
// Leerzeichen und Großschreibung; eigeneID nimmt den Titel aus, der gerade geändert wird.
// Ohne Treffer nil. Die Ablehnung beim Speichern und die Auskunft vorab fragen beide hier,
// damit die Maske vorab dasselbe erfährt, was das Speichern sagen wird.
func titelMitISBN(ctx context.Context, q repository.DBQueryer, isbn, eigeneID string) (*DubletteISBN, error) {
	vorhanden := DubletteISBN{}
	err := q.QueryRow(ctx, `
		SELECT bt.id::text, bt.titel, `+repository.SQLTitelHatExemplar("bt")+`
		FROM buecher_titel bt
		WHERE replace(replace(lower(bt.isbn), '-', ''), ' ', '') = replace(replace(lower($1), '-', ''), ' ', '')
		  AND ($2 = '' OR bt.id <> $2::uuid)
		LIMIT 1`, isbn, eigeneID).Scan(&vorhanden.ID, &vorhanden.Titel, &vorhanden.HatExemplar)
	switch {
	case err == nil:
		return &vorhanden, nil
	case istKeineZeile(err):
		return nil, nil
	default:
		return nil, fmt.Errorf("dublettenkontrolle (isbn): %w", err)
	}
}

// TitelMitISBN sagt vor dem Speichern, was die Dublettenkontrolle zu dieser ISBN sagen wird.
func (repo *BookRepository) TitelMitISBN(ctx context.Context, isbn string) (*DubletteISBN, error) {
	return titelMitISBN(ctx, repo.db, isbn, "")
}

// TitelUnterAndererForm sucht den Titel, der dieselbe ISBN in der anderen Länge trägt (ISBN-10
// und ISBN-13 mit 978, isbnutil.AndereForm), und nennt diese Form. Die Normalform trennt beide
// Längen, und der Strichcode auf dem Buch ist dreizehnstellig: Ein Titel aus Littera mit
// zehnstelliger ISBN stünde nach dem Scan sonst ein zweites Mal im Katalog. Er wird nur
// vorgeschlagen und nicht abgelehnt — unter der anderen Form kann ein anderes Buch stehen
// (eine ISBN-10 mit falschem Prüfzeichen).
func (repo *BookRepository) TitelUnterAndererForm(ctx context.Context, isbn string) (*DubletteISBN, string, error) {
	andere := isbnutil.AndereForm(isbn)
	if andere == "" {
		return nil, "", nil
	}
	titel, err := titelMitISBN(ctx, repo.db, andere, "")
	return titel, andere, err
}

// MeldungAndereForm ist der Satz der Maske zu einem Titel unter der anderen Form der ISBN.
func (d *DubletteISBN) MeldungAndereForm(andere string) string {
	laenge := "dreizehnstelliger"
	if len(andere) == 10 {
		laenge = "zehnstelliger"
	}
	meldung := "Im Katalog steht diese ISBN in " + laenge + " Form (" + andere + ") am Titel „" + d.Titel + "“."
	if !d.HatExemplar {
		meldung += " Er hat kein Exemplar und steht deshalb in der Titelliste nur unter „Ohne Exemplare“."
	}
	return meldung
}

// pruefeDublette lehnt einen Titel ab, den es schon gibt (OFFEN.md 4.18, Stufe 2): mit ISBN
// den, der die Nummer trägt, ohne ISBN das Paar aus Titel und Autor. Als zwei Titel hätte ein
// Buch zwei Bestände und zwei Zeilen in der Nachbestellung; Littera bietet an dieser Stelle an,
// statt des Titels ein weiteres Exemplar aufzunehmen.
//
// Die Prüfung läuft vor dem Schreiben in derselben Transaktion und nennt den vorhandenen
// Titel. Zwei gleichzeitige Anfragen mit derselben ISBN fängt der UNIQUE-Index: Die Datenbank
// bringt jede ISBN vor dem Schreiben in eine Schreibweise (isbn_normalform).
func pruefeDublette(ctx context.Context, q repository.DBQueryer, b Book, eigeneID string) error {
	if b.ISBN != "" {
		vorhanden, err := titelMitISBN(ctx, q, b.ISBN, eigeneID)
		if err != nil {
			return err
		}
		// Ein nil-Zeiger in der error-Schnittstelle wäre nicht nil.
		if vorhanden != nil {
			return vorhanden
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
