package druck

import (
	"errors"
	"net/http"
	"strings"

	"bibliothek/api/tuer"
	"bibliothek/apierrors"
	"bibliothek/repository"
)

// titelSucheAntwort ist die Antwort von GET /api/buecher/titel/suche: Titel, sonst nichts.
type titelSucheAntwort struct {
	Titel      []repository.Titel `json:"books"`
	Gesamtzahl int                `json:"books_total"`
}

// TitelSuche ist die Titel-Hälfte von SearchHandler als eigene Tür (view_books).
//
// Die Etiketten-Titelsuche im Druck-Center rief bis 05.09.2026 GET /api/search und las aus
// der Antwort nur `books` — die Schüler-Kiosk-Daten darin zeigte sie nie, holte sie aber
// mit; und weil /api/search am Theken-Recht perform_actions hängt, meldete der
// Druckbildschirm ohne dieses Recht nur noch „Suche nicht möglich". Die Inventur-Liste
// GET /api/books wäre keine Alternative: Sie sucht per ILIKE über Titel/Autor/ISBN, ohne
// Signatur und ohne Umlaut-Normalisierung — KatalogAbfragen.FindeTitelUnscharf ist die Suchgüte der Theke,
// und die bleibt hier erhalten.
func (t *Tueren) TitelSuche(katalogAbfragen repository.KatalogAbfragen) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		suchtext := strings.TrimSpace(r.URL.Query().Get("q"))
		if suchtext == "" {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("search query 'q' cannot be empty"))
			return
		}
		titel, gesamt, err := katalogAbfragen.FindeTitelUnscharf(r.Context(), suchtext, 10)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		if titel == nil {
			titel = []repository.Titel{}
		}
		tuer.AntworteJSON(w, http.StatusOK, titelSucheAntwort{Titel: titel, Gesamtzahl: gesamt})
	}
}
