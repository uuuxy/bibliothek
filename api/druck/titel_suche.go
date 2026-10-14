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

// TitelSuche sucht Titel für die Etiketten im Druck-Center und braucht dafür nur das Recht
// view_books.
//
// Die Suche der Theke (GET /api/search) hängt am Recht perform_actions und liefert neben den
// Titeln Schüler, die der Druck nicht zeigt; die Liste GET /api/books sucht ohne Signatur und
// ohne Umlaut-Normalform. Diese Tür sucht deshalb mit der Abfrage der Theke,
// KatalogAbfragen.FindeTitelUnscharf.
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
