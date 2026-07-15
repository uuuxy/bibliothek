package repository

import (
	"context"
	"fmt"
)

// InventoryRepository handles all database interactions required during a physical
// book inventory (stock-take). It encapsulates setting scopes, tracking scans,
// and finalizing the inventory by marking missing books as lost.
type InventoryRepository struct {
	db DBQueryer
}

// NewInventoryRepository initializes a new InventoryRepository.
func NewInventoryRepository(db DBQueryer) *InventoryRepository {
	return &InventoryRepository{db: db}
}

// ResetInventoryStatus globally removes any active inventory status from all books.
func (r *InventoryRepository) ResetInventoryStatus(ctx context.Context) error {
	_, err := r.db.Exec(ctx, "UPDATE buecher_exemplare SET inventur_status = NULL")
	return err
}

// SetInventoryScopeGlobal marks all active, non-discarded books as 'ausstehend'
// and returns the total expected count.
func (r *InventoryRepository) SetInventoryScopeGlobal(ctx context.Context) (int, error) {
	var count int
	query := `
		WITH updated AS (
			UPDATE buecher_exemplare
			SET inventur_status = 'ausstehend'
			WHERE ist_ausgesondert = false AND ist_ausleihbar = true
			RETURNING id
		)
		SELECT count(*) FROM updated
	`
	err := r.db.QueryRow(ctx, query).Scan(&count)
	return count, err
}

// SetInventoryScopeSignature marks all active books belonging to a specific signature ID
// as 'ausstehend' and returns the total expected count.
func (r *InventoryRepository) SetInventoryScopeSignature(ctx context.Context, signatureID int) (int, error) {
	var count int
	query := `
		WITH updated AS (
			UPDATE buecher_exemplare e
			SET inventur_status = 'ausstehend'
			FROM buecher_titel t
			WHERE e.titel_id = t.id 
			  AND e.ist_ausgesondert = false 
			  AND e.ist_ausleihbar = true 
			  AND t.signature_id = $1
			RETURNING e.id
		)
		SELECT count(*) FROM updated
	`
	err := r.db.QueryRow(ctx, query, signatureID).Scan(&count)
	return count, err
}

// InventoryScanResult contains the book data retrieved during a barcode scan.
type InventoryScanResult struct {
	CopyID         string
	Title          string
	CoverURL       string
	IsAusgesondert bool
	InventurStatus *string
	IsLent         bool
}

// GetExemplarForInventoryScan retrieves the details of a copy by its barcode
// to determine if it is eligible for inventory check-in.
func (r *InventoryRepository) GetExemplarForInventoryScan(ctx context.Context, barcodeID string) (*InventoryScanResult, error) {
	var res InventoryScanResult
	query := `
		SELECT e.id, t.titel, coalesce(t.cover_url, ''), e.ist_ausgesondert, e.inventur_status, EXISTS (
			SELECT 1 FROM ausleihen a 
			WHERE a.exemplar_id = e.id AND a.rueckgabe_am IS NULL
		) AS is_lent
		FROM buecher_exemplare e
		JOIN buecher_titel t ON e.titel_id = t.id
		WHERE e.barcode_id = $1
		LIMIT 1
	`
	err := r.db.QueryRow(ctx, query, barcodeID).Scan(&res.CopyID, &res.Title, &res.CoverURL, &res.IsAusgesondert, &res.InventurStatus, &res.IsLent)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

// MarkExemplarScanned registers an inventory scan by setting the status to 'erfasst'.
func (r *InventoryRepository) MarkExemplarScanned(ctx context.Context, copyID string) error {
	updateQuery := `
		UPDATE buecher_exemplare
		SET inventur_status = 'erfasst',
		    inventur_geprueft_am = CURRENT_TIMESTAMP,
		    aktualisiert_am = CURRENT_TIMESTAMP
		WHERE id = $1
	`
	_, err := r.db.Exec(ctx, updateQuery, copyID)
	return err
}

// MarkRemainingAsLostAndReset marks all remaining 'ausstehend' items as discarded/lost
// due to inventory, resets the global inventory status to NULL, and returns the
// total number of books declared lost.
//
// Hinweis: Der frühere Aufruf von update_verfuegbar_count($1) ist entfernt —
// die SQL-Funktion existierte nirgends. Innerhalb der Finish-Transaktion
// brach der fehlgeschlagene Aufruf die Transaktion ab (SQLSTATE 25P02) und
// machte JEDEN Inventur-Abschluss zum 500; die Verfügbarkeit wird ohnehin
// dynamisch über view_buecher_bestand berechnet.
func (r *InventoryRepository) MarkRemainingAsLostAndReset(ctx context.Context) (int, error) {
	tag, err := r.db.Exec(ctx, `
		UPDATE buecher_exemplare
		SET ist_ausleihbar = false,
		    ist_ausgesondert = true,
		    aussonderung_grund = 'VERLUST',
		    zustand_notiz = 'Verlust bei Inventur',
		    aktualisiert_am = CURRENT_TIMESTAMP
		WHERE inventur_status = 'ausstehend'
	`)
	if err != nil {
		return 0, fmt.Errorf("failed to mark as lost: %w", err)
	}

	// Reset all inventur_status to NULL globally
	if err := r.ResetInventoryStatus(ctx); err != nil {
		return 0, fmt.Errorf("failed to reset status: %w", err)
	}

	return int(tag.RowsAffected()), nil
}
