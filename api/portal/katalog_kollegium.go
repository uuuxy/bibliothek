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
	OPACTitel
	ImZulauf int `json:"im_zulauf"` // bestellt, noch nicht eingetroffen
}

// KollegiumKatalogSuche sucht für „Mein Portal": GET /api/reservierungen/klassensatz/katalog?q=…
//
// schlagwort_id beschränkt auf die Titel eines Worts, den Filter unter der Suche. Ohne
// Suchtext liefert er alle Titel des Worts, mit Suchtext die, die beides treffen.
func (t *Tueren) KollegiumKatalogSuche() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := strings.TrimSpace(r.URL.Query().Get("q"))
		schlagwortID, err := tuer.LeseUUIDParameter(r, "schlagwort_id")
		if err != nil {
			apierrors.AntworteFehler(w, http.StatusBadRequest, err)
			return
		}
		if q == "" && schlagwortID == "" {
			schreibeLeereTrefferliste(w)
			return
		}

		treffer, gesamt, err := repository.FindeImKatalog(r.Context(), t.DB.Pool, repository.KatalogSuche{
			Sichtbar:     repository.SQLKollegiumSichtbar("bt"),
			Suchtext:     q,
			SchlagwortID: schlagwortID,
			Grenze:       opacGrenze,
		})
		if err != nil {
			apierrors.AntworteFehler(w, http.StatusInternalServerError, err)
			return
		}

		liste := make([]KollegiumTitel, 0, len(treffer))
		for _, eintrag := range treffer {
			liste = append(liste, KollegiumTitel{OPACTitel: opacTitelAus(eintrag), ImZulauf: eintrag.ImZulauf})
		}
		w.Header().Set(opacTrefferKopf, strconv.Itoa(gesamt))
		tuer.AntworteJSON(w, http.StatusOK, liste)
	}
}

// KollegiumKatalogFilter liefert für GET /api/reservierungen/klassensatz/katalog/filter die
// Schlagworte, die unter der Suche als Filter stehen (Pflegeseite, ist_filter), mit
// ihrer Kennung für ?schlagwort_id=. Nur Wörter, zu denen dieser Katalog einen Titel zeigt.
func (t *Tueren) KollegiumKatalogFilter() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		filter, err := repository.LadeSchlagwortFilterImKatalog(r.Context(), t.DB.Pool, repository.SQLKollegiumSichtbar("bt"))
		if err != nil {
			apierrors.AntworteFehler(w, http.StatusInternalServerError, err)
			return
		}
		tuer.AntworteJSON(w, http.StatusOK, filter)
	}
}
