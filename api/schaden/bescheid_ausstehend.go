package schaden

import (
	"net/http"

	"bibliothek/api/tuer"
	"bibliothek/apierrors"
	"bibliothek/repository"
)

// BescheideAusstehend liefert die Kinder mit offenen Forderungen, die noch auf
// keinem Bescheid stehen — die Zeilen „Bescheid noch nicht erstellt" im Reiter
// „Schadensersatz" des Mahnwesens.
func (t *Tueren) BescheideAusstehend(bescheidAbfragen *repository.BescheidAbfragen) http.HandlerFunc {
	return apierrors.MitFehlerantwort(func(w http.ResponseWriter, r *http.Request) error {
		liste, err := bescheidAbfragen.LadeForderungenOhneBescheid(r.Context())
		if err != nil {
			return apierrors.Internal("Offene Forderungen konnten nicht gelesen werden", err)
		}
		tuer.AntworteJSON(w, http.StatusOK, liste)
		return nil
	})
}
