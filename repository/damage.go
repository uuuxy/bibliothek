package repository

import (
	"context"

	"bibliothek/db"

	"github.com/jackc/pgx/v5"
)

// DamageRepository defines operations for managing book damages and related loan actions.
type DamageRepository interface {
	MarkCopyDefekt(ctx context.Context, copyID string, loanID, schuelerID *string, benutzerID string, betrag float64, beschreibung string) (string, error)
	ReportDamage(ctx context.Context, copyID, loanID, schuelerID string, benutzerID string, beschreibung string, betrag float64) (string, error)
}

type pgDamageRepository struct {
	db db.PgxPoolIface
}

// NewDamageRepository returns a new PostgreSQL implementation of DamageRepository.
func NewDamageRepository(db db.PgxPoolIface) DamageRepository {
	return &pgDamageRepository{db: db}
}

// MarkCopyDefekt marks a book copy as defective and records a damage entry.
func (r *pgDamageRepository) MarkCopyDefekt(ctx context.Context, copyID string, loanID, schuelerID *string, benutzerID string, betrag float64, beschreibung string) (string, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer db.SafeRollback(ctx, tx)

	res, err := tx.Exec(ctx, `
		UPDATE buecher_exemplare
		SET ist_ausleihbar = false,
		    zustand_notiz = $1,
		    aktualisiert_am = CURRENT_TIMESTAMP
		WHERE id = $2
	`, beschreibung, copyID)
	if err != nil {
		return "", err
	}
	if res.RowsAffected() == 0 {
		return "", pgx.ErrNoRows
	}

	var schadensID string
	if schuelerID != nil && *schuelerID != "" {
		err = tx.QueryRow(ctx, `
			INSERT INTO schadensfaelle
			    (exemplar_id, ausleihe_id, schueler_id, beschreibung, betrag)
			VALUES ($1, $2, $3, $4, $5)
			RETURNING id
		`, copyID, loanID, schuelerID, beschreibung, betrag).Scan(&schadensID)
	} else {
		err = tx.QueryRow(ctx, `
			INSERT INTO schadensfaelle
			    (exemplar_id, ausleihe_id, benutzer_id, beschreibung, betrag)
			VALUES ($1, $2, $3, $4, $5)
			RETURNING id
		`, copyID, loanID, benutzerID, beschreibung, betrag).Scan(&schadensID)
	}
	if err != nil {
		return "", err
	}

	if err := tx.Commit(ctx); err != nil {
		return "", err
	}
	return schadensID, nil
}

// ReportDamage sets ist_ausgesondert = true, inserts a damage record, and ends the loan.
func (r *pgDamageRepository) ReportDamage(ctx context.Context, copyID, loanID, schuelerID string, benutzerID string, beschreibung string, betrag float64) (string, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer db.SafeRollback(ctx, tx)

	_, err = tx.Exec(ctx, `
		UPDATE buecher_exemplare
		SET ist_ausgesondert = true, ist_ausleihbar = false, aussonderung_grund = 'BESCHAEDIGUNG',
		    zustand_notiz = $1, aktualisiert_am = CURRENT_TIMESTAMP
		WHERE id = $2
	`, beschreibung, copyID)
	if err != nil {
		return "", err
	}

	var schadensID string
	err = tx.QueryRow(ctx, `
		INSERT INTO schadensfaelle (exemplar_id, ausleihe_id, schueler_id, beschreibung, betrag)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`, copyID, loanID, schuelerID, beschreibung, betrag).Scan(&schadensID)
	if err != nil {
		return "", err
	}

	_, err = tx.Exec(ctx, `
		UPDATE ausleihen
		SET rueckgabe_am = CURRENT_TIMESTAMP, rueckgabe_bearbeiter_id = $1
		WHERE id = $2
	`, benutzerID, loanID)
	if err != nil {
		return "", err
	}

	if err := tx.Commit(ctx); err != nil {
		return "", err
	}
	return schadensID, nil
}
