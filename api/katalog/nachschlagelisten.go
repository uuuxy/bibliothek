package katalog

import (
	"errors"
	"net/http"

	"bibliothek/api/tuer"
	"bibliothek/apierrors"
	"bibliothek/repository"
)

// Systematik returns all entries from systematik_kategorien
func (t *Tueren) Systematik() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		kategorien, err := repository.LadeSystematikKategorien(ctx, t.DB.Pool)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, errors.New("database error"))
			return
		}

		type sachgruppe struct {
			ID          string `json:"id"`
			Kuerzel     string `json:"kuerzel"`
			Bezeichnung string `json:"bezeichnung"`
		}
		var sachgruppen []sachgruppe
		for _, k := range kategorien {
			sachgruppen = append(sachgruppen, sachgruppe{ID: k.ID, Kuerzel: k.Kuerzel, Bezeichnung: k.Bezeichnung})
		}

		tuer.AntworteJSON(w, http.StatusOK, sachgruppen)
	}
}

// Faecher liefert die distinkten Fächer (buecher_titel.subject) — für die
// Fach-Auswahl beim gezielten Inventur-Scope ("nur Mathe, Klasse 5").
func (t *Tueren) Faecher() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		faecher, err := repository.LadeFaecher(ctx, t.DB.Pool)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, errors.New("database error"))
			return
		}
		tuer.AntworteJSON(w, http.StatusOK, faecher)
	}
}
