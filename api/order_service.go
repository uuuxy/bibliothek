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

	var copyInserts []repository.BookCopyInsert

	for _, item := range req.Items {
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

		orderSummaryItems = append(orderSummaryItems, OrderedItem{
			Titel:  title.Titel,
			Autor:  title.Autor,
			ISBN:   title.ISBN,
			Verlag: title.Verlag,
			Menge:  item.Menge,
		})

		// ALWAYS pre-allocate barcodes in the database
		barcodes, err := s.bookRepo.GenerateBarcodes(ctx, item.Menge)
		if err != nil {
			return nil, fmt.Errorf("sequence error: %w", err)
		}

		statusText := fmt.Sprintf("Im Zulauf - %s", supplier.Name)
		if !item.GenerateBarcodes {
			statusText = fmt.Sprintf("Bestellt (ohne Vorab-Barcode) - %s", supplier.Name)
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
					Titel:     title.Titel,
					Autor:     title.Autor,
					ISBN:      title.ISBN,
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
		SupplierName:   supplier.Name,
		SupplierEmail:  supplier.Email,
		CustomerNumber: supplier.Kundennummer,
		Labels:         labels,
		SummaryItems:   orderSummaryItems,
		TotalAllocated: totalAllocated,
	}, nil
}
