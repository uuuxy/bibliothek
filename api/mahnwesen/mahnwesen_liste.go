package mahnwesen

import (
	"log"
	"net/http"

	"bibliothek/api/tuer"
	"bibliothek/apierrors"
	"bibliothek/repository"
)

// UeberfaelligeAusleihen returns overdue loans grouped by class and student.
// GET /api/mahnwesen
func (t *Tueren) UeberfaelligeAusleihen(mahnAbfragen *repository.MahnwesenAbfragen) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		klassen, err := mahnAbfragen.LadeUeberfaelligeNachKlasse(ctx, "")
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		heuteRetourniert, err := mahnAbfragen.ZaehleRueckgabenHeute(ctx)
		if err != nil {
			log.Printf("mahnwesen: Anzahl heutiger Rückgaben konnte nicht ermittelt werden: %v", err)
		}

		tuer.AntworteJSON(w, http.StatusOK, map[string]any{
			"klassen":           klassen,
			"heute_retourniert": heuteRetourniert,
		})
	}
}
