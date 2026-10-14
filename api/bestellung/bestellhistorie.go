package bestellung

import (
	"net/http"
	"strconv"

	"bibliothek/api/tuer"
	"bibliothek/apierrors"
	"bibliothek/repository"
)

// bestellhistorieVorgabeGrenze und bestellhistorieHoechstGrenze deckeln die Liste.
//
// Ohne Grenze lieferte dieser Endpunkt alle Bestellungen samt Positionen: auf einer
// gewachsenen Datenbank (5.257 Bestellungen) waren das 2,45 MB und 3,9 Sekunden, und es
// wird jedes Schuljahr mehr. Dieselbe Bugklasse hatte das Protokoll (247k Zeilen, 72 MB).
//
// Die Summen im Kopf der Oberfläche dürfen davon nicht abhängen — sie kommen aus
// /api/bestellhistorie/uebersicht und zählen weiterhin alles.
const (
	bestellhistorieVorgabeGrenze = 200
	bestellhistorieHoechstGrenze = 500
)

// Bestellhistorie liefert die jüngsten Bestellungen mit ihren Positionen.
func (t *Tueren) Bestellhistorie() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		grenze := bestellhistorieVorgabeGrenze
		if roh := r.URL.Query().Get("limit"); roh != "" {
			if n, err := strconv.Atoi(roh); err == nil && n > 0 {
				grenze = min(n, bestellhistorieHoechstGrenze)
			}
		}

		// Der Topf-Filter: Die Liste ist gedeckelt, gefiltert wird deshalb im SQL und
		// nicht im Browser — sonst zeigte „Lernmittelfreiheit" nur, was von den neuesten
		// 200 Bestellungen übrig bleibt.
		mittel := r.URL.Query().Get("mittel")
		if !repository.IstMittelFilterGueltig(mittel) {
			apierrors.SendHTTPError(w, http.StatusBadRequest, repository.MittelFilterFehler(mittel))
			return
		}

		bestellungen, bestellIndex, err := repository.LadeBestellVerlauf(ctx, t.DB.Pool, grenze, mittel)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		if len(bestellungen) == 0 {
			tuer.AntworteJSON(w, http.StatusOK, bestellungen)
			return
		}

		if err := repository.LadeBestellVerlaufPositionen(ctx, t.DB.Pool, bestellungen, bestellIndex); err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		tuer.AntworteJSON(w, http.StatusOK, bestellungen)
	}
}
