package api

import (
	"net/http"
	"strconv"
	"strings"

	"bibliothek/apierrors"
	"bibliothek/pkg/httpresp"
	"bibliothek/repository"
)

// opacTrefferKopf nennt, wie viele Titel eine Katalogsuche trifft. Die Antwort zeigt
// höchstens opacGrenze davon; ohne die Zahl sähen 50 gezeigte Treffer aus wie alle. Ein
// Kopf statt eines Felds, weil die Antwort eine Liste ist, die OPAC-Seite und „Mein Portal"
// so lesen — dasselbe Muster wie X-Sperre an der Theke (api/action.go).
const (
	opacTrefferKopf = "X-Treffer-Gesamt"
	opacGrenze      = 50
)

// OpacTitel is a DSGVO-compliant book view for the public catalog.
// Contains no loan data and no reader data.
type OpacTitel struct {
	ID         string `json:"id"`
	Titel      string `json:"titel"`
	Autor      string `json:"autor"`
	ISBN       string `json:"isbn,omitempty"`
	CoverURL   string `json:"cover_url,omitempty"`
	Verfuegbar int    `json:"verfuegbar"` // copies currently available
	Gesamt     int    `json:"gesamt"`     // total copies
}

// opacTitelAus bildet einen Katalogtreffer auf die Antwort des öffentlichen Katalogs ab.
func opacTitelAus(t repository.KatalogTreffer) OpacTitel {
	return OpacTitel{
		ID: t.ID, Titel: t.Titel, Autor: t.Autor, ISBN: t.ISBN, CoverURL: t.CoverURL,
		Verfuegbar: t.Verfuegbar, Gesamt: t.Gesamt,
	}
}

// schreibeLeereTrefferliste antwortet auf eine Suche ohne Suchtext und ohne Filter.
func schreibeLeereTrefferliste(w http.ResponseWriter) {
	w.Header().Set(headerContentType, contentTypeJSON)
	httpresp.Write(w, []byte("[]"))
}

// PublicCatalogSearchHandler handles GET /api/public/opac/suche?q=...
// Public endpoint: no auth required. Never exposes loan or reader data (DSGVO).
//
// Was ohne Anmeldung sichtbar ist (kein Lernmittel, mindestens ein Exemplar im Haus), steht
// genau einmal in repository.OeffentlichSichtbar — dieselbe Regel wie im Flur-Monitor. Das
// Kollegium sucht über eine eigene Tür hinter der Anmeldung (katalog_kollegium.go), die
// auch Lernmittel und bestellte Titel zeigt; die Abfrage ist dieselbe.
func (s *Server) PublicCatalogSearchHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := strings.TrimSpace(r.URL.Query().Get("q"))
		if q == "" {
			schreibeLeereTrefferliste(w)
			return
		}

		treffer, gesamt, err := repository.SucheImKatalog(r.Context(), s.DB.Pool, repository.KatalogSuche{
			Sichtbar: repository.OeffentlichSichtbar("bt"),
			Suchtext: q,
			Grenze:   opacGrenze,
		})
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		result := make([]OpacTitel, 0, len(treffer))
		for _, t := range treffer {
			result = append(result, opacTitelAus(t))
		}
		w.Header().Set(opacTrefferKopf, strconv.Itoa(gesamt))
		RespondJSON(w, http.StatusOK, result)
	}
}
