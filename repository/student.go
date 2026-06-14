package repository

import (
	"bibliothek/db"
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	
)

// Scanner defines an interface for both pgx.Row and pgx.Rows to enable shared scan helpers.
type Scanner interface {
	Scan(dest ...any) error
}

// StudentRepository defines operations for fetching student records from the database.
type StudentRepository interface {
	// GetByBarcode fetches a student by their unique barcode identifier. Returns nil if not found.
	GetByBarcode(ctx context.Context, barcode string) (*Student, error)
	// GetByID fetches a student by their primary key UUID. Returns nil if not found.
	GetByID(ctx context.Context, id string) (*Student, error)
	// SearchStudentsFuzzy performs a fuzzy search on students.
	SearchStudentsFuzzy(ctx context.Context, queryText string, limit int) ([]Student, error)
	// GetNextSequence gets the next student barcode sequence number.
	GetNextSequence(ctx context.Context) (int, error)
	// GetAllLUSDStudents fetches essential data for LUSD diffing.
	GetAllLUSDStudents(ctx context.Context) ([]Student, error)
	// BulkSyncLUSD performs the mass update/insert and marks graduates.
	BulkSyncLUSD(ctx context.Context, updates []StudentUpdate, inserts []StudentInsert, allLusdIDs []string) (int, error)
}

type StudentUpdate struct {
	ID           string
	Vorname      string
	Nachname     string
	Klasse       string
	Geburtsdatum *string // Format: YYYY-MM-DD
	LusdID       *string
}

type StudentInsert struct {
	BarcodeID     string
	Vorname       string
	Nachname      string
	Klasse        string
	Geburtsdatum  *string // Format: YYYY-MM-DD
	AbgaengerJahr int
	LusdID        *string
	IstAbgaenger  bool
}

type pgStudentRepository struct {
	db db.PgxPoolIface
}

// NewStudentRepository builds a PostgreSQL-backed StudentRepository.
func NewStudentRepository(db db.PgxPoolIface) StudentRepository {
	return &pgStudentRepository{db: db}
}

