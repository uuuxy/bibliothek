package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"bibliothek/apierrors"

	"github.com/jackc/pgx/v5"
)

// Vormerkung represents a pending book reservation entry.
type Vormerkung struct {
	ID           string    `json:"id"`
	TitelID      string    `json:"titel_id"`
	TitelName    string    `json:"titel"`
	Notiz        string    `json:"notiz,omitempty"`
	ErstelltAm   time.Time `json:"erstellt_am"`
	SchuelerID   string    `json:"schueler_id,omitempty"`
	SchuelerName string    `json:"schueler_name,omitempty"`
}

// CreateVormerkungRequest is the body for POST /api/vormerkungen.
type CreateVormerkungRequest struct {
	TitelID    string `json:"titel_id"`
	Notiz      string `json:"notiz,omitempty"`
	SchuelerID string `json:"schueler_id,omitempty"`
}

// ListVormerkungHandler handles GET /api/vormerkungen?titel_id=...
func (s *Server) ListVormerkungHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		titelID := r.URL.Query().Get("titel_id")
		var rows pgx.Rows
		var err error
		if titelID != "" {
			rows, err = s.DB.Pool.Query(ctx, `
				SELECT v.id, v.titel_id, bt.titel, COALESCE(v.notiz, ''), v.erstellt_am,
				       COALESCE(s.id::text, ''), COALESCE(s.vorname || ' ' || s.nachname || ', ' || s.klasse, '')
				FROM vormerkungen v
				JOIN buecher_titel bt ON bt.id = v.titel_id
				LEFT JOIN schueler s ON s.id = v.schueler_id
				WHERE v.titel_id = $1
				ORDER BY v.erstellt_am ASC
			`, titelID)
		} else {
			rows, err = s.DB.Pool.Query(ctx, `
				SELECT v.id, v.titel_id, bt.titel, COALESCE(v.notiz, ''), v.erstellt_am,
				       COALESCE(s.id::text, ''), COALESCE(s.vorname || ' ' || s.nachname || ', ' || s.klasse, '')
				FROM vormerkungen v
				JOIN buecher_titel bt ON bt.id = v.titel_id
				LEFT JOIN schueler s ON s.id = v.schueler_id
				ORDER BY v.erstellt_am ASC
			`)
		}
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		defer rows.Close()

		result := make([]Vormerkung, 0)
		for rows.Next() {
			var v Vormerkung
			if err := rows.Scan(&v.ID, &v.TitelID, &v.TitelName, &v.Notiz, &v.ErstelltAm, &v.SchuelerID, &v.SchuelerName); err != nil {
				apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
				return
			}
			result = append(result, v)
		}
		if result == nil {
			result = []Vormerkung{}
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(result)
	}
}

// CreateVormerkungHandler handles POST /api/vormerkungen.
func (s *Server) CreateVormerkungHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req CreateVormerkungRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.TitelID == "" {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("titel_id ist erforderlich"))
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		var id string
		err := s.DB.Pool.QueryRow(ctx, `
			INSERT INTO vormerkungen (titel_id, notiz, schueler_id)
			VALUES ($1, NULLIF($2, ''), NULLIF($3, '')::uuid)
			RETURNING id
		`, req.TitelID, req.Notiz, req.SchuelerID).Scan(&id)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]string{"id": id})
	}
}

// DeleteVormerkungHandler handles DELETE /api/vormerkungen/{id}.
func (s *Server) DeleteVormerkungHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if id == "" {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("ID fehlt"))
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		if _, err := s.DB.Pool.Exec(ctx, `DELETE FROM vormerkungen WHERE id = $1`, id); err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"gelöscht"}`))
	}
}

// checkVormerkung returns the earliest pending reservation for a given titel_id, or nil if none.
func (s *Server) checkVormerkung(ctx context.Context, titelID string) (*Vormerkung, error) {
	var v Vormerkung
	err := s.DB.Pool.QueryRow(ctx, `
		SELECT v.id, v.titel_id, bt.titel, COALESCE(v.notiz, ''), v.erstellt_am,
		       COALESCE(s.id::text, ''), COALESCE(s.vorname || ' ' || s.nachname || ', ' || s.klasse, '')
		FROM vormerkungen v
		JOIN buecher_titel bt ON bt.id = v.titel_id
		LEFT JOIN schueler s ON s.id = v.schueler_id
		WHERE v.titel_id = $1
		ORDER BY v.erstellt_am ASC
		LIMIT 1
	`, titelID).Scan(&v.ID, &v.TitelID, &v.TitelName, &v.Notiz, &v.ErstelltAm, &v.SchuelerID, &v.SchuelerName)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return &v, err
}
