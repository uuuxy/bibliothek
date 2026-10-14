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

// ErsatzwertVorschlagHandler liefert den Betragsvorschlag für ein Exemplar.
//
// @Summary      Suggest a replacement amount for a copy
// @Tags         schadensersatz
// @Produce      json
// @Param        id path string true "Copy ID"
// @Success      200 {object} service.ErsatzwertVorschlag
// @Failure      404 {object} apierrors.APIError
// @Router       /buecher/exemplare/{id}/ersatzwert-vorschlag [get]
func (s *Tueren) ErsatzwertVorschlagHandler(bescheidRepo repository.BescheidAbfragen) http.HandlerFunc {
	return apierrors.Wrap(func(w http.ResponseWriter, r *http.Request) error {
		id := r.PathValue("id")
		if id == "" {
			return apierrors.BadRequest(tuer.MeldungIDLeer, errors.New("missing id"))
		}

		groessen, err := bescheidRepo.LadeGroessenFuerExemplar(r.Context(), id)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return apierrors.NotFound("Exemplar nicht gefunden", err)
			}
			return apierrors.Internal("Ersatzwert konnte nicht berechnet werden", err)
		}

		tuer.RespondJSON(w, http.StatusOK, service.ErsatzwertVorschlagAus(groessen, service.LadeErsatzwertPreisquelle(r.Context(), s.DB.Pool)))
		return nil
	})
}
