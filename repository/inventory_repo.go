package repository

import (
	"context"
	"fmt"
)

// InventoryRepository handles all database interactions required during a physical
// book inventory (stock-take). Der Fortschritt lebt seit Migration 045 in Sessions
// (inventur_sessions / inventur_erfassungen) statt in einer globalen Spalte; die
// Lebenszyklus-Methoden dazu stehen in inventur_session_repo.go und
// inventur_session_finish.go.
type InventoryRepository struct {
	db DBQueryer
}

// NewInventoryRepository initializes a new InventoryRepository.
func NewInventoryRepository(db DBQueryer) *InventoryRepository {
	return &InventoryRepository{db: db}
}

// InventoryScanResult contains the book data retrieved during a barcode scan.
type InventoryScanResult struct {
	CopyID         string
	Title          string
	CoverURL       string
	IsAusgesondert bool
	IsLent         bool
}

// GetExemplarForInventoryScan retrieves the details of a copy by its barcode
// to determine if it is eligible for inventory check-in.
func (r *InventoryRepository) GetExemplarForInventoryScan(ctx context.Context, barcodeID string) (*InventoryScanResult, error) {
	var res InventoryScanResult
	query := `
		SELECT e.id, t.titel, coalesce(t.cover_url, ''), e.ist_ausgesondert, EXISTS (
			SELECT 1 FROM ausleihen a
			WHERE a.exemplar_id = e.id AND a.rueckgabe_am IS NULL
		) AS is_lent
		FROM buecher_exemplare e
		JOIN buecher_titel t ON e.titel_id = t.id
		WHERE e.barcode_id = $1
		LIMIT 1
	`
	err := r.db.QueryRow(ctx, query, barcodeID).Scan(&res.CopyID, &res.Title, &res.CoverURL, &res.IsAusgesondert, &res.IsLent)
	if err != nil {
		return nil, fmt.Errorf("exemplar für inventur-scan nicht ladbar: %w", err)
	}
	return &res, nil
}
