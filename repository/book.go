package repository

import (
	"bibliothek/db"
	"context"

	"github.com/jackc/pgx/v5"
)

// BookRepository definiert alle Datenbank-Operationen für physische Buchexemplare
// und die Suche nach Buchtiteln im Katalog.
type BookRepository interface {
	// GetCopyByBarcode sucht ein physisches Buchexemplar anhand seines Barcodes.
	// Liefert die Exemplardaten inklusive der Metadaten des verknüpften Buchtitels zurück.
	// Gibt nil zurück, wenn kein Exemplar gefunden wurde.
	GetCopyByBarcode(ctx context.Context, barcode string) (*BookCopy, error)

	// SearchTitles führt eine Volltextsuche (german tsvector) über Titel, Autoren und ISBNs aus
	// und sortiert die Ergebnisse nach ihrer Relevanz (ts_rank).
	SearchTitles(ctx context.Context, queryText string) ([]BookTitle, error)

	// SearchTitlesFuzzy sucht tokenweise über Titel, Autor, ISBN und Signatur mit
	// einer Ergebnismengenbegrenzung. Zweiter Rückgabewert ist die Gesamtzahl der
	// Treffer vor dem Limit (siehe StudentRepository.SearchStudentsFuzzy).
	SearchTitlesFuzzy(ctx context.Context, queryText string, limit int) ([]BookTitle, int, error)

	// GetTitleByIDTx sucht einen Buchtitel anhand seiner UUID, optimiert für Transaktionen.
	GetTitleByIDTx(ctx context.Context, tx pgx.Tx, id string) (*BookTitle, error)

	// UpdateCopyDamageNote aktualisiert die Zustandsnotiz (z. B. Beschädigungen) eines Buchexemplars.
	UpdateCopyDamageNote(ctx context.Context, id string, note string) error

	// UpdateCopyBarcode ändert den zugewiesenen Barcode eines physischen Exemplars.
	UpdateCopyBarcode(ctx context.Context, id string, barcode string) error

	// UpdateCopyStatus ändert den Ausleih- und Aussonderungsstatus eines Exemplars.
	UpdateCopyStatus(ctx context.Context, id string, istAusleihbar bool, istAusgesondert bool, zustandNotiz string) error

	// DecommissionCopy kennzeichnet ein Exemplar als dauerhaft ausgesondert und sperrt die Ausleihe.
	DecommissionCopy(ctx context.Context, id string) error

	// GenerateBarcodes erzeugt eine Serie fortlaufender Buch-Barcodes (Präfix "B-") über eine DB-Sequence.
	GenerateBarcodes(ctx context.Context, count int) ([]string, error)

	// BulkInsertCopiesTx führt BulkInsertCopies innerhalb einer expliziten SQL-Transaktion aus.
	BulkInsertCopiesTx(ctx context.Context, tx pgx.Tx, copies []BookCopyInsert) error

	// BulkUpsertBookTitles speichert viele Titel in einem einzigen gepipelineten
	// Batch (statt je Titel eine eigene Rundreise) und liefert die Zahl der
	// verarbeiteten Titel. Titel ohne ISBN werden per Titel dedupliziert.
	BulkUpsertBookTitles(ctx context.Context, titles []BookTitle) (int, error)
}

// BookCopyInsert beschreibt die Datenstruktur für das Einfügen neuer Buchexemplare im Bulk-Verfahren.
type BookCopyInsert struct {
	// TitelID verweist auf die Metadaten des Buchtitels.
	TitelID string
	// BarcodeID ist der eindeutige Barcode des neuen Exemplars.
	BarcodeID string
	// ZustandNotiz dokumentiert den Initialzustand des Buchs (optional).
	ZustandNotiz string
	// IstAusleihbar gibt an, ob das Buch direkt verliehen werden darf.
	IstAusleihbar bool
	// EtikettGedruckt speichert, ob das Barcode-Etikett bereits gedruckt wurde.
	EtikettGedruckt bool
	// Einkaufspreis speichert den Netto-Anschaffungspreis des Exemplars.
	Einkaufspreis float64
	// BestellungID verweist auf die Bestellung, aus der das Exemplar entstanden ist.
	// Leer bei Handanlage und Import — dann wird NULL geschrieben. Der Verweis trägt die
	// Etikettenseite des Lieferanten-Links: Sie muss genau die Exemplare DIESER Lieferung
	// drucken, nicht alle Exemplare des Titels.
	BestellungID string
}

// pgBookRepository implementiert das BookRepository für PostgreSQL.
type pgBookRepository struct {
	db db.PgxPoolIface
}

// NewBookRepository erstellt eine neue Instanz des PostgreSQL-basierten Book-Repositorys.
func NewBookRepository(db db.PgxPoolIface) BookRepository {
	return &pgBookRepository{db: db}
}

// scanBookCopy ist eine Hilfsfunktion zum Einlesen einer Zeile in ein BookCopy-Objekt.
func scanBookCopy(row Scanner) (*BookCopy, error) {
	var bc BookCopy
	err := row.Scan(
		&bc.ID, &bc.TitelID, &bc.BarcodeID, &bc.ZustandNotiz, &bc.ErworbenAm, &bc.IstAusleihbar, &bc.IstAusgesondert, &bc.ErstelltAm, &bc.AktualisiertAm,
		&bc.Titel, &bc.Autor, &bc.Verlag, &bc.ISBN, &bc.CoverURL, &bc.Medientyp, &bc.Signatur, &bc.ZielJahrgang, &bc.ErweiterteEigenschaften,
	)
	if err != nil {
		return nil, err
	}
	return &bc, nil
}

// scanBookTitle ist eine Hilfsfunktion zum Einlesen einer Zeile in ein BookTitle-Objekt.
func scanBookTitle(row Scanner) (*BookTitle, error) {
	return scanBookTitleMitZusatz(row)
}

// scanBookTitleMitZusatz scannt die Standard-Spaltenliste und danach beliebige
// Zusatzspalten (z. B. count(*) OVER () für die Gesamttrefferzahl) — siehe
// scanStudentMitZusatz, gleiche Begründung.
func scanBookTitleMitZusatz(row Scanner, zusatz ...any) (*BookTitle, error) {
	var t BookTitle
	ziele := []any{
		&t.ID, &t.Titel, &t.Untertitel, &t.Autor, &t.ISBN, &t.Verlag, &t.Erscheinungsjahr, &t.Beschreibung, &t.CoverURL, &t.Medientyp, &t.Signatur, &t.ZielJahrgang, &t.ErstelltAm, &t.AktualisiertAm, &t.ErweiterteEigenschaften,
	}
	if err := row.Scan(append(ziele, zusatz...)...); err != nil {
		return nil, err
	}
	return &t, nil
}
