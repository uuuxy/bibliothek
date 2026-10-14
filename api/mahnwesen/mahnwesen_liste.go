package mahnwesen

import (
	"log"
	"net/http"

	"bibliothek/api/tuer"
	"bibliothek/apierrors"
	"bibliothek/repository"
)

// GetMahnwesenHandler returns overdue loans grouped by class and student.
// GET /api/mahnwesen
func (s *Tueren) GetMahnwesenHandler(mahnRepo *repository.MahnwesenAbfragen) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		klassen, err := mahnRepo.LadeUeberfaelligeNachKlasse(ctx, "")
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		heuteRetourniert, err := mahnRepo.ZaehleRueckgabenHeute(ctx)
		if err != nil {
			log.Printf("mahnwesen: Anzahl heutiger Rückgaben konnte nicht ermittelt werden: %v", err)
		}

		tuer.AntworteJSON(w, http.StatusOK, map[string]any{
			"klassen":           klassen,
			"heute_retourniert": heuteRetourniert,
		})
	}
}
