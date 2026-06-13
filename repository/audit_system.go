package repository

import (
	"context"
	"fmt"
	"time"
)

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
