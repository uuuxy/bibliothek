package api

import (
	"net/http"
	"strconv"

	"bibliothek/apierrors"
	"bibliothek/repository"
)

// bestellhistorieStandardLimit / -MaxLimit deckeln die Liste.
//
// Ohne Grenze lieferte dieser Endpunkt ALLE Bestellungen samt Positionen: auf einer
// gewachsenen Datenbank (5.257 Bestellungen) waren das 2,45 MB und 3,9 Sekunden, und es
// wird jedes Schuljahr mehr. Dieselbe Bugklasse hatte das Audit-Log (247k Zeilen, 72 MB).
//
// Die Summen im Kopf der Oberfläche dürfen davon NICHT abhängen — sie kommen aus
// /api/bestellhistorie/uebersicht und zählen weiterhin alles.
const (
	bestellhistorieStandardLimit = 200
	bestellhistorieMaxLimit      = 500
)

// GetBestellhistorieHandler returns the most recent orders with their line items.
func (s *Server) GetBestellhistorieHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		limit := bestellhistorieStandardLimit
		if roh := r.URL.Query().Get("limit"); roh != "" {
			if n, err := strconv.Atoi(roh); err == nil && n > 0 {
				limit = min(n, bestellhistorieMaxLimit)
			}
		}

		// Der Topf-Filter: Die Liste ist gedeckelt, gefiltert wird deshalb im SQL und
		// nicht im Browser — sonst zeigte „Lernmittelfreiheit" nur, was von den neuesten
		// 200 Bestellungen übrig bleibt.
		mittel := r.URL.Query().Get("mittel")
		if !repository.MittelFilterGueltig(mittel) {
			apierrors.SendHTTPError(w, http.StatusBadRequest, repository.MittelFilterFehler(mittel))
			return
		}

		orders, orderIndex, err := repository.LadeBestellVerlauf(ctx, s.DB.Pool, limit, mittel)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		if len(orders) == 0 {
			RespondJSON(w, http.StatusOK, orders)
			return
		}

		if err := repository.LadeBestellVerlaufPositionen(ctx, s.DB.Pool, orders, orderIndex); err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		RespondJSON(w, http.StatusOK, orders)
	}
}