func scanStudent(row Scanner) (*Student, error) {
	var s Student
	err := row.Scan(
		&s.ID, &s.BarcodeID, &s.Vorname, &s.Nachname, &s.Klasse, &s.AbgaengerJahr, &s.IstGesperrt, &s.LusdID, &s.IstAbgaenger, &s.Geburtsdatum, &s.ErstelltAm, &s.AktualisiertAm,
	)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// GetByBarcode fetches a student by barcode.
func (r *pgStudentRepository) GetByBarcode(ctx context.Context, barcode string) (*Student, error) {
	query := `
		SELECT id, barcode_id, vorname, nachname, klasse, abgaenger_jahr, ist_gesperrt, lusd_id, ist_abgaenger, TO_CHAR(geburtsdatum, 'YYYY-MM-DD'), erstellt_am, aktualisiert_am
		FROM schueler
		WHERE barcode_id = $1
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

// GetByID fetches a student by ID.
func (r *pgStudentRepository) GetByID(ctx context.Context, id string) (*Student, error) {
	query := `
		SELECT id, barcode_id, vorname, nachname, klasse, abgaenger_jahr, ist_gesperrt, lusd_id, ist_abgaenger, TO_CHAR(geburtsdatum, 'YYYY-MM-DD'), erstellt_am, aktualisiert_am
		FROM schueler
		WHERE id = $1
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

// GetNextSequence retrieves the next barcode sequence integer.
func (r *pgStudentRepository) GetNextSequence(ctx context.Context) (int, error) {
	var lastBarcode string
	qLast := `
		SELECT barcode_id 
		FROM schueler 
		WHERE barcode_id LIKE 'S-%' 
		ORDER BY barcode_id DESC 
		LIMIT 1
	`
	err := r.db.QueryRow(ctx, qLast).Scan(&lastBarcode)
	startNum := 10001
	if err == nil && len(lastBarcode) > 2 {
		if parsed, err2 := strconv.Atoi(lastBarcode[2:]); err2 == nil {
			startNum = parsed + 1
		}
	}
	return startNum, nil
}

// GetAllLUSDStudents fetches essential fields for diffing.
func (r *pgStudentRepository) GetAllLUSDStudents(ctx context.Context) ([]Student, error) {
	rows, err := r.db.Query(ctx, "SELECT id, lusd_id, lower(vorname), lower(nachname), coalesce(geburtsdatum, '1900-01-01'::DATE) FROM schueler")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []Student
	for rows.Next() {
		var s Student
		var geb time.Time
		if err := rows.Scan(&s.ID, &s.LusdID, &s.Vorname, &s.Nachname, &geb); err == nil {
			dateStr := geb.Format("2006-01-02")
			s.Geburtsdatum = &dateStr
			results = append(results, s)
		}
	}
	return results, rows.Err()
}

// BulkSyncLUSD performs the batch operations in a single transaction.
func (r *pgStudentRepository) BulkSyncLUSD(ctx context.Context, updates []StudentUpdate, inserts []StudentInsert, allLusdIDs []string) (int, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if len(updates) > 0 {
		var updID, updVorname, updNach, updKlasse []string
		var updGeb, updLusd []*string

		for _, u := range updates {
			updID = append(updID, u.ID)
			updVorname = append(updVorname, u.Vorname)
			updNach = append(updNach, u.Nachname)
			updKlasse = append(updKlasse, u.Klasse)
			updGeb = append(updGeb, u.Geburtsdatum)
			updLusd = append(updLusd, u.LusdID)
		}

		qUpdate := `
			UPDATE schueler s
			SET vorname = d.vorname,
				nachname = d.nachname,
				klasse = d.klasse,
				geburtsdatum = d.geburtsdatum::date,
				ist_abgaenger = false,
				aktualisiert_am = CURRENT_TIMESTAMP,
				lusd_id = COALESCE(d.lusd_id, s.lusd_id)
			FROM (
				SELECT * FROM UNNEST($1::uuid[], $2::varchar[], $3::varchar[], $4::varchar[], $5::varchar[], $6::varchar[])
				AS u(id, vorname, nachname, klasse, geburtsdatum, lusd_id)
			) d
			WHERE s.id = d.id
		`
		_, err = tx.Exec(ctx, qUpdate, updID, updVorname, updNach, updKlasse, updGeb, updLusd)
		if err != nil {
			return 0, err
		}
	}

	if len(inserts) > 0 {
		var copyRows [][]any
		for _, i := range inserts {
			var geb any = nil
			if i.Geburtsdatum != nil {
				geb = *i.Geburtsdatum
			}
			copyRows = append(copyRows, []any{
				i.BarcodeID, i.Vorname, i.Nachname, i.Klasse, geb, i.AbgaengerJahr, i.LusdID, i.IstAbgaenger,
			})
		}
		// PR 89: Use pgx.CopyFromRows for massive performance gains in inserts
		_, err = tx.CopyFrom(
			ctx,
			pgx.Identifier{"schueler"},
			[]string{"barcode_id", "vorname", "nachname", "klasse", "geburtsdatum", "abgaenger_jahr", "lusd_id", "ist_abgaenger"},
			pgx.CopyFromRows(copyRows),
		)
		if err != nil {
			return 0, err
		}
	}

	qMarkAbgaenger := `
		UPDATE schueler
		SET ist_abgaenger = true, 
		    abgaenger_jahr = EXTRACT(YEAR FROM CURRENT_DATE),
		    aktualisiert_am = CURRENT_TIMESTAMP
		WHERE lusd_id IS NOT NULL AND NOT (lusd_id = ANY($1)) AND ist_abgaenger = false
	`
	_, err = tx.Exec(ctx, qMarkAbgaenger, allLusdIDs)
	if err != nil {
		return 0, err
	}

	qDeleteVormerkungen := `
		DELETE FROM vormerkungen 
		WHERE schueler_id IN (
			SELECT id FROM schueler WHERE ist_abgaenger = true
		)
	`
	_, err = tx.Exec(ctx, qDeleteVormerkungen)
	if err != nil {
		return 0, err
	}

	var abgaengerOpenCount int
	qCountLoans := `
		SELECT COUNT(DISTINCT schueler_id)
		FROM ausleihen
		WHERE rueckgabe_am IS NULL 
		  AND schueler_id IN (
			  SELECT id FROM schueler WHERE ist_abgaenger = true
		  )
	`
	err = tx.QueryRow(ctx, qCountLoans).Scan(&abgaengerOpenCount)
	if err != nil {
		return 0, err
	}

	return abgaengerOpenCount, tx.Commit(ctx)
}
