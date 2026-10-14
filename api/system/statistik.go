package system

// Die Tür der Statistik. Die Abfragen und ihre Filter stehen in repository/statistik.go.

import (
	"bibliothek/api/tuer"
	"bibliothek/repository"
	"log"
	"net/http"
	"strconv"
)

// leseListenGrenze begrenzt den ?limit=-Parameter für die Renner-/Ladenhüter-
// Listen. Default 5 (Dashboard-Kacheln); das Drill-Down-Panel lädt einmalig
// mehr und filtert rein clientseitig. Hartes Cap gegen Missbrauch.
func leseListenGrenze(wert string) int {
	grenze, err := strconv.Atoi(wert)
	if err != nil || grenze < 1 {
		return 5
	}
	if grenze > 200 {
		return 200
	}
	return grenze
}

// Statistik returns analytical metadata details.
// Optional query parameters:
//   - ?zeitraum=all|schuljahr|monat filtert das Renner-Ranking zeitlich.
//   - ?type=lmf|freihand filtert ALLE Kennzahlen und Listen auf den
//     LMF-Bestand (Lernmittel, Titel-Präfix „lmf-") bzw. die Schülerbücherei.
//     Ohne Parameter: Gesamtbestand.
func (t *Tueren) Statistik() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		ausleihenFilter := repository.AusleihZeitraumBedingung(r.URL.Query().Get("zeitraum"))
		bestandsFilter, bestandsName := repository.BestandsFilterBedingung(r.URL.Query().Get("type"))
		listenGrenze := leseListenGrenze(r.URL.Query().Get("limit"))

		// 1. Beliebteste Titel (Die Renner) — inkl. Drill-Down-Feldern
		renner := repository.LadeRenner(ctx, t.DB.Pool, ausleihenFilter, bestandsFilter, listenGrenze)

		// 2. Ladenhüter (No checkouts since 2 years or never) — inkl. Drill-Down-Feldern
		ladenhueter := repository.LadeLadenhueter(ctx, t.DB.Pool, bestandsFilter, listenGrenze)

		// 3. Verlust-, Finanz- und Zirkulationskennzahlen (EIN aggregierter Scan)
		kennzahlen, err := repository.LadeBestandKennzahlen(ctx, t.DB.Pool, bestandsFilter)
		if err != nil {
			log.Printf("stats: Bestandskennzahlen konnten nicht ermittelt werden: %v", err)
			kennzahlen = &repository.BestandKennzahlen{}
		}

		// 4. Aktivitäts-Zeitreihe (Ausleihen/Rückgaben je Monat, letzte 12 Monate).
		//    Bewusst NICHT vom ?zeitraum-Parameter abhängig: der Trend definiert sein
		//    eigenes 12-Monats-Fenster; der Bestand-Filter (LMF/Freihand) gilt aber.
		monatsTrend := repository.LadeMonatsTrend(ctx, t.DB.Pool, bestandsFilter)

		tuer.AntworteJSON(w, http.StatusOK, map[string]any{
			"filter_type":    bestandsName,
			"popular_titles": renner,
			"shelf_warmers":  ladenhueter,
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
