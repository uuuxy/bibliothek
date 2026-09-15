package api

import (
	"net/http"

	"bibliothek/apierrors"
	"bibliothek/repository"
)

// BescheidAusstehendHandler liefert die Kinder mit offenen Forderungen, die noch auf
// keinem Bescheid stehen — die Zeilen „Bescheid noch nicht erstellt" im Reiter
// „Schadensersatz" des Mahnwesens.
//
// @Summary      Students with open claims that are not on a notice yet
// @Tags         schadensersatz
// @Produce      json
// @Success      200 {array} repository.ForderungOhneBescheid
// @Router       /bescheide/ausstehend [get]
func (s *Server) BescheidAusstehendHandler(bescheidRepo repository.BescheidRepository) http.HandlerFunc {
	return apierrors.Wrap(func(w http.ResponseWriter, r *http.Request) error {
		liste, err := bescheidRepo.Ausstehend(r.Context())
		if err != nil {
			return apierrors.Internal("Offene Forderungen konnten nicht gelesen werden", err)
		}
		RespondJSON(w, http.StatusOK, liste)
		return nil
	})
}
