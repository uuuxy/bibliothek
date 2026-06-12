package repository

import (
	"bibliothek/db"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	
)

// AuditRepository manages immutable logs and auditable resource deletions.
type AuditRepository interface {
	// Manual administrative deletions
	DeleteTitle(ctx context.Context, titleID string, bearbeiterID string) error
	DeleteCopy(ctx context.Context, copyID string, bearbeiterID string) error
	DeleteUser(ctx context.Context, userID string, bearbeiterID string) error

	// Student hard-delete with audit trail (called by API and GDPR Cronjob)
	DeleteStudent(ctx context.Context, studentID string, bearbeiterID string, grund string) error

	// Fee cancellation audit
	StornierungGebuehr(ctx context.Context, schadensfallID string, bearbeiterID string, betrag float64, grund string) error

	// Loan checkout/return audit (append-only event log)
	LogAusleihe(ctx context.Context, exemplarID string, schuelerID string, benutzerID string, bearbeiterID string) error
	LogRueckgabe(ctx context.Context, exemplarID string, schuelerID string, benutzerID string, bearbeiterID string) error

	// System-triggered batch audit (no user actor)
	LogSystemAktion(ctx context.Context, tabelle string, aktion string, kontext string, details map[string]any) error
}

type pgAuditRepository struct {
	db db.PgxPoolIface
}

// NewAuditRepository instantiates a pgAuditRepository.
func NewAuditRepository(db db.PgxPoolIface) AuditRepository {
	return &pgAuditRepository{db: db}
}

// insertAuditLog is the single internal helper that writes to audit_log.
// All writes go through here to ensure consistency and append-only semantics.
func (r *pgAuditRepository) insertAuditLog(
	ctx context.Context,
	tx pgx.Tx,
	tabelle, aktion, datensatzID string,
	bearbeiterID *string,
	akteur string,
	kontext *string,
	details map[string]any,
) error {
	var detailsJSON []byte
	if details != nil {
		var err error
		detailsJSON, err = json.Marshal(details)
		if err != nil {
			return fmt.Errorf("audit details serialization: %w", err)
		}
	}

	const q = `
		INSERT INTO audit_log
		  (tabelle, aktion, datensatz_id, bearbeiter_id, akteur, kontext, details)
		VALUES ($1, $2, $3::uuid, $4, $5, $6, $7)
	`
	_, err := tx.Exec(ctx, q,
		tabelle, aktion, datensatzID,
		bearbeiterID, akteur, kontext,
		func() interface{} {
			if detailsJSON == nil {
				return nil
			}
			return string(detailsJSON)
		}(),
	)
	return err
}

// ── Administrative deletions ──────────────────────────────────────────────────

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
		return errors.New("exemplar ist aktuell noch verliehen")
	}

	// Clean up related records (damage reports and past loans) before deleting the copy
	if _, err = tx.Exec(ctx, "DELETE FROM schadensfaelle WHERE exemplar_id = $1", copyID); err != nil {
		return fmt.Errorf("failed to delete damage records: %w", err)
	}
	if _, err = tx.Exec(ctx, "DELETE FROM ausleihen WHERE exemplar_id = $1 AND rueckgabe_am IS NOT NULL", copyID); err != nil {
		return fmt.Errorf("failed to delete past loans: %w", err)
	}

	if _, err = tx.Exec(ctx, "DELETE FROM buecher_exemplare WHERE id = $1", copyID); err != nil {
		return err
	}

	kontext := "Buch ausgebuchen"
	if err = r.insertAuditLog(ctx, tx, "buecher_exemplare", "DELETE", copyID,
		&bearbeiterID, "USER", &kontext,
		map[string]any{"barcode_id": barcode, "zustand_notiz": zustandNotiz, "titel_id": titelID, "titel": titel},
	); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// DeleteUser purges a system user and records the deletion in audit_log.
func (r *pgAuditRepository) DeleteUser(ctx context.Context, userID string, bearbeiterID string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Snapshot before deletion
	var vorname, nachname, email, rolle string
	err = tx.QueryRow(ctx,
		`SELECT coalesce(vorname,''), coalesce(nachname,''), coalesce(email,''), coalesce(rolle::text,'')
		 FROM benutzer WHERE id = $1`,
		userID,
	).Scan(&vorname, &nachname, &email, &rolle)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("failed to snapshot user for audit: %w", err)
	}

	if _, err = tx.Exec(ctx, "DELETE FROM benutzer WHERE id = $1", userID); err != nil {
		return err
	}

	if err = r.insertAuditLog(ctx, tx, "benutzer", "DELETE", userID,
		&bearbeiterID, "USER", nil,
		map[string]any{"vorname": vorname, "nachname": nachname, "email": email, "rolle": rolle},
	); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// ── DSGVO: Student hard-delete ────────────────────────────────────────────────

