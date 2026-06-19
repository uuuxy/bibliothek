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

	var copyInserts []repository.BookCopyInsert

	// Vorab alle Titel abfragen, um das N+1 Query Problem zu beheben
	titelIDs := make([]string, 0, len(req.Items))
	for _, item := range req.Items {
		if item.Menge <= 0 || item.Menge > 200 {
			return nil, fmt.Errorf("invalid quantity %d for title %s", item.Menge, item.TitelID)
		}
		titelIDs = append(titelIDs, item.TitelID)
	}

	type titelInfo struct {
		Titel  string
		Autor  string
		ISBN   string
		Verlag string
	}

	titelMap := make(map[string]titelInfo, len(titelIDs))
	if len(titelIDs) > 0 {
		rows, err := tx.Query(ctx, "SELECT id, titel, coalesce(autor, ''), coalesce(isbn, ''), coalesce(verlag, '') FROM buecher_titel WHERE id = ANY($1)", titelIDs)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch book titles: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			var id, titel, autor, isbn, verlag string
			if err := rows.Scan(&id, &titel, &autor, &isbn, &verlag); err != nil {
				return nil, fmt.Errorf("failed to scan book title: %w", err)
			}
			titelMap[id] = titelInfo{Titel: titel, Autor: autor, ISBN: isbn, Verlag: verlag}
		}
		if err := rows.Err(); err != nil {
			return nil, fmt.Errorf("error iterating book titles: %w", err)
		}
	}

	for _, item := range req.Items {
		info, ok := titelMap[item.TitelID]
		if !ok {
			return nil, fmt.Errorf("book title %s not found", item.TitelID)
		}

		orderSummaryItems = append(orderSummaryItems, OrderedItem{
			Titel:  info.Titel,
			Autor:  info.Autor,
			ISBN:   info.ISBN,
			Verlag: info.Verlag,
			Menge:  item.Menge,
		})

		titel := info.Titel
		autor := info.Autor
		isbn := info.ISBN

		if req.GenerateBarcodes {
			barcodes, err := s.bookRepo.GenerateBarcodes(ctx, item.Menge)
			if err != nil {
				return nil, fmt.Errorf("sequence error: %w", err)
			}

			statusText := fmt.Sprintf("Im Zulauf - %s", supplierName)

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

				labels = append(labels, BarcodeLabelDetail{
					BarcodeID: barcodeID,
					Titel:     titel,
					Autor:     autor,
					ISBN:      isbn,
				})
			}
		}
	}

	if err := s.bookRepo.BulkInsertCopies(ctx, copyInserts); err != nil {
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
	}, nil
}
