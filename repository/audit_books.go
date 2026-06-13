package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// DeleteTitle removes a book title from the master catalog and creates an immutable audit record.
func (r *pgAuditRepository) DeleteTitle(ctx context.Context, titleID string, bearbeiterID string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Snapshot: capture title metadata before deletion for the audit trail
	var titel, autor, isbn string
	err = tx.QueryRow(ctx,
		`SELECT coalesce(titel,''), coalesce(autor,''), coalesce(isbn,'') FROM buecher_titel WHERE id = $1`,
		titleID,
	).Scan(&titel, &autor, &isbn)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("failed to snapshot title for audit: %w", err)
	}

	// Loan protection: Check if any copy of this title is currently on loan
	var activeLoans []string
	rows, err := tx.Query(ctx, `
		SELECT e.barcode_id 
		FROM ausleihen a 
		JOIN buecher_exemplare e ON a.exemplar_id = e.id 
		WHERE e.titel_id = $1 AND a.rueckgabe_am IS NULL
	`, titleID)
	if err != nil {
		return fmt.Errorf("failed to check active loans for title: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var barcode string
		if err := rows.Scan(&barcode); err == nil {
			activeLoans = append(activeLoans, barcode)
		}
	}
	if len(activeLoans) > 0 {
		return fmt.Errorf("löschen fehlgeschlagen: Folgende Exemplare sind noch verliehen: %v", activeLoans)
	}

	// Clean up related records for ALL copies of this title to prevent ON DELETE RESTRICT errors
	if _, err = tx.Exec(ctx, "DELETE FROM schadensfaelle WHERE exemplar_id IN (SELECT id FROM buecher_exemplare WHERE titel_id = $1)", titleID); err != nil {
		return fmt.Errorf("failed to delete damage records for title: %w", err)
	}
	if _, err = tx.Exec(ctx, "DELETE FROM ausleihen WHERE exemplar_id IN (SELECT id FROM buecher_exemplare WHERE titel_id = $1) AND rueckgabe_am IS NOT NULL", titleID); err != nil {
		return fmt.Errorf("failed to delete past loans for title: %w", err)
	}

	// Delete all associated copies first
	if _, err = tx.Exec(ctx, "DELETE FROM buecher_exemplare WHERE titel_id = $1", titleID); err != nil {
		return fmt.Errorf("failed to delete associated copies: %w", err)
	}

	if _, err = tx.Exec(ctx, "DELETE FROM buecher_titel WHERE id = $1", titleID); err != nil {
		return err
	}

	if err = r.insertAuditLog(ctx, tx, "buecher_titel", "DELETE", titleID,
		&bearbeiterID, "USER", nil,
		map[string]any{"titel": titel, "autor": autor, "isbn": isbn},
	); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// DeleteCopy removes a physical book copy from circulation and creates an immutable audit record.
func (r *pgAuditRepository) DeleteCopy(ctx context.Context, copyID string, bearbeiterID string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Snapshot: capture copy details before deletion for the audit trail
	var barcode, zustandNotiz, titel string
	var titelID string
	err = tx.QueryRow(ctx,
		`SELECT e.barcode_id, coalesce(e.zustand_notiz,''), e.titel_id, t.titel
		 FROM buecher_exemplare e
		 JOIN buecher_titel t ON e.titel_id = t.id
		 WHERE e.id = $1`,
		copyID,
	).Scan(&barcode, &zustandNotiz, &titelID, &titel)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("failed to snapshot copy for audit: %w", err)
	}

	// Loan protection: Check if the copy is currently on loan
	var activeLoanCount int
	err = tx.QueryRow(ctx, "SELECT count(*) FROM ausleihen WHERE exemplar_id = $1 AND rueckgabe_am IS NULL", copyID).Scan(&activeLoanCount)
	if err != nil {
		return fmt.Errorf("failed to check active loans for copy: %w", err)
	}
	if activeLoanCount > 0 {
		return errors.New("Exemplar ist aktuell noch verliehen!")
	}

	// Soft-Delete: We mark the copy as decommissioned instead of hard-deleting it.
	// We DO NOT delete from 'ausleihen' and 'schadensfaelle' to preserve history.
	if _, err = tx.Exec(ctx, "UPDATE buecher_exemplare SET ist_ausgesondert = true, ist_ausleihbar = false, zustand_notiz = 'Systematisch gelöscht' WHERE id = $1", copyID); err != nil {
		return err
	}

	kontext := "Buch ausgebuchen (Soft-Delete)"
	if err = r.insertAuditLog(ctx, tx, "buecher_exemplare", "UPDATE", copyID,
		&bearbeiterID, "USER", &kontext,
		map[string]any{"barcode_id": barcode, "zustand_notiz": zustandNotiz, "titel_id": titelID, "titel": titel, "action": "soft_delete"},
	); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// LogAusleihe writes an immutable checkout event to the audit log.
// This is NOT called from within a larger transaction – it creates its own.
func (r *pgAuditRepository) LogAusleihe(ctx context.Context, exemplarID string, schuelerID string, benutzerID string, bearbeiterID string) error {
	return r.logLoanEvent(ctx, "ausleihen", "CHECKOUT", exemplarID, schuelerID, benutzerID, bearbeiterID)
}

// LogRueckgabe writes an immutable return event to the audit log.
func (r *pgAuditRepository) LogRueckgabe(ctx context.Context, exemplarID string, schuelerID string, benutzerID string, bearbeiterID string) error {
	return r.logLoanEvent(ctx, "ausleihen", "RETURN", exemplarID, schuelerID, benutzerID, bearbeiterID)
}

func (r *pgAuditRepository) logLoanEvent(ctx context.Context, tabelle, aktion, exemplarID, schuelerID, benutzerID, bearbeiterID string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var bearbeiterPtr *string
	if bearbeiterID != "" {
		bearbeiterPtr = &bearbeiterID
	}

	details := map[string]any{
		"exemplar_id": exemplarID,
		"zeitpunkt":   time.Now().UTC().Format(time.RFC3339),
	}
	if schuelerID != "" {
		details["schueler_id"] = schuelerID
	}
	if benutzerID != "" {
		details["benutzer_id"] = benutzerID
	}

	if err = r.insertAuditLog(ctx, tx, tabelle, aktion, exemplarID,
		bearbeiterPtr, "USER", nil,
		details,
	); err != nil {
		return err
	}

	return tx.Commit(ctx)
}
