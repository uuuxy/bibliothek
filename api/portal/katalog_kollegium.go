package portal

import (
	"net/http"
	"strconv"
	"strings"

	"bibliothek/api/tuer"
	"bibliothek/apierrors"
	"bibliothek/repository"
)

// Der Katalog des Kollegiums: die Suche in „Mein Portal". Dieselbe Abfrage wie der
// öffentliche Katalog (repository.FindeImKatalog), aber hinter der Anmeldung und mit
// repository.SQLKollegiumSichtbar: auch Lernmittel und Titel, deren Exemplare bestellt und
// noch nicht eingetroffen sind. Im Portal werden die Klassensätze reserviert, und das sind
// zum größten Teil Schulbücher; ohne Anmeldung bleiben sie verborgen, weil die
// Bestandszahlen der Schule nicht offen ins Netz gehören.

// KollegiumTitel ist ein Treffer im Katalog des Kollegiums: die Angaben des öffentlichen
// Katalogs und dazu die Zahl der bestellten Exemplare.
type KollegiumTitel struct {
	OpacTitel
	ImZulauf int `json:"im_zulauf"` // bestellt, noch nicht eingetroffen
}

// KollegiumKatalogSucheHandler handles GET /api/reservierungen/klassensatz/katalog?q=…
//
// schlagwort_id beschränkt auf die Titel eines Worts, den Filter unter der Suche. Ohne
// Suchtext liefert er alle Titel des Worts, mit Suchtext die, die beides treffen.
func (s *Tueren) KollegiumKatalogSucheHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := strings.TrimSpace(r.URL.Query().Get("q"))
		schlagwortID, err := tuer.UUIDAusQuery(r, "schlagwort_id")
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusBadRequest, err)
			return
		}
		if q == "" && schlagwortID == "" {
			schreibeLeereTrefferliste(w)
			return
		}

		treffer, gesamt, err := repository.FindeImKatalog(r.Context(), s.DB.Pool, repository.KatalogSuche{
			Sichtbar:     repository.SQLKollegiumSichtbar("bt"),
			Suchtext:     q,
			SchlagwortID: schlagwortID,
			Grenze:       opacGrenze,
		})
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		result := make([]KollegiumTitel, 0, len(treffer))
		for _, t := range treffer {
			result = append(result, KollegiumTitel{OpacTitel: opacTitelAus(t), ImZulauf: t.ImZulauf})
		}
		w.Header().Set(opacTrefferKopf, strconv.Itoa(gesamt))
		tuer.RespondJSON(w, http.StatusOK, result)
	}
}

// KollegiumKatalogFilterHandler handles GET /api/reservierungen/klassensatz/katalog/filter —
// die Schlagworte, die unter der Suche als Filter stehen (Pflegeseite, ist_filter), mit
// ihrer Kennung für ?schlagwort_id=. Nur Wörter, zu denen dieser Katalog einen Titel zeigt.
func (s *Tueren) KollegiumKatalogFilterHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		filter, err := repository.LadeSchlagwortFilterImKatalog(r.Context(), s.DB.Pool, repository.SQLKollegiumSichtbar("bt"))
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		tuer.RespondJSON(w, http.StatusOK, filter)
	}
}
