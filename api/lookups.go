package api

import (
	"bibliothek/apierrors"
	"errors"

	"net/http"
)

// GetSystematicsHandler returns all entries from systematik_kategorien
func (s *Server) GetSystematicsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		rows, err := s.DB.Pool.Query(ctx, "SELECT id, kuerzel, bezeichnung FROM systematik_kategorien ORDER BY bezeichnung ASC")
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, errors.New("database error"))
			return
		}
		defer rows.Close()

		type Systematik struct {
			ID          string `json:"id"`
			Kuerzel     string `json:"kuerzel"`
			Bezeichnung string `json:"bezeichnung"`
		}
		var results []Systematik

		for rows.Next() {
			var sys Systematik
			if err := rows.Scan(&sys.ID, &sys.Kuerzel, &sys.Bezeichnung); err == nil {
				results = append(results, sys)
			}
		}
		if err := rows.Err(); err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, errors.New("database error"))
			return
		}

		RespondJSON(w, http.StatusOK, results)
	}
}

// GetFaecherHandler liefert die distinkten Fächer (buecher_titel.subject) — für die
// Fach-Auswahl beim gezielten Inventur-Scope ("nur Mathe, Klasse 5").
func (s *Server) GetFaecherHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		rows, err := s.DB.Pool.Query(ctx, `
			SELECT DISTINCT btrim(subject)
			FROM buecher_titel
			WHERE subject IS NOT NULL AND btrim(subject) <> ''
			ORDER BY 1
		`)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, errors.New("database error"))
			return
		}
		defer rows.Close()

		faecher := []string{}
		for rows.Next() {
			var f string
			if err := rows.Scan(&f); err == nil {
				faecher = append(faecher, f)
			}
		}
		if err := rows.Err(); err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, errors.New("database error"))
			return
		}
		RespondJSON(w, http.StatusOK, faecher)
	}
}

// GetReaderGroupsHandler returns all entries from lesergruppen
func (s *Server) GetReaderGroupsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		rows, err := s.DB.Pool.Query(ctx, "SELECT id, kuerzel, bezeichnung FROM lesergruppen ORDER BY bezeichnung ASC")
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, errors.New("database error"))
			return
		}
		defer rows.Close()

		type ReaderGroup struct {
			ID          string `json:"id"`
			Kuerzel     string `json:"kuerzel"`
			Bezeichnung string `json:"bezeichnung"`
		}
		var results []ReaderGroup

		for rows.Next() {
			var rg ReaderGroup
			if err := rows.Scan(&rg.ID, &rg.Kuerzel, &rg.Bezeichnung); err == nil {
				results = append(results, rg)
			}
		}
		if err := rows.Err(); err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, errors.New("database error"))
			return
		}

		RespondJSON(w, http.StatusOK, results)
	}
}
