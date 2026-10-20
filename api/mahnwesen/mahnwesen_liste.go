package mahnwesen

import (
	"net/http"

	"bibliothek/api/tuer"
	"bibliothek/apierrors"
	"bibliothek/repository"
)

// UeberfaelligeAusleihen liefert die überfälligen Ausleihen nach Klasse und Schüler.
// GET /api/mahnwesen
func (t *Tueren) UeberfaelligeAusleihen(mahnAbfragen *repository.MahnwesenAbfragen) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		klassen, err := mahnAbfragen.LadeUeberfaelligeNachKlasse(ctx, "")
		if err != nil {
			apierrors.AntworteFehler(w, http.StatusInternalServerError, err)
			return
		}

		tuer.AntworteJSON(w, http.StatusOK, map[string]any{
			"klassen": klassen,
		})
	}
}
