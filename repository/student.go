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
		SELECT id, barcode_id, vorname, nachname, klasse, abgaenger_jahr, ist_gesperrt, erstellt_am, aktualisiert_am
		FROM schueler
		WHERE barcode_id = $1
		LIMIT 1
	`
	var s Student
	err := r.db.QueryRow(ctx, query, barcode).Scan(
		&s.ID, &s.BarcodeID, &s.Vorname, &s.Nachname, &s.Klasse, &s.AbgaengerJahr, &s.IstGesperrt, &s.ErstelltAm, &s.AktualisiertAm,
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
		SELECT id, barcode_id, vorname, nachname, klasse, abgaenger_jahr, ist_gesperrt, erstellt_am, aktualisiert_am
		FROM schueler
		WHERE id = $1
		LIMIT 1
	`
	var s Student
	err := r.db.QueryRow(ctx, query, id).Scan(
		&s.ID, &s.BarcodeID, &s.Vorname, &s.Nachname, &s.Klasse, &s.AbgaengerJahr, &s.IstGesperrt, &s.ErstelltAm, &s.AktualisiertAm,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &s, nil
}
