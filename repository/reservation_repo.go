package repository

import (
	"context"
	"time"

	"bibliothek/db"
)

// KlassensatzReservierung represents a pending class-set reservation.
type KlassensatzReservierung struct {
	ID             string  `json:"id"`
	TitelID        string  `json:"titel_id"`
	TitelName      string  `json:"titel_name"`
	CoverURL       string  `json:"cover_url,omitempty"`
	Klasse         string  `json:"klasse"`
	Anzahl         int     `json:"anzahl"`
	Notiz          *string `json:"notiz,omitempty"`
	AngefordertVon *string `json:"angefordert_von,omitempty"`
	Erledigt       bool    `json:"erledigt"`
	ErstelltAm     string  `json:"erstellt_am"`
}

// ReservationRepository kapselt die Datenbankzugriffe für Reservierungen.
type ReservationRepository interface {
	CheckTitleExists(ctx context.Context, titelID string) (bool, error)
	CreateKlassensatzReservierung(ctx context.Context, titelID, klasse string, anzahl int, notiz *string, angefordertVon string) (string, error)
	GetKlassensatzReservierungen(ctx context.Context) ([]KlassensatzReservierung, error)
	GetKlassensatzReservierungenAnzahl(ctx context.Context) (int, error)
	ErledigeKlassensatzReservierung(ctx context.Context, id string) (int64, error)
}

type pgReservationRepository struct {
	db db.PgxPoolIface
}

func NewReservationRepository(pool db.PgxPoolIface) ReservationRepository {
	return &pgReservationRepository{db: pool}
}

func (r *pgReservationRepository) CheckTitleExists(ctx context.Context, titelID string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM buecher_titel WHERE id = $1)`, titelID).Scan(&exists)
	return exists, err
}

func (r *pgReservationRepository) CreateKlassensatzReservierung(ctx context.Context, titelID, klasse string, anzahl int, notiz *string, angefordertVon string) (string, error) {
	var newID string
	err := r.db.QueryRow(ctx, `
		INSERT INTO klassensatz_reservierungen
			(titel_id, klasse, anzahl, notiz, angefordert_von)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`, titelID, klasse, anzahl, notiz, angefordertVon).Scan(&newID)
	return newID, err
}

func (r *pgReservationRepository) GetKlassensatzReservierungen(ctx context.Context) ([]KlassensatzReservierung, error) {
	rows, err := r.db.Query(ctx, `
		SELECT r.id, r.titel_id, t.titel, coalesce(t.cover_url,''),
		       r.klasse, r.anzahl, r.notiz, r.erledigt, r.erstellt_am
		FROM klassensatz_reservierungen r
		JOIN buecher_titel t ON r.titel_id = t.id
		ORDER BY r.erledigt ASC, r.erstellt_am DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []KlassensatzReservierung
	for rows.Next() {
		var res KlassensatzReservierung
		var t time.Time
		if err := rows.Scan(
			&res.ID, &res.TitelID, &res.TitelName, &res.CoverURL,
			&res.Klasse, &res.Anzahl, &res.Notiz, &res.Erledigt, &t,
		); err != nil {
			continue
		}
		res.ErstelltAm = t.Format("02.01.2006")
		result = append(result, res)
	}
	return result, nil
}

func (r *pgReservationRepository) GetKlassensatzReservierungenAnzahl(ctx context.Context) (int, error) {
	var count int
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM klassensatz_reservierungen WHERE erledigt = false`).Scan(&count)
	return count, err
}

func (r *pgReservationRepository) ErledigeKlassensatzReservierung(ctx context.Context, id string) (int64, error) {
	tag, err := r.db.Exec(ctx, `UPDATE klassensatz_reservierungen SET erledigt = true WHERE id = $1`, id)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}
