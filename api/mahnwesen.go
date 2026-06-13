package api

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"bibliothek/apierrors"
	"bibliothek/repository"
)

// GetMahnwesenHandler returns overdue loans grouped by class and student.
// GET /api/mahnwesen
func (s *Server) GetMahnwesenHandler(mahnRepo *repository.MahnwesenRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
		defer cancel()

		klassen, err := mahnRepo.QueryUeberfaelligeNachKlasse(ctx, "")
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"klassen": klassen})
	}
}

// GetMahnwesenJahrgangHandler returns overdue loans based on grade level logic.
// GET /api/mahnwesen/ueberfaellig_jahrgang
func (s *Server) GetMahnwesenJahrgangHandler(mahnRepo *repository.MahnwesenRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
		defer cancel()

		klassen, err := mahnRepo.QueryUeberfaelligeNachJahrgang(ctx, "")
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"klassen": klassen})
	}
}
