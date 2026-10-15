package katalog

import (
	"errors"
	"net/http"

	"bibliothek/api/tuer"
	"bibliothek/apierrors"
	"bibliothek/auth"
	"bibliothek/repository"
)

// ExemplarAussondern sondert ein Exemplar aus: Die Zeile bleibt und gilt als ausgesondert, weil
// Ausleihen und Schadensfälle an ihr hängen (ProtokollAbfragen.SondereExemplarAus). Ein
// verliehenes Exemplar lehnt die Tür mit 400 ab.
//
// @Summary      Exemplar aussondern
// @Description  Sondert ein Exemplar aus (die Zeile bleibt, das Exemplar gilt als ausgesondert) und schreibt den Vorgang ins Protokoll. Ein verliehenes Exemplar ist 400.
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "Kennung des Exemplars (UUID)"
// @Success      200  {object}  map[string]string
// @Failure      400  {object}  map[string]string
// @Failure      401  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /buecher/exemplare/{id} [delete]
func (t *Tueren) ExemplarAussondern(protokollAbfragen repository.ProtokollAbfragen) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := auth.ClaimsAusKontext(r.Context())
		if !ok {
			apierrors.AntworteFehler(w, http.StatusUnauthorized, errors.New("missing session information"))
			return
		}

		id := r.PathValue("id")
		if id == "" {
			apierrors.AntworteFehler(w, http.StatusBadRequest, errors.New("missing copy ID parameter"))
			return
		}

		ctx := r.Context()

		err := protokollAbfragen.SondereExemplarAus(ctx, id, claims.KontoID)
		if err != nil {
			// Am Fehlerwert erkannt, nicht am Text der Meldung: Eine umformulierte Meldung machte
			// aus der Auskunft sonst einen Serverfehler.
			if errors.Is(err, repository.ErrExemplarNochVerliehen) {
				apierrors.AntworteFehler(w, http.StatusBadRequest, err)
				return
			}
			if errors.Is(err, repository.ErrExemplarNichtGefunden) {
				apierrors.AntworteFehler(w, http.StatusNotFound, err)
				return
			}
			apierrors.AntworteFehler(w, http.StatusInternalServerError, err)
			return
		}

		tuer.AntworteErfolg(w)
	}
}
