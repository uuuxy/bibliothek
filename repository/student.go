package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// StudentRepository defines operations for fetching student records from the database.
type StudentRepository interface {
	// GetByBarcode fetches a student by their unique barcode identifier. Returns nil if not found.
	GetByBarcode(ctx context.Context, barcode string) (*Student, error)
	// GetByID fetches a student by their primary key UUID. Returns nil if not found.
	GetByID(ctx context.Context, id string) (*Student, error)
	// SearchStudentsFuzzy performs a fuzzy search on students.
	SearchStudentsFuzzy(ctx context.Context, queryText string, limit int) ([]Student, error)
}

type pgStudentRepository struct {
	db *pgxpool.Pool
}

// NewStudentRepository builds a PostgreSQL-backed StudentRepository.
func NewStudentRepository(db *pgxpool.Pool) StudentRepository {
	return &pgStudentRepository{db: db}
}

// GetByBarcode fetches a student by barcode.
func (r *pgStudentRepository) GetByBarcode(ctx context.Context, barcode string) (*Student, error) {
	query := `
		SELECT id, barcode_id, vorname, nachname, klasse, abgaenger_jahr, ist_gesperrt, lusd_id, ist_abgaenger, TO_CHAR(geburtsdatum, 'YYYY-MM-DD'), erstellt_am, aktualisiert_am
		FROM schueler
		WHERE barcode_id = $1
		LIMIT 1
	`
	var s Student
	err := r.db.QueryRow(ctx, query, barcode).Scan(
		&s.ID, &s.BarcodeID, &s.Vorname, &s.Nachname, &s.Klasse, &s.AbgaengerJahr, &s.IstGesperrt, &s.LusdID, &s.IstAbgaenger, &s.Geburtsdatum, &s.ErstelltAm, &s.AktualisiertAm,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &s, nil
}

// GetByID fetches a student by ID.
func (r *pgStudentRepository) GetByID(ctx context.Context, id string) (*Student, error) {
	query := `
		SELECT id, barcode_id, vorname, nachname, klasse, abgaenger_jahr, ist_gesperrt, lusd_id, ist_abgaenger, TO_CHAR(geburtsdatum, 'YYYY-MM-DD'), erstellt_am, aktualisiert_am
		FROM schueler
		WHERE id = $1
		LIMIT 1
	`
	var s Student
	err := r.db.QueryRow(ctx, query, id).Scan(
		&s.ID, &s.BarcodeID, &s.Vorname, &s.Nachname, &s.Klasse, &s.AbgaengerJahr, &s.IstGesperrt, &s.LusdID, &s.IstAbgaenger, &s.Geburtsdatum, &s.ErstelltAm, &s.AktualisiertAm,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &s, nil
}

// SearchStudentsFuzzy performs a fuzzy search on student names.
func (r *pgStudentRepository) SearchStudentsFuzzy(ctx context.Context, queryText string, limit int) ([]Student, error) {
	query := `
		SELECT id, barcode_id, vorname, nachname, klasse, abgaenger_jahr, ist_gesperrt, lusd_id, ist_abgaenger, TO_CHAR(geburtsdatum, 'YYYY-MM-DD'), erstellt_am, aktualisiert_am
		FROM schueler
		WHERE vorname ILIKE '%' || $1 || '%' 
		   OR nachname ILIKE '%' || $1 || '%'
		   OR barcode_id ILIKE '%' || $1 || '%'
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
		var s Student
		err := rows.Scan(
			&s.ID, &s.BarcodeID, &s.Vorname, &s.Nachname, &s.Klasse, &s.AbgaengerJahr, &s.IstGesperrt, &s.LusdID, &s.IstAbgaenger, &s.Geburtsdatum, &s.ErstelltAm, &s.AktualisiertAm,
		)
		if err != nil {
			return nil, err
		}
		results = append(results, s)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return results, nil
}