// DeleteStudent transactionally hard-deletes a student record and writes an immutable
// audit trail. Safe to call from both the HTTP API (bearbeiterID = user UUID) and
// the GDPR Cronjob (bearbeiterID = "" → SYSTEM actor).
func (r *pgAuditRepository) DeleteStudent(ctx context.Context, studentID string, bearbeiterID string, grund string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Snapshot before deletion (DSGVO-konforme Protokollierung)
	var vorname, nachname, klasse, barcodeID string
	var abgaengerJahr int
	err = tx.QueryRow(ctx,
		`SELECT coalesce(vorname,''), coalesce(nachname,''), coalesce(klasse,''),
		        coalesce(barcode_id,''), coalesce(abgaenger_jahr, 0)
		 FROM schueler WHERE id = $1`,
		studentID,
	).Scan(&vorname, &nachname, &klasse, &barcodeID, &abgaengerJahr)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("failed to snapshot student for audit: %w", err)
	}

	// Anonymize closed loans: set schueler_id = NULL for all returned loans
	if _, err = tx.Exec(ctx,
		`UPDATE ausleihen SET schueler_id = NULL WHERE schueler_id = $1 AND rueckgabe_am IS NOT NULL`,
		studentID,
	); err != nil {
		return fmt.Errorf("anonymizing returned loans: %w", err)
	}

	// Anonymize any past audit logs for this student to remove PII
	if _, err = tx.Exec(ctx, `
		UPDATE audit_log
		SET details = details || '{"vorname":"Anonymisiert", "nachname":"Anonymisiert", "klasse":"Anonymisiert"}'::jsonb
		WHERE (datensatz_id = $1 OR (details->>'schueler_id') = $2) AND details ? 'vorname'
	`, studentID, studentID); err != nil {
		return fmt.Errorf("anonymizing past audit_logs: %w", err)
	}

	// Delete paid damage cases (unpaid ones block deletion – enforced by caller)
	if _, err = tx.Exec(ctx,
		`DELETE FROM schadensfaelle WHERE schueler_id = $1 AND ist_bezahlt = true`,
		studentID,
	); err != nil {
		return fmt.Errorf("deleting paid damages: %w", err)
	}

	// Hard-delete the student record
	tag, err := tx.Exec(ctx, `DELETE FROM schueler WHERE id = $1`, studentID)
	if err != nil {
		return fmt.Errorf("deleting student: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("student %s not found", studentID)
	}

	// Determine actor type
	var akteur string
	var bearbeiterPtr *string
	if bearbeiterID != "" {
		akteur = "USER"
		bearbeiterPtr = &bearbeiterID
	} else {
		akteur = "SYSTEM"
	}

	kontext := "DSGVO-Löschroutine"

	if err = r.insertAuditLog(ctx, tx, "schueler", "DELETE", studentID,
		bearbeiterPtr, akteur, &kontext,
		map[string]any{
			"vorname":        "Anonymisiert",
			"nachname":       "Anonymisiert",
			"klasse":         "Anonymisiert",
			"barcode_id":     barcodeID,
			"abgaenger_jahr": abgaengerJahr,
			"grund":          grund,
			"geloescht_am":   time.Now().UTC().Format(time.RFC3339),
		},
	); err != nil {
		return fmt.Errorf("writing audit log: %w", err)
	}

	return tx.Commit(ctx)
}

// ── Audit: Gebühren-Stornierung ───────────────────────────────────────────────

// StornierungGebuehr marks a damage case as cancelled (storniert) and writes an
// immutable audit record. This replaces hard-deletes of Schadensfälle.
func (r *pgAuditRepository) StornierungGebuehr(ctx context.Context, schadensfallID string, bearbeiterID string, betrag float64, grund string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Mark as cancelled in schadensfaelle
	tag, err := tx.Exec(ctx, `
		UPDATE schadensfaelle
		SET ist_bezahlt = true,
		    storniert_am = NOW(),
		    storniert_von = $1,
		    stornierungsgrund = $2,
		    aktualisiert_am = NOW()
		WHERE id = $3 AND ist_bezahlt = false
	`, bearbeiterID, grund, schadensfallID)
	if err != nil {
		return fmt.Errorf("stornierung schadensfaelle: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("schadensfall %s nicht gefunden oder bereits bezahlt/storniert", schadensfallID)
	}

	kontext := "Gebühr storniert"
	if err = r.insertAuditLog(ctx, tx, "schadensfaelle", "STORNIERUNG", schadensfallID,
		&bearbeiterID, "USER", &kontext,
		map[string]any{
			"betrag":        betrag,
			"grund":         grund,
			"storniert_am":  time.Now().UTC().Format(time.RFC3339),
			"bearbeiter_id": bearbeiterID,
		},
	); err != nil {
		return fmt.Errorf("writing audit log: %w", err)
	}

	return tx.Commit(ctx)
}

// ── Audit: Ausleihe & Rückgabe ────────────────────────────────────────────────

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

// ── System-triggered batch audit ─────────────────────────────────────────────

// LogSystemAktion writes a SYSTEM-actor audit record (no bearbeiter_id).
// Used by Cronjobs (GDPR anonymization, backup, etc.).
func (r *pgAuditRepository) LogSystemAktion(ctx context.Context, tabelle string, aktion string, kontext string, details map[string]any) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// System actions use a sentinel UUID-shaped kontext key, not a real record ID.
	// We use a canonical SYSTEM ID for datensatz_id.
	const systemSentinelID = "00000000-0000-0000-0000-000000000000"

	if err = r.insertAuditLog(ctx, tx, tabelle, aktion, systemSentinelID,
		nil, "SYSTEM", &kontext, details,
	); err != nil {
		return err
	}

	return tx.Commit(ctx)
}
