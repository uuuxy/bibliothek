package katalog

import (
	"context"
	"net/http"

	"bibliothek/api/tuer"
	"bibliothek/internal/service"
)

// CoverAbgleichen startet im Hintergrund den Abgleich fehlender Cover
// (CoverDienst.GleicheFehlendeCoverAb) und antwortet sofort. abgleich ist derselbe Dienst wie
// beim Start und im Zeitplan; läuft er schon, entfällt der zweite Lauf.
func (t *Tueren) CoverAbgleichen(abgleich *service.CoverDienst) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Der Abgleich läuft weiter, wenn die Antwort hinaus ist; er erbt die Werte der Anfrage,
		// nicht ihr Ende.
		go abgleich.GleicheFehlendeCoverAb(context.WithoutCancel(r.Context()))

		tuer.AntworteJSON(w, http.StatusOK, map[string]string{
			"status":  "success",
			"message": "Fehlende Cover werden im Hintergrund geladen",
		})
	}
}
