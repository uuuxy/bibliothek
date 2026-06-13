package repository

import (
	"bibliothek/db"
	"context"
	"encoding/json"
	"fmt"

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
