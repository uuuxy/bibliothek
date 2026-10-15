package katalog

import (
	"errors"
	"net/http"

	"bibliothek/api/tuer"
	"bibliothek/apierrors"
	"bibliothek/auth"
	"bibliothek/repository"
)

// TitelLoeschen löscht einen Titel mit seinen Exemplaren und schreibt den Vorgang ins
// Protokoll (ProtokollAbfragen.LoescheTitel). Ist ein Exemplar verliehen, lehnt die Tür mit
// 400 ab.
//
// @Summary      Titel löschen
// @Description  Löscht einen Titel mit allen Exemplaren und schreibt den Vorgang ins Protokoll. Ist ein Exemplar verliehen, ist das 400.
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "Kennung des Titels (UUID)"
// @Success      200  {object}  map[string]string
// @Failure      400  {object}  map[string]string
// @Failure      401  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /buecher/titel/{id} [delete]
func (t *Tueren) TitelLoeschen(protokollAbfragen repository.ProtokollAbfragen) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := auth.ClaimsAusKontext(r.Context())
		if !ok {
			apierrors.AntworteFehler(w, http.StatusUnauthorized, errors.New("missing session information"))
			return
		}
		id := r.PathValue("id")
		if id == "" {
			apierrors.AntworteFehler(w, http.StatusBadRequest, errors.New("missing title ID parameter"))
			return
		}

		ctx := r.Context()

		err := protokollAbfragen.LoescheTitel(ctx, id, claims.KontoID)
		if err != nil {
			if errors.Is(err, repository.ErrTitelHatAktiveAusleihen) {
				apierrors.AntworteFehler(w, http.StatusBadRequest, err)
			} else if errors.Is(err, repository.ErrTitelNichtGefunden) {
				apierrors.AntworteFehler(w, http.StatusNotFound, err)
			} else {
				apierrors.AntworteFehler(w, http.StatusInternalServerError, err)
			}
			return
		}

		tuer.AntworteErfolg(w)
	}
}
