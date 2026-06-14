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
			apierrors.SendHTTPError(w, http.StatusInternalServerError, errors.New("Database error"))
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

		RespondJSON(w, http.StatusOK, results)
	}
}

// GetReaderGroupsHandler returns all entries from lesergruppen
func (s *Server) GetReaderGroupsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		rows, err := s.DB.Pool.Query(ctx, "SELECT id, kuerzel, bezeichnung FROM lesergruppen ORDER BY bezeichnung ASC")
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, errors.New("Database error"))
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

		RespondJSON(w, http.StatusOK, results)
	}
}
