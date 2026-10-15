package repository

import (
	"context"

	"bibliothek/db"

	"github.com/jackc/pgx/v5"
)

// KatalogAbfragen liest und schreibt Titel und Exemplare: die Suche, das Anlegen und den Status
// eines Exemplars.
type KatalogAbfragen interface {
	// FindeExemplar liefert das Exemplar zu einem Barcode mit den Angaben seines Titels; nil,
	// wenn keines die Nummer trägt.
	FindeExemplar(ctx context.Context, barcode string) (*Exemplar, error)

	// FindeTitel sucht über Titel, Autor und ISBN, als Volltext und als Teil des Texts, und
	// ordnet die Treffer nach ihrem Rang; höchstens 50, nur Titel mit einem Exemplar.
	FindeTitel(ctx context.Context, suchtext string) ([]Titel, error)

	// FindeTitelUnscharf sucht Wort für Wort über Titel, Autor, ISBN und Signatur, höchstens
	// limit Treffer. Der zweite Wert ist die Zahl aller Treffer vor dem Limit (wie
	// LeserAbfragen.FindeUnscharf).
	FindeTitelUnscharf(ctx context.Context, suchtext string, limit int) ([]Titel, int, error)

	// LadeTitel liefert Titel, Autor, ISBN, Verlag und Signatur eines Titels, gelesen in der
	// Transaktion des Rufers; die übrigen Felder bleiben leer. Kein Treffer ist pgx.ErrNoRows.
	LadeTitel(ctx context.Context, tx pgx.Tx, id string) (*Titel, error)

	// SetzeExemplarBarcode gibt einem Exemplar einen anderen Barcode.
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

	// ErzeugeBarcodes zieht anzahl fortlaufende Buch-Barcodes (Präfix "B-") aus der Folge der
	// Datenbank.
	ErzeugeBarcodes(ctx context.Context, anzahl int) ([]string, error)

	// LegeExemplareAn legt die Exemplare in der Transaktion des Rufers an, alle in einem Zug.
	LegeExemplareAn(ctx context.Context, tx pgx.Tx, exemplare []ExemplarNeu) error

	// SchreibeTitel legt viele Titel an oder ändert sie, in einem Sammelauftrag statt mit einer
	// Anfrage je Titel, und liefert die Zahl der verarbeiteten Titel. Einen Titel ohne ISBN
	// ordnet die Funktion über seinen Titel zu.
	SchreibeTitel(ctx context.Context, titel []Titel) (int, error)
}

type pgKatalogAbfragen struct {
	db db.Pool
}

// NewKatalogAbfragen liefert die Abfragen des Katalogs an Postgres.
func NewKatalogAbfragen(db db.Pool) KatalogAbfragen {
	return &pgKatalogAbfragen{db: db}
}
