package repository

import (
	"bibliothek/db"
	"context"

	"github.com/jackc/pgx/v5"
)

// KatalogAbfragen definiert alle Datenbank-Operationen für physische Buchexemplare
// und die Suche nach Buchtiteln im Katalog.
type KatalogAbfragen interface {
	// FindeExemplar sucht ein physisches Buchexemplar anhand seines Barcodes.
	// Liefert die Exemplardaten inklusive der Metadaten des verknüpften Buchtitels zurück.
	// Gibt nil zurück, wenn kein Exemplar gefunden wurde.
	FindeExemplar(ctx context.Context, barcode string) (*Exemplar, error)

	// FindeTitel führt eine Volltextsuche (german tsvector) über Titel, Autoren und ISBNs aus
	// und sortiert die Ergebnisse nach ihrer Relevanz (ts_rank).
	FindeTitel(ctx context.Context, queryText string) ([]Titel, error)

	// FindeTitelUnscharf sucht tokenweise über Titel, Autor, ISBN und Signatur mit
	// einer Ergebnismengenbegrenzung. Zweiter Rückgabewert ist die Gesamtzahl der
	// Treffer vor dem Limit (siehe LeserAbfragen.FindeUnscharf).
	FindeTitelUnscharf(ctx context.Context, queryText string, limit int) ([]Titel, int, error)

	// LadeTitel sucht einen Buchtitel anhand seiner UUID, optimiert für Transaktionen.
	LadeTitel(ctx context.Context, tx pgx.Tx, id string) (*Titel, error)

	// SetzeExemplarBarcode ändert den zugewiesenen Barcode eines physischen Exemplars.
	SetzeExemplarBarcode(ctx context.Context, id string, barcode string) error

	// AendereExemplarStatus ändert den Ausleih- und Aussonderungsstatus eines Exemplars.
	//
	// zustandAbwertungProzent ist der Beschädigungsgrad dieses Exemplars.
	// nil heißt unangetastet, nicht 0: Der Status-Editor schickt das Feld nur mit, wenn ein
	// Mensch es angefasst hat, und ein Statuswechsel darf einen erfassten Wasserschaden
	// nicht stillschweigend auf 0 zurückstellen (Upsert-Blanking).
	//
	// bearbeiterID nennt, wer speichert. Aussondern verlangt sie
	// (ErrAussonderungOhneBearbeiter): Der Wechsel steht mit der Person im Protokoll.
	AendereExemplarStatus(ctx context.Context, id string, istAusleihbar bool, istAusgesondert bool, zustandNotiz string, zustandAbwertungProzent *int, bearbeiterID string) error

	// IstExemplarImBestand sagt, ob ein Exemplar zum Bestand zählt (SQLExemplarImBestand).
	IstExemplarImBestand(ctx context.Context, id string) (bool, error)

	// ErzeugeBarcodes erzeugt eine Serie fortlaufender Buch-Barcodes (Präfix "B-") über eine DB-Sequence.
	ErzeugeBarcodes(ctx context.Context, count int) ([]string, error)

	// LegeExemplareAn führt BulkInsertCopies innerhalb einer expliziten SQL-Transaktion aus.
	LegeExemplareAn(ctx context.Context, tx pgx.Tx, copies []ExemplarNeu) error

	// SchreibeTitel speichert viele Titel in einem einzigen gepipelineten
	// Batch (statt je Titel eine eigene Rundreise) und liefert die Zahl der
	// verarbeiteten Titel. Titel ohne ISBN werden per Titel dedupliziert.
	SchreibeTitel(ctx context.Context, titles []Titel) (int, error)
}

// pgKatalogAbfragen implementiert das KatalogAbfragen für PostgreSQL.
type pgKatalogAbfragen struct {
	db db.PgxPoolIface
}

// NewKatalogAbfragen erstellt eine neue Instanz des PostgreSQL-basierten Book-Repositorys.
func NewKatalogAbfragen(db db.PgxPoolIface) KatalogAbfragen {
	return &pgKatalogAbfragen{db: db}
}
