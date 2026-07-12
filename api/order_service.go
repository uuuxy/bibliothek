package api

import (
	"context"
	"errors"
	"fmt"

	"bibliothek/db"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5"
)

// OrderService verarbeitet die Geschäftslogik zum Erstellen und Verarbeiten von Bestellungen.
type OrderService struct {
	db           *db.Database
	bookRepo     repository.BookRepository
	supplierRepo repository.SupplierRepository
}

// NewOrderService erstellt eine neue OrderService-Instanz.
func NewOrderService(database *db.Database, bookRepo repository.BookRepository) *OrderService {
	return &OrderService{
		db:           database,
		bookRepo:     bookRepo,
		supplierRepo: repository.NewSupplierRepository(database.Pool),
	}
}

// OrderResult enthält das Ergebnis einer verarbeiteten Bestellung, einschließlich der generierten Barcodes.
type OrderResult struct {
	SupplierName   string
	SupplierEmail  string
	CustomerNumber string
	Labels         []BarcodeLabelDetail
	SummaryItems   []OrderedItem
	TotalAllocated int
}

type bestellungPosition struct {
	titelID   string
	titelName string
	isbn      string
	menge     int
	preis     float64
}

// bestellItemResult bündelt die aus einer einzelnen Bestellposition erzeugten Daten.
type bestellItemResult struct {
	summary  OrderedItem
	position bestellungPosition
	copies   []repository.BookCopyInsert
	labels   []BarcodeLabelDetail
	betrag   float64
}

// ProcessOrder verarbeitet eine eingehende SubmitOrderRequest innerhalb einer Transaktion, generiert Barcodes und gibt das OrderResult zurück.
func (s *OrderService) ProcessOrder(ctx context.Context, req SubmitOrderRequest) (*OrderResult, error) {
	// 1. Lieferantendetails abrufen
	supplier, err := s.supplierRepo.GetSupplierByID(ctx, req.SupplierID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("supplier not found")
		}
		return nil, err
	}

	tx, err := s.db.Pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer db.SafeRollback(ctx, tx)

	labels := make([]BarcodeLabelDetail, 0)
	orderSummaryItems := make([]OrderedItem, 0)
	var totalAllocated int
	var gesamtbetrag float64

	var copyInserts []repository.BookCopyInsert
	var positionen []bestellungPosition

	for _, item := range req.Items {
		res, err := s.verarbeiteBestellItem(ctx, tx, item, supplier.Name)
		if err != nil {
			return nil, err
		}
		orderSummaryItems = append(orderSummaryItems, res.summary)
		positionen = append(positionen, res.position)
		copyInserts = append(copyInserts, res.copies...)
		labels = append(labels, res.labels...)
		gesamtbetrag += res.betrag
		totalAllocated += res.position.menge
	}

	if err := s.bookRepo.BulkInsertCopiesTx(ctx, tx, copyInserts); err != nil {
		return nil, fmt.Errorf("bulk insert error: %w", err)
	}

	// Bestellverlauf + Positionen in derselben Transaktion mitschreiben
	bestellungID, err := s.insertBestellverlauf(ctx, tx, req, supplier, gesamtbetrag, totalAllocated)
	if err != nil {
		return nil, err
	}
	if err := s.insertBestellpositionen(ctx, tx, bestellungID, positionen); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return &OrderResult{
		SupplierName:   supplier.Name,
		SupplierEmail:  supplier.Email,
		CustomerNumber: supplier.Kundennummer,
		Labels:         labels,
		SummaryItems:   orderSummaryItems,
		TotalAllocated: totalAllocated,
	}, nil
}

// verarbeiteBestellItem validiert eine Bestellposition, lädt den Titel, reserviert die
// Barcodes und erzeugt die Exemplar-Datensätze samt (optionalen) Etiketten.
func (s *OrderService) verarbeiteBestellItem(ctx context.Context, tx pgx.Tx, item OrderItemRequest, supplierName string) (*bestellItemResult, error) {
	if item.Menge <= 0 || item.Menge > 200 {
		return nil, fmt.Errorf("invalid quantity %d for title %s", item.Menge, item.TitelID)
	}

	title, err := s.bookRepo.GetTitleByIDTx(ctx, tx, item.TitelID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("book title %s not found", item.TitelID)
		}
		return nil, err
	}

	res := &bestellItemResult{
		summary: OrderedItem{
			Titel:  title.Titel,
			Autor:  title.Autor,
			ISBN:   title.ISBN,
			Verlag: title.Verlag,
			Menge:  item.Menge,
		},
		position: bestellungPosition{
			titelID:   item.TitelID,
			titelName: title.Titel,
			isbn:      title.ISBN,
			menge:     item.Menge,
			preis:     item.Preis,
		},
		betrag: float64(item.Menge) * item.Preis,
	}

	// ALWAYS pre-allocate barcodes in the database
	barcodes, err := s.bookRepo.GenerateBarcodes(ctx, item.Menge)
	if err != nil {
		return nil, fmt.Errorf("sequence error: %w", err)
	}

	statusText := fmt.Sprintf("Im Zulauf - %s", supplierName)
	if !item.GenerateBarcodes {
		statusText = fmt.Sprintf("Bestellt (ohne Vorab-Barcode) - %s", supplierName)
	}

	for i := 0; i < item.Menge; i++ {
		barcodeID := barcodes[i]
		res.copies = append(res.copies, repository.BookCopyInsert{
			TitelID:         item.TitelID,
			BarcodeID:       barcodeID,
			ZustandNotiz:    statusText,
			IstAusleihbar:   false,
			EtikettGedruckt: false,
			Einkaufspreis:   item.Preis,
		})

		// Only add to labels for the supplier PDF if requested
		if item.GenerateBarcodes {
			res.labels = append(res.labels, BarcodeLabelDetail{
				BarcodeID: barcodeID,
				Titel:     title.Titel,
				Autor:     title.Autor,
				ISBN:      title.ISBN,
			})
		}
	}

	return res, nil
}

// insertBestellverlauf schreibt den Bestellkopf und liefert die erzeugte Bestell-ID.
func (s *OrderService) insertBestellverlauf(ctx context.Context, tx pgx.Tx, req SubmitOrderRequest, supplier *repository.Supplier, gesamtbetrag float64, totalAllocated int) (string, error) {
	var bestellungID string
	err := tx.QueryRow(ctx, `
		INSERT INTO bestellungen_verlauf
			(lieferant_id, lieferant_name, lieferant_email, kundennummer, gesamtbetrag, anzahl_exemplare)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id`,
		req.SupplierID, supplier.Name, supplier.Email, supplier.Kundennummer,
		gesamtbetrag, totalAllocated,
	).Scan(&bestellungID)
	if err != nil {
		return "", fmt.Errorf("bestellverlauf insert: %w", err)
	}
	return bestellungID, nil
}

// insertBestellpositionen schreibt alle Positionen des Bestellkopfs.
func (s *OrderService) insertBestellpositionen(ctx context.Context, tx pgx.Tx, bestellungID string, positionen []bestellungPosition) error {
	for _, pos := range positionen {
		if _, err := tx.Exec(ctx, `
			INSERT INTO bestellungen_positionen
				(bestellung_id, titel_id, titel_name, isbn, menge, einzelpreis)
			VALUES ($1, $2, $3, $4, $5, $6)`,
			bestellungID, pos.titelID, pos.titelName, pos.isbn, pos.menge, pos.preis,
		); err != nil {
			return fmt.Errorf("position insert: %w", err)
		}
	}
	return nil
}
