package api

import (
	"net/http"

	"bibliothek/apierrors"
	"bibliothek/repository"
)

// GetMahnwesenHandler returns overdue loans grouped by class and student.
// GET /api/mahnwesen
func (s *Server) GetMahnwesenHandler(mahnRepo *repository.MahnwesenRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		isFerien, ferienName, err := mahnRepo.CheckFerienAktiv(ctx)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		klassen, err := mahnRepo.QueryUeberfaelligeNachKlasse(ctx, "")
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		RespondJSON(w, http.StatusOK, map[string]any{
			"klassen":            klassen,
			"ferien_aktiv":       isFerien,
			"ferien_bezeichnung": ferienName,
		})
	}
}

// GetMahnwesenJahrgangHandler returns overdue loans based on grade level logic.
// GET /api/mahnwesen/ueberfaellig_jahrgang
func (s *Server) GetMahnwesenJahrgangHandler(mahnRepo *repository.MahnwesenRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		isFerien, ferienName, err := mahnRepo.CheckFerienAktiv(ctx)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		klassen, err := mahnRepo.QueryUeberfaelligeNachJahrgang(ctx, "")
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		RespondJSON(w, http.StatusOK, map[string]any{
			"klassen":            klassen,
			"ferien_aktiv":       isFerien,
			"ferien_bezeichnung": ferienName,
		})
	}
}
