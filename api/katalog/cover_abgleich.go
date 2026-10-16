package katalog

import (
	"net/http"

	"bibliothek/api/tuer"
	"bibliothek/internal/service"
)

// CoverAbgleichen startet im Hintergrund den Abgleich fehlender Cover
// (CoverDienst.GleicheFehlendeCoverAb) und antwortet sofort. abgleich ist derselbe Dienst wie
// beim Start und im Zeitplan; läuft er schon, entfällt der zweite Lauf.
func (t *Tueren) CoverAbgleichen(abgleich *service.CoverDienst) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		go abgleich.GleicheFehlendeCoverAb()

		tuer.AntworteJSON(w, http.StatusOK, map[string]string{
			"status":  "success",
			"message": "Fehlende Cover werden im Hintergrund geladen",
		})
	}
}
