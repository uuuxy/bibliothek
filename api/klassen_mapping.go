package api

import (
	"errors"
	"net/http"
	"time"

	"bibliothek/apierrors"
)

// KlassenLehrerMapping associates a class with the class teacher's e-mail address.
type KlassenLehrerMapping struct {
	Klasse      string `json:"klasse"`
	LehrerEmail string `json:"lehrer_email"`
	ErstelltAm  string `json:"erstellt_am,omitempty"`
}

// GetKlassenMappingHandler returns all class → teacher-e-mail mappings.
// GET /api/klassen-mapping
func (s *Server) GetKlassenMappingHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		rows, err := s.DB.Pool.Query(ctx,
			`SELECT klasse, lehrer_email, erstellt_am FROM klassen_lehrer_mapping ORDER BY klasse`)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		defer rows.Close()

		mappings := []KlassenLehrerMapping{}
		for rows.Next() {
			var m KlassenLehrerMapping
			var t time.Time
			if err := rows.Scan(&m.Klasse, &m.LehrerEmail, &t); err != nil {
				continue
			}
			m.ErstelltAm = t.Format("2006-01-02")
			mappings = append(mappings, m)
		}

		RespondJSON(w, http.StatusOK, mappings)
	}
}

// UpsertKlassenMappingHandler creates or updates a class → teacher-e-mail mapping.
// POST /api/klassen-mapping  { "klasse": "5b", "lehrer_email": "..." }
func (s *Server) UpsertKlassenMappingHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req KlassenLehrerMapping
		if !DecodeJSON(w, r, &req) {
			return
		}
		if req.Klasse == "" || req.LehrerEmail == "" {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("klasse und lehrer_email sind erforderlich"))
			return
		}

		ctx := r.Context()

		_, err := s.DB.Pool.Exec(ctx, `
			INSERT INTO klassen_lehrer_mapping (klasse, lehrer_email)
			VALUES ($1, $2)
			ON CONFLICT (klasse) DO UPDATE SET lehrer_email = EXCLUDED.lehrer_email
		`, req.Klasse, req.LehrerEmail)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		RespondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}
}

// DeleteKlassenMappingHandler removes a class → teacher-e-mail mapping.
// DELETE /api/klassen-mapping/{klasse}
func (s *Server) DeleteKlassenMappingHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		klasse := r.PathValue("klasse")
		if klasse == "" {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("klasse erforderlich"))
			return
		}

		ctx := r.Context()

		_, err := s.DB.Pool.Exec(ctx,
			`DELETE FROM klassen_lehrer_mapping WHERE klasse = $1`, klasse)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
