package portal

import (
	"net/http"
	"strconv"
	"strings"

	"bibliothek/api/tuer"
	"bibliothek/apierrors"
	"bibliothek/pkg/httpresp"
	"bibliothek/repository"
)

// opacTrefferKopf nennt, wie viele Titel eine Katalogsuche trifft. Die Antwort zeigt
// höchstens opacGrenze davon; ohne die Zahl sähen 50 gezeigte Treffer aus wie alle. Ein
// Kopf statt eines Felds, weil die Antwort eine Liste ist, die OPAC-Seite und „Mein Portal"
// so lesen — dasselbe Muster wie X-Sperre an der Theke (api/theke/eingabe.go).
const (
	opacTrefferKopf = "X-Treffer-Gesamt"
	opacGrenze      = 50
)

// OPACTitel ist ein Treffer im öffentlichen Katalog: nur Angaben zum Titel, keine zu Ausleihen
// oder Lesern (DSGVO).
type OPACTitel struct {
	ID         string `json:"id"`
	Titel      string `json:"titel"`
	Autor      string `json:"autor"`
	ISBN       string `json:"isbn,omitempty"`
	CoverURL   string `json:"cover_url,omitempty"`
	Verfuegbar int    `json:"verfuegbar"` // Exemplare, die gerade ausleihbar sind
	Gesamt     int    `json:"gesamt"`     // alle Exemplare
}

// opacTitelAus bildet einen Katalogtreffer auf die Antwort des öffentlichen Katalogs ab.
func opacTitelAus(t repository.KatalogTreffer) OPACTitel {
	return OPACTitel{
		ID: t.ID, Titel: t.Titel, Autor: t.Autor, ISBN: t.ISBN, CoverURL: t.CoverURL,
		Verfuegbar: t.Verfuegbar, Gesamt: t.Gesamt,
	}
}

// schreibeLeereTrefferliste antwortet auf eine Suche ohne Suchtext und ohne Filter.
func schreibeLeereTrefferliste(w http.ResponseWriter) {
	w.Header().Set(tuer.HeaderContentType, tuer.ContentTypeJSON)
	httpresp.Schreibe(w, []byte("[]"))
}

// OPACSuche sucht im öffentlichen Katalog: GET /api/public/opac/suche?q=…, ohne Anmeldung und
// deshalb ohne Angaben zu Ausleihen oder Lesern (DSGVO).
//
// Was ohne Anmeldung sichtbar ist (kein Lernmittel, mindestens ein Exemplar im Haus), steht
// genau einmal in repository.OeffentlichSichtbar — dieselbe Regel wie im Flur-Monitor. Das
// Kollegium sucht über eine eigene Tür hinter der Anmeldung (katalog_kollegium.go), die
// auch Lernmittel und bestellte Titel zeigt; die Abfrage ist dieselbe.
func (t *Tueren) OPACSuche() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := strings.TrimSpace(r.URL.Query().Get("q"))
		if q == "" {
			schreibeLeereTrefferliste(w)
			return
		}

		treffer, gesamt, err := repository.FindeImKatalog(r.Context(), t.DB.Pool, repository.KatalogSuche{
			Sichtbar: repository.OeffentlichSichtbar("bt"),
			Suchtext: q,
			Grenze:   opacGrenze,
		})
		if err != nil {
			apierrors.AntworteFehler(w, http.StatusInternalServerError, err)
			return
		}

		liste := make([]OPACTitel, 0, len(treffer))
		for _, eintrag := range treffer {
			liste = append(liste, opacTitelAus(eintrag))
		}
		w.Header().Set(opacTrefferKopf, strconv.Itoa(gesamt))
		tuer.AntworteJSON(w, http.StatusOK, liste)
	}
}
