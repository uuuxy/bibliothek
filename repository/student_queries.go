package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

// GetByBarcode liest einen Schüler anhand seiner Barcode-ID aus.
func (r *pgStudentRepository) GetByBarcode(ctx context.Context, barcode string) (*Student, error) {
	query := `
		SELECT id, coalesce(barcode_id, ''), coalesce(vorname, ''), coalesce(nachname, ''), coalesce(klasse, ''), coalesce(abgaenger_jahr, 0), coalesce(ist_gesperrt, false), lusd_id, coalesce(ist_abgaenger, false), TO_CHAR(geburtsdatum, 'YYYY-MM-DD'), erstellt_am, aktualisiert_am, coalesce(is_manually_blocked, false), block_reason, coalesce(strasse, ''), coalesce(hausnummer, ''), coalesce(plz, ''), coalesce(ort, ''), coalesce(eltern_email, '')
		FROM schueler
		WHERE barcode_id = $1 AND deleted_at IS NULL
		LIMIT 1
	`
	s, err := scanStudent(r.db.QueryRow(ctx, query, barcode))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return s, nil
}

// GetByID liest einen Schüler anhand seiner UUID aus.
func (r *pgStudentRepository) GetByID(ctx context.Context, id string) (*Student, error) {
	query := `
		SELECT id, coalesce(barcode_id, ''), coalesce(vorname, ''), coalesce(nachname, ''), coalesce(klasse, ''), coalesce(abgaenger_jahr, 0), coalesce(ist_gesperrt, false), lusd_id, coalesce(ist_abgaenger, false), TO_CHAR(geburtsdatum, 'YYYY-MM-DD'), erstellt_am, aktualisiert_am, coalesce(is_manually_blocked, false), block_reason, coalesce(strasse, ''), coalesce(hausnummer, ''), coalesce(plz, ''), coalesce(ort, ''), coalesce(eltern_email, '')
		FROM schueler
		WHERE id = $1 AND deleted_at IS NULL
		LIMIT 1
	`
	s, err := scanStudent(r.db.QueryRow(ctx, query, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return s, nil
}

// SearchStudentsFuzzy durchsucht die Schülerschaft nach Namen oder Barcodes.
func (r *pgStudentRepository) SearchStudentsFuzzy(ctx context.Context, queryText string, limit int) ([]Student, error) {
	query := `
		SELECT id, coalesce(barcode_id, ''), coalesce(vorname, ''), coalesce(nachname, ''), coalesce(klasse, ''), coalesce(abgaenger_jahr, 0), coalesce(ist_gesperrt, false), lusd_id, coalesce(ist_abgaenger, false), TO_CHAR(geburtsdatum, 'YYYY-MM-DD'), erstellt_am, aktualisiert_am, coalesce(is_manually_blocked, false), block_reason, coalesce(strasse, ''), coalesce(hausnummer, ''), coalesce(plz, ''), coalesce(ort, ''), coalesce(eltern_email, '')
		FROM schueler
		WHERE (vorname ILIKE '%' || $1::text || '%'
		   OR nachname ILIKE '%' || $1::text || '%'
		   OR barcode_id ILIKE '%' || $1::text || '%')
		  AND deleted_at IS NULL
		ORDER BY nachname ASC, vorname ASC
		LIMIT $2
	`
	rows, err := r.db.Query(ctx, query, queryText, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []Student
	for rows.Next() {
		s, err := scanStudent(rows)
		if err != nil {
			return nil, err
		}
		results = append(results, *s)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return results, nil
}
