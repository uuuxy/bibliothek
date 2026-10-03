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
	return "Diese ISBN trägt schon der Titel „" + d.Titel + "“." + hinweisOhneExemplar(d.HatExemplar)
}

// alsAntwort ist der Titel in der Form, in der beide Türen ihn unter „vorhanden" nennen.
func (d *DubletteISBN) alsAntwort() map[string]any {
	return map[string]any{"id": d.ID, "title": d.Titel, "ohneExemplar": !d.HatExemplar}
}

// hinweisOhneExemplar nennt die Sicht, in der ein Titel ohne Exemplar steht: Keine Suche
// zeigt ihn.
func hinweisOhneExemplar(hatExemplar bool) string {
	if hatExemplar {
		return ""
	}
	return " Er hat kein Exemplar und steht deshalb in der Titelliste nur unter „Ohne Exemplare“."
}

// DubletteTitel ist ErrDubletteTitel mit dem Titel, der ohne ISBN gleich heißt. Anders als
// eine vergebene ISBN ist das eine Frage und keine Ablehnung: Hefte einer Zeitschrift und
// Bände eines Werks tragen denselben Titel und Autor.
type DubletteTitel DubletteISBN

func (d *DubletteTitel) Error() string {
	return fmt.Sprintf("%v: %q (%s) heißt gleich", ErrDubletteTitel, d.Titel, d.ID)
}

// Unwrap hält errors.Is(err, ErrDubletteTitel) für alle Aufrufer gültig.
func (d *DubletteTitel) Unwrap() error { return ErrDubletteTitel }

// Meldung ist der Satz, mit dem die Maske fragt, ob es dasselbe Medium ist.
func (d *DubletteTitel) Meldung() string {
	return "Ohne ISBN steht schon ein Titel „" + d.Titel + "“ im Katalog." + hinweisOhneExemplar(d.HatExemplar)
}

func (d *DubletteTitel) alsAntwort() map[string]any { return (*DubletteISBN)(d).alsAntwort() }

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
	return "Im Katalog steht diese ISBN in " + laenge + " Form (" + andere + ") am Titel „" + d.Titel + "“." +
		hinweisOhneExemplar(d.HatExemplar)
}

// isbnVergeben lehnt eine ISBN ab, die schon ein anderer Titel trägt, und nennt ihn: Als zwei
// Titel hätte ein Buch zwei Bestände und zwei Zeilen in der Nachbestellung. Die Prüfung läuft
// vor dem Schreiben in derselben Transaktion; zwei gleichzeitige Anfragen fängt der
// UNIQUE-Index, weil die Datenbank jede ISBN in eine Schreibweise bringt (isbn_normalform).
func isbnVergeben(ctx context.Context, q repository.DBQueryer, isbn, eigeneID string) error {
	vorhanden, err := titelMitISBN(ctx, q, isbn, eigeneID)
	if err != nil {
		return err
	}
	// Ein nil-Zeiger in der error-Schnittstelle wäre nicht nil.
	if vorhanden != nil {
		return vorhanden
	}
	return nil
}

// gleichnamigOhneISBN nennt den Titel, der ohne ISBN in Titel, Autor und Auflage gleich ist —
// wie Littera, das ohne Nummer Verfasser und Haupttitel vergleicht und ein weiteres Exemplar
// anbietet. Wer die Auflage füllt, meint ein anderes Buch. Von mehreren gleichnamigen kommt
// zuerst einer mit Exemplar: Den zeigt auch der Katalog.
func gleichnamigOhneISBN(ctx context.Context, q repository.DBQueryer, b Book) error {
	vorhanden := DubletteTitel{}
	hatExemplar := repository.SQLTitelHatExemplar("bt")
	err := q.QueryRow(ctx, `
		SELECT bt.id::text, bt.titel, `+hatExemplar+`
		FROM buecher_titel bt
		WHERE bt.isbn IS NULL
		  AND lower(btrim(bt.titel)) = lower(btrim($1))
		  AND lower(btrim(coalesce(bt.autor, ''))) = lower(btrim($2))
		  AND lower(btrim(coalesce(bt.auflage, ''))) = lower(btrim($3))
		ORDER BY `+hatExemplar+` DESC, bt.sort_order
		LIMIT 1`, b.Title, b.Author, b.Auflage).Scan(&vorhanden.ID, &vorhanden.Titel, &vorhanden.HatExemplar)
	switch {
	case err == nil:
		return &vorhanden
	case istKeineZeile(err):
		return nil
	default:
		return fmt.Errorf("dublettenkontrolle (titel): %w", err)
	}
}

// pruefeNeuenTitel ist die Dublettenkontrolle der Aufnahme: mit ISBN der Titel, der die
// Nummer trägt, ohne ISBN der gleichnamige. anderesMedium ist die Antwort der Maske auf die
// Frage nach dem gleichnamigen und übergeht nur diese.
func pruefeNeuenTitel(ctx context.Context, q repository.DBQueryer, b Book, anderesMedium bool) error {
	if b.ISBN != "" {
		return isbnVergeben(ctx, q, b.ISBN, "")
	}
	if anderesMedium {
		return nil
	}
	return gleichnamigOhneISBN(ctx, q, b)
}
