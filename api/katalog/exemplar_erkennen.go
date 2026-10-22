package katalog

import (
	"net/http"
	"strings"

	"bibliothek/api/tuer"
	"bibliothek/apierrors"
	"bibliothek/internal/service"
	"bibliothek/repository"
)

// exemplarErkennenAntwort ist die Antwort von GET /api/buecher/exemplare/erkennen: das Exemplar
// zur Nummer, oder null, wenn keines sie trägt.
type exemplarErkennenAntwort struct {
	Treffer *service.ExemplarTreffer `json:"treffer"`
}

// ExemplarErkennen löst eine gescannte oder getippte Nummer zu einem Exemplar auf, ohne Wirkung:
// über den Barcode selbst und über das Etikett von Littera (service.ErkenneExemplar, dieselbe
// Erkennung wie an der Theke). Die Suche im Medienkatalog zeigt damit den Titel des Exemplars;
// eine Exemplarnummer steht in keiner Angabe eines Titels.
//
// Eine Nummer, die kein Exemplar trägt, ist kein Fehler, sondern ein leerer Treffer: Die Suche
// fragt bei jeder Eingabe, die wie eine Nummer aussieht.
func (t *Tueren) ExemplarErkennen(katalogAbfragen *repository.KatalogAbfragen) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		code := strings.TrimSpace(r.URL.Query().Get("code"))
		if code == "" {
			apierrors.AntworteFehlerMitMeldung(w, http.StatusBadRequest, "Bitte eine Nummer angeben", nil)
			return
		}
		treffer, err := service.ErkenneExemplar(r.Context(), katalogAbfragen, code)
		if err != nil {
			apierrors.AntworteFehler(w, http.StatusInternalServerError, err)
			return
		}
		tuer.AntworteJSON(w, http.StatusOK, exemplarErkennenAntwort{Treffer: treffer})
	}
}
