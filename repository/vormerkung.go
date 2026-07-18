package repository

import (
	"context"
	"time"

	"bibliothek/db"
	"github.com/jackc/pgx/v5"
)

// Vormerkung represents a pending book reservation entry for a student.
type Vormerkung struct {
	ID           string    `json:"id"`
	TitelID      string    `json:"titel_id"`
	TitelName    string    `json:"titel"`
	Notiz        string    `json:"notiz,omitempty"`
	ErstelltAm   time.Time `json:"erstellt_am"`
	SchuelerID   string    `json:"schueler_id,omitempty"`
	SchuelerName string    `json:"schueler_name,omitempty"`
}

// VormerkungRepository defines operations for managing individual book reservations.
type VormerkungRepository interface {
	List(ctx context.Context, titelID, schuelerID string) ([]Vormerkung, error)
	Create(ctx context.Context, titelID, notiz, schuelerID string) (string, error)
	Delete(ctx context.Context, id string) error
	GetEarliestPending(ctx context.Context, titelID string) (*Vormerkung, error)
}

type pgVormerkungRepository struct {
	db db.PgxPoolIface
}

// NewVormerkungRepository returns a new PostgreSQL implementation of VormerkungRepository.
func NewVormerkungRepository(db db.PgxPoolIface) VormerkungRepository {
	return &pgVormerkungRepository{db: db}
}

// List retrieves reservations filtered by either title or student.
func (r *pgVormerkungRepository) List(ctx context.Context, titelID, schuelerID string) ([]Vormerkung, error) {
	var rows pgx.Rows
	var err error

	if titelID != "" {
		rows, err = r.db.Query(ctx, `
			SELECT v.id, v.titel_id, bt.titel, COALESCE(v.notiz, ''), v.erstellt_am,
			       COALESCE(s.id::text, ''), COALESCE(s.vorname || ' ' || s.nachname || ', ' || s.klasse, '')
			FROM vormerkungen v
			JOIN buecher_titel bt ON bt.id = v.titel_id
			LEFT JOIN schueler s ON s.id = v.schueler_id
			WHERE v.titel_id = $1
			ORDER BY v.erstellt_am ASC
		`, titelID)
	} else if schuelerID != "" {
		rows, err = r.db.Query(ctx, `
			SELECT v.id, v.titel_id, bt.titel, COALESCE(v.notiz, ''), v.erstellt_am,
			       COALESCE(s.id::text, ''), COALESCE(s.vorname || ' ' || s.nachname || ', ' || s.klasse, '')
			FROM vormerkungen v
			JOIN buecher_titel bt ON bt.id = v.titel_id
			LEFT JOIN schueler s ON s.id = v.schueler_id
			WHERE v.schueler_id = $1
			ORDER BY v.erstellt_am ASC
		`, schuelerID)
	} else {
		rows, err = r.db.Query(ctx, `
			SELECT v.id, v.titel_id, bt.titel, COALESCE(v.notiz, ''), v.erstellt_am,
			       COALESCE(s.id::text, ''), COALESCE(s.vorname || ' ' || s.nachname || ', ' || s.klasse, '')
			FROM vormerkungen v
			JOIN buecher_titel bt ON bt.id = v.titel_id
			LEFT JOIN schueler s ON s.id = v.schueler_id
			ORDER BY v.erstellt_am ASC
		`)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []Vormerkung
	for rows.Next() {
		var v Vormerkung
		if err := rows.Scan(&v.ID, &v.TitelID, &v.TitelName, &v.Notiz, &v.ErstelltAm, &v.SchuelerID, &v.SchuelerName); err != nil {
			return nil, err
		}
		result = append(result, v)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

// Create creates a new reservation.
func (r *pgVormerkungRepository) Create(ctx context.Context, titelID, notiz, schuelerID string) (string, error) {
	var id string
	err := r.db.QueryRow(ctx, `
		INSERT INTO vormerkungen (titel_id, notiz, schueler_id)
		VALUES ($1, NULLIF($2, ''), NULLIF($3, '')::uuid)
		RETURNING id
	`, titelID, notiz, schuelerID).Scan(&id)
	return id, err
}

// Delete removes a reservation.
func (r *pgVormerkungRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM vormerkungen WHERE id = $1`, id)
	return err
}

// GetEarliestPending returns the earliest pending reservation of an eligible student for a
// given title. Nur abholberechtigte Schüler (nicht soft-gelöscht, nicht gesperrt) kommen in
// Frage — analog zum Live-Fulfillment in internal/service/loan_return.go, damit ein Buch
// nicht für einen gelöschten/gesperrten "Geister"-Schüler reserviert wird.
//
// Hinweis: aktuell ohne Produktionsaufrufer; der Fulfillment-Pfad läuft über den
// Service-Layer. Der Filter bleibt hier bewusst konsistent, falls die Methode wieder
// verdrahtet wird.
func (r *pgVormerkungRepository) GetEarliestPending(ctx context.Context, titelID string) (*Vormerkung, error) {
	var v Vormerkung
	err := r.db.QueryRow(ctx, `
		SELECT v.id, v.titel_id, bt.titel, COALESCE(v.notiz, ''), v.erstellt_am,
		       s.id::text, s.vorname || ' ' || s.nachname || ', ' || s.klasse
		FROM vormerkungen v
		JOIN buecher_titel bt ON bt.id = v.titel_id
		JOIN schueler s ON s.id = v.schueler_id
		WHERE v.titel_id = $1 AND v.status = 'wartend'
		  AND s.deleted_at IS NULL AND s.ist_gesperrt = false
		  AND COALESCE(s.is_manually_blocked, false) = false
		ORDER BY v.erstellt_am ASC
		LIMIT 1
	`, titelID).Scan(&v.ID, &v.TitelID, &v.TitelName, &v.Notiz, &v.ErstelltAm, &v.SchuelerID, &v.SchuelerName)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return &v, err
}
