package katalog

import (
	"errors"
	"net/http"

	"bibliothek/api/tuer"
	"bibliothek/apierrors"
	"bibliothek/repository"
)

// GetSystematicsHandler returns all entries from systematik_kategorien
func (s *Tueren) GetSystematicsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		kategorien, err := repository.LadeSystematikKategorien(ctx, s.DB.Pool)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, errors.New("database error"))
			return
		}

		type Systematik struct {
			ID          string `json:"id"`
			Kuerzel     string `json:"kuerzel"`
			Bezeichnung string `json:"bezeichnung"`
		}
		var results []Systematik
		for _, k := range kategorien {
			results = append(results, Systematik{ID: k.ID, Kuerzel: k.Kuerzel, Bezeichnung: k.Bezeichnung})
		}

		tuer.RespondJSON(w, http.StatusOK, results)
	}
}

// GetFaecherHandler liefert die distinkten Fächer (buecher_titel.subject) — für die
// Fach-Auswahl beim gezielten Inventur-Scope ("nur Mathe, Klasse 5").
func (s *Tueren) GetFaecherHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		faecher, err := repository.LadeFaecher(ctx, s.DB.Pool)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, errors.New("database error"))
			return
		}
		tuer.RespondJSON(w, http.StatusOK, faecher)
	}
}
