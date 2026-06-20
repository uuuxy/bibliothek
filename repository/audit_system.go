package repository

import (
	"context"
	"fmt"
	"time"
)

// StornierungGebuehr markiert eine ausstehende Gebühr (Schadensfall) als storniert und schreibt einen
// revisionssicheren Audit-Eintrag. Dies ersetzt das physische Löschen von Schadensfällen, um die Historie zu wahren.
func (r *pgAuditRepository) StornierungGebuehr(ctx context.Context, schadensfallID string, bearbeiterID string, betrag float64, grund string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Schadensfall als bezahlt/storniert markieren und Metadaten eintragen
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

// LogSystemAktion schreibt einen Audit-Eintrag für eine rein systemgesteuerte Aktion (ohne Bearbeiter-ID).
// Dies wird typischerweise von Cronjobs, DSGVO-Routinebereinigungen oder Datensicherungen verwendet.
func (r *pgAuditRepository) LogSystemAktion(ctx context.Context, tabelle string, aktion string, kontext string, details map[string]any) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Systemaktionen verwenden eine Standard-Null-UUID als Datensatz-ID
	const systemSentinelID = "00000000-0000-0000-0000-000000000000"

	if err = r.insertAuditLog(ctx, tx, tabelle, aktion, systemSentinelID,
		nil, "SYSTEM", &kontext, details,
	); err != nil {
		return err
	}

	return tx.Commit(ctx)
}
