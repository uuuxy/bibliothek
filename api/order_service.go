package api

import (
	"context"
	"errors"
	"fmt"

	"bibliothek/db"
	"github.com/jackc/pgx/v5"
)

// OrderService verarbeitet die Geschäftslogik zum Erstellen und Verarbeiten von Bestellungen.
type OrderService struct {
	db *db.Database
}

// NewOrderService erstellt eine neue OrderService-Instanz.
func NewOrderService(database *db.Database) *OrderService {
	return &OrderService{db: database}
}

// OrderResult enthält das Ergebnis einer verarbeiteten Bestellung, einschließlich der generierten Barcodes.
type OrderResult struct {
	SupplierName   string
	SupplierEmail  string
	CustomerNumber string
	Labels         []BarcodeLabelDetail
	SummaryItems   []OrderedItem
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

	var copyRows [][]any

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

		if req.GenerateBarcodes {
			rows, err := tx.Query(ctx, "SELECT 'B-' || LPAD(nextval('barcode_seq')::TEXT, 5, '0') FROM generate_series(1, $1)", item.Menge)
			if err != nil {
				return nil, fmt.Errorf("sequence fetch error: %w", err)
			}
			var barcodes []string
			for rows.Next() {
				var barcodeID string
				if err := rows.Scan(&barcodeID); err != nil {
					rows.Close()
					return nil, fmt.Errorf("sequence scan error: %w", err)
				}
				barcodes = append(barcodes, barcodeID)
			}
			rows.Close()

			if len(barcodes) != item.Menge {
				return nil, fmt.Errorf("expected %d sequences, got %d", item.Menge, len(barcodes))
			}

			statusText := fmt.Sprintf("Im Zulauf - %s", supplierName)

			for i := 0; i < item.Menge; i++ {
				barcodeID := barcodes[i]
				copyRows = append(copyRows, []any{item.TitelID, barcodeID, statusText, false, false, item.Preis})

				labels = append(labels, BarcodeLabelDetail{
					BarcodeID: barcodeID,
					Titel:     titel,
					Autor:     autor,
					ISBN:      isbn,
				})
			}
		}
	}

	if len(copyRows) > 0 {
		_, err = tx.CopyFrom(
			ctx,
			pgx.Identifier{"buecher_exemplare"},
			[]string{"titel_id", "barcode_id", "zustand_notiz", "ist_ausleihbar", "etikett_gedruckt", "einkaufspreis"},
			pgx.CopyFromRows(copyRows),
		)
		if err != nil {
			return nil, fmt.Errorf("bulk insert error: %w", err)
		}
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
	}, nil
}
