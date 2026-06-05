package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"bibliothek/apierrors"
	"bibliothek/auth"

	"github.com/jackc/pgx/v5"
)

// KlassensatzReservierungRequest is the payload for a class-set reservation.
type KlassensatzReservierungRequest struct {
	TitelID string `json:"titel_id"`
	Klasse  string `json:"klasse"`
	Anzahl  int    `json:"anzahl"`
	Notiz   string `json:"notiz,omitempty"`
}

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

// CreateKlassensatzReservierungHandler lets a LEHRER submit a class-set reservation.
// POST /api/reservierungen/klassensatz
func (s *Server) CreateKlassensatzReservierungHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req KlassensatzReservierungRequest
		if !apierrors.DecodeJSONRequest(w, r, &req) {
			return
		}
		if req.TitelID == "" || req.Klasse == "" {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("titel_id und klasse sind erforderlich"))
			return
		}
		if req.Anzahl <= 0 {
			req.Anzahl = 1
		}
		if req.Anzahl > 200 {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("anzahl darf 200 nicht überschreiten"))
			return
		}

		claims, ok := auth.GetClaims(r.Context())
		if !ok {
			apierrors.SendHTTPError(w, http.StatusUnauthorized, errors.New("fehlende Sitzungsinformationen"))
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		// Verify the title exists.
		var exists bool
		if err := s.DB.Pool.QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM buecher_titel WHERE id = $1)`, req.TitelID,
		).Scan(&exists); err != nil || !exists {
			apierrors.SendHTTPError(w, http.StatusNotFound, errors.New("buchtitel nicht gefunden"))
			return
		}

		var newID string
		err := s.DB.Pool.QueryRow(ctx, `
			INSERT INTO klassensatz_reservierungen
				(titel_id, klasse, anzahl, notiz, angefordert_von)
			VALUES ($1, $2, $3, $4, $5)
			RETURNING id
		`, req.TitelID, req.Klasse, req.Anzahl,
			nullableString(req.Notiz), claims.UserID,
		).Scan(&newID)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]string{"id": newID, "status": "erstellt"})
	}
}

// GetKlassensatzReservierungenHandler lists all pending class-set reservations for admins.
// GET /api/reservierungen/klassensatz
func (s *Server) GetKlassensatzReservierungenHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		rows, err := s.DB.Pool.Query(ctx, `
			SELECT r.id, r.titel_id, t.titel, coalesce(t.cover_url,''),
			       r.klasse, r.anzahl, r.notiz, r.erledigt, r.erstellt_am
			FROM klassensatz_reservierungen r
			JOIN buecher_titel t ON r.titel_id = t.id
			ORDER BY r.erledigt ASC, r.erstellt_am DESC
		`)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		defer rows.Close()

		result := []KlassensatzReservierung{}
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

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(result)
	}
}

// GetKlassensatzReservierungenAnzahlHandler returns the count of open reservations (for red badge).
// GET /api/reservierungen/klassensatz/anzahl
func (s *Server) GetKlassensatzReservierungenAnzahlHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		var count int
		_ = s.DB.Pool.QueryRow(ctx,
			`SELECT COUNT(*) FROM klassensatz_reservierungen WHERE erledigt = false`,
		).Scan(&count)

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]int{"anzahl": count})
	}
}

// ErledigeKlassensatzReservierungHandler marks a class-set reservation as done.
// PUT /api/reservierungen/klassensatz/{id}/erledigen
func (s *Server) ErledigeKlassensatzReservierungHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if id == "" {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("id fehlt"))
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		tag, err := s.DB.Pool.Exec(ctx,
			`UPDATE klassensatz_reservierungen SET erledigt = true WHERE id = $1`, id)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		if tag.RowsAffected() == 0 {
			apierrors.SendHTTPError(w, http.StatusNotFound, pgx.ErrNoRows)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

// nullableString converts an empty string to nil for nullable DB columns.
func nullableString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
