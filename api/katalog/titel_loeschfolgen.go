package katalog

import (
	"net/http"

	"bibliothek/api/tuer"
	"bibliothek/apierrors"
)

// TitelLoeschfolgen ist die Tür GET /api/buecher/titel/loeschfolgen?ids=a,b: die Vorschau des
// Löschdialogs. Sie nennt die Exemplare im Haus und die verliehenen mit ihrem Leser, unter
// denselben Bedingungen, unter denen das Löschen sie behandelt (TitelAbfragen.LadeLoeschfolgen).
func (t *Tueren) TitelLoeschfolgen() http.HandlerFunc {
	return func(antwort http.ResponseWriter, anfrage *http.Request) {
		ids, fehler := tuer.LeseUUIDListe(anfrage.URL.Query(), "ids")
		if fehler != nil {
			apierrors.AntworteFehler(antwort, http.StatusBadRequest, fehler)
			return
		}
		if meldung := meldungZuTitelKennungen(ids, alleUUIDs(ids)); meldung != "" {
			apierrors.AntworteFehlerMitMeldung(antwort, http.StatusBadRequest, meldung, nil)
			return
		}
		folgen, fehler := t.titelAbfragen().LadeLoeschfolgen(anfrage.Context(), ids)
		if fehler != nil {
			apierrors.AntworteFehler(antwort, http.StatusInternalServerError, fehler)
			return
		}
		tuer.AntworteJSON(antwort, http.StatusOK, folgen)
	}
}
