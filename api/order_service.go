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
	db       *db.Database
	bookRepo repository.BookRepository
}

// NewOrderService erstellt eine neue OrderService-Instanz.
func NewOrderService(database *db.Database, bookRepo repository.BookRepository) *OrderService {
	return &OrderService{db: database, bookRepo: bookRepo}
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

// ProcessOrder verarbeitet eine eingehende SubmitOrderRequest innerhalb einer Transaktion, generiert Barcodes und gibt das OrderResult zurück.
func (s *OrderService) ProcessOrder(ctx context.Context, req SubmitOrderRequest) (*OrderResult, error) {
	// 1. Lieferantendetails abrufen
	var supplierName, supplierEmail, customerNumber string
	err := s.db.Pool.QueryRow(ctx, `
		SELECT name, email, kundennummer 
		FROM lieferanten 
		WHERE id = $1
	`, req.SupplierID).Scan(&supplierName, &supplierEmail, &customerNumber)
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
	defer func() { _ = tx.Rollback(ctx) }()

	labels := make([]BarcodeLabelDetail, 0)
	orderSummaryItems := make([]OrderedItem, 0)
	var totalAllocated int

	var copyInserts []repository.BookCopyInsert

	for _, item := range req.Items {
		if item.Menge <= 0 || item.Menge > 200 {
			return nil, fmt.Errorf("invalid quantity %d for title %s", item.Menge, item.TitelID)
		}

		var titel, autor, isbn, verlag string
		err = tx.QueryRow(ctx, "SELECT titel, coalesce(autor, ''), coalesce(isbn, ''), coalesce(verlag, '') FROM buecher_titel WHERE id = $1", item.TitelID).Scan(&titel, &autor, &isbn, &verlag)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, fmt.Errorf("book title %s not found", item.TitelID)
			}
			return nil, err
		}

		orderSummaryItems = append(orderSummaryItems, OrderedItem{
			Titel:  titel,
			Autor:  autor,
			ISBN:   isbn,
			Verlag: verlag,
			Menge:  item.Menge,
		})

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
			copyInserts = append(copyInserts, repository.BookCopyInsert{
				TitelID:         item.TitelID,
				BarcodeID:       barcodeID,
				ZustandNotiz:    statusText,
				IstAusleihbar:   false,
				EtikettGedruckt: false,
				Einkaufspreis:   item.Preis,
			})

			// Only add to labels for the supplier PDF if requested
			if item.GenerateBarcodes {
				labels = append(labels, BarcodeLabelDetail{
					BarcodeID: barcodeID,
					Titel:     titel,
					Autor:     autor,
					ISBN:      isbn,
				})
			}
		}
		totalAllocated += item.Menge
	}

	if err := s.bookRepo.BulkInsertCopiesTx(ctx, tx, copyInserts); err != nil {
		return nil, fmt.Errorf("bulk insert error: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return &OrderResult{
		SupplierName:   supplierName,
		SupplierEmail:  supplierEmail,
		CustomerNumber: customerNumber,
		Labels:         labels,
		SummaryItems:   orderSummaryItems,
		TotalAllocated: totalAllocated,
	}, nil
}
