package api

// Die Tür der Statistik. Die Abfragen und ihre Filter stehen in repository/statistik.go.

import (
	"bibliothek/repository"
	"log"
	"net/http"
	"strconv"
)

// resolveListLimit begrenzt den ?limit=-Parameter für die Renner-/Ladenhüter-
// Listen. Default 5 (Dashboard-Kacheln); das Drill-Down-Panel lädt einmalig
// mehr und filtert rein clientseitig. Hartes Cap gegen Missbrauch.
func resolveListLimit(raw string) int {
	limit, err := strconv.Atoi(raw)
	if err != nil || limit < 1 {
		return 5
	}
	if limit > 200 {
		return 200
	}
	return limit
}

// GetStatisticsHandler returns analytical metadata details.
// Optional query parameters:
//   - ?zeitraum=all|schuljahr|monat filtert das Renner-Ranking zeitlich.
//   - ?type=lmf|freihand filtert ALLE Kennzahlen und Listen auf den
//     LMF-Bestand (Lernmittel, Titel-Präfix „lmf-") bzw. die Schülerbücherei.
//     Ohne Parameter: Gesamtbestand.
func (s *Server) GetStatisticsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		ausleihenFilter := repository.AusleihZeitraumBedingung(r.URL.Query().Get("zeitraum"))
		typeFilter, typeName := repository.BestandsFilterBedingung(r.URL.Query().Get("type"))
		listLimit := resolveListLimit(r.URL.Query().Get("limit"))

		// 1. Beliebteste Titel (Die Renner) — inkl. Drill-Down-Feldern
		popularTitles := repository.ListeRenner(ctx, s.DB.Pool, ausleihenFilter, typeFilter, listLimit)

		// 2. Ladenhüter (No checkouts since 2 years or never) — inkl. Drill-Down-Feldern
		shelfWarmers := repository.ListeLadenhueter(ctx, s.DB.Pool, typeFilter, listLimit)

		// 3. Verlust-, Finanz- und Zirkulationskennzahlen (EIN aggregierter Scan)
		kennzahlen, err := repository.LadeBestandKennzahlen(ctx, s.DB.Pool, typeFilter)
		if err != nil {
			log.Printf("stats: Bestandskennzahlen konnten nicht ermittelt werden: %v", err)
			kennzahlen = &repository.BestandKennzahlen{}
		}

		// 4. Aktivitäts-Zeitreihe (Ausleihen/Rückgaben je Monat, letzte 12 Monate).
		//    Bewusst NICHT vom ?zeitraum-Parameter abhängig: der Trend definiert sein
		//    eigenes 12-Monats-Fenster; der Bestand-Filter (LMF/Freihand) gilt aber.
		monatsTrend := repository.LadeMonatsTrend(ctx, s.DB.Pool, typeFilter)

		RespondJSON(w, http.StatusOK, map[string]any{
			"filter_type":    typeName,
			"popular_titles": popularTitles,
			"shelf_warmers":  shelfWarmers,
			"monats_trend":   monatsTrend,
			"loss_stats": map[string]any{
				"gesamt_bestand":      kennzahlen.GesamtBestand,
				"verlorene_exemplare": kennzahlen.VerloreneExemplare,
				"verlust_quote":       kennzahlen.VerlustQuote,
			},
			"wiederbeschaffungswert_defekt": kennzahlen.WiederbeschaffungswertDefekt,
			"zirkulationsquote":             kennzahlen.Zirkulationsquote,
			"zirkulation": map[string]any{
				"aktuell_verliehen": kennzahlen.AktuellVerliehen,
				"aktiver_bestand":   kennzahlen.AktiverBestand,
			},
		})
	}
}
