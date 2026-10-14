package schaden

import (
	"errors"
	"net/http"

	"bibliothek/api/tuer"
	"bibliothek/apierrors"
	"bibliothek/internal/service"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5"
)

// ErsatzwertVorschlag liefert den Betragsvorschlag für ein Exemplar.
//
// @Summary      Betragsvorschlag für ein Exemplar
// @Tags         schadensersatz
// @Produce      json
// @Param        id path string true "Kennung des Exemplars"
// @Success      200 {object} service.ErsatzwertVorschlag
// @Failure      404 {object} apierrors.APIError
// @Router       /buecher/exemplare/{id}/ersatzwert-vorschlag [get]
func (t *Tueren) ErsatzwertVorschlag(bescheidAbfragen repository.BescheidAbfragen) http.HandlerFunc {
	return apierrors.Wrap(func(w http.ResponseWriter, r *http.Request) error {
		id := r.PathValue("id")
		if id == "" {
			return apierrors.BadRequest(tuer.MeldungIDLeer, errors.New("missing id"))
		}

		groessen, err := bescheidAbfragen.LadeGroessenFuerExemplar(r.Context(), id)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return apierrors.NotFound("Exemplar nicht gefunden", err)
			}
			return apierrors.Internal("Ersatzwert konnte nicht berechnet werden", err)
		}

		tuer.AntworteJSON(w, http.StatusOK, service.ErsatzwertVorschlagAus(groessen, service.LadeErsatzwertPreisquelle(r.Context(), t.DB.Pool)))
		return nil
	})
}
