package schaden

import (
	"errors"
	"net/http"

	"bibliothek/internal/api/tuer"
	"bibliothek/internal/apierrors"
	"bibliothek/internal/repository"
	"bibliothek/internal/service"

	"github.com/jackc/pgx/v5"
)

// ErsatzwertVorschlag liefert den Betragsvorschlag für ein Exemplar.
func (t *Tueren) ErsatzwertVorschlag(bescheidAbfragen *repository.BescheidAbfragen) http.HandlerFunc {
	return apierrors.MitFehlerantwort(func(w http.ResponseWriter, r *http.Request) error {
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
