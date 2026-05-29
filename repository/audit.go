package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// AuditRepository manages immutable logs and auditable resource deletions.
type AuditRepository interface {
	DeleteTitle(ctx context.Context, titleID string, bearbeiterID string) error
	DeleteUser(ctx context.Context, userID string, bearbeiterID string) error
}

type pgAuditRepository struct {
	db *pgxpool.Pool
}

// NewAuditRepository instantiates a pgAuditRepository.
func NewAuditRepository(db *pgxpool.Pool) AuditRepository {
	return &pgAuditRepository{db: db}
}

// DeleteTitle removes a book title from the master catalog and creates an immutable log in audit_log.
func (r *pgAuditRepository) DeleteTitle(ctx context.Context, titleID string, bearbeiterID string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Delete title from the database (cascades to copies via foreign key)
	_, err = tx.Exec(ctx, "DELETE FROM buecher_titel WHERE id = $1", titleID)
	if err != nil {
		return err
	}

	// Insert audit record
	queryAudit := `
		INSERT INTO audit_log (tabelle, aktion, datensatz_id, bearbeiter_id)
		VALUES ($1, $2, $3, $4)
	`
	_, err = tx.Exec(ctx, queryAudit, "buecher_titel", "DELETE", titleID, bearbeiterID)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// DeleteUser purges a system user and records the transaction in audit_log.
func (r *pgAuditRepository) DeleteUser(ctx context.Context, userID string, bearbeiterID string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Delete user from the database
	_, err = tx.Exec(ctx, "DELETE FROM benutzer WHERE id = $1", userID)
	if err != nil {
		return err
	}

	// Insert audit record
	queryAudit := `
		INSERT INTO audit_log (tabelle, aktion, datensatz_id, bearbeiter_id)
		VALUES ($1, $2, $3, $4)
	`
	_, err = tx.Exec(ctx, queryAudit, "benutzer", "DELETE", userID, bearbeiterID)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}
