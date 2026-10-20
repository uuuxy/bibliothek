package system

// Die Tür der Statistik. Die Abfragen und ihre Filter stehen in repository/statistik.go.

import (
	"bibliothek/api/tuer"
	"bibliothek/repository"
	"log"
	"net/http"
	"strconv"
)

// leseListenGrenze begrenzt den Parameter ?limit= für die Listen der Renner und Ladenhüter.
// Vorgabe 5 (Kacheln der Übersicht); die Detailseite lädt einmalig mehr (limit=100) und
// filtert dann in der Oberfläche. Höchstens 200, gegen Missbrauch.
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

// Statistik liefert die Kennzahlen und Listen der Statistik. Parameter, beide wahlweise:
//   - ?zeitraum=all|schuljahr|monat grenzt das Ranking der Renner zeitlich ein.
//   - ?type=lmf|freihand grenzt alle Kennzahlen und Listen auf den Bestand der Lernmittel
//     (Titel mit dem Anfang „lmf-") oder die Schülerbücherei ein.
//     Ohne Parameter: der ganze Bestand.
//
// nicht_gelesen nennt die Abschnitte, deren Abfrage scheiterte: renner, ladenhueter,
// kennzahlen, monats_trend. Sie stehen leer in der Antwort und sind kein leerer Bestand.
func (t *Tueren) Statistik() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		ausleihenFilter := repository.AusleihZeitraumBedingung(r.URL.Query().Get("zeitraum"))
		bestandsFilter, bestandsName := repository.BestandsFilterBedingung(r.URL.Query().Get("type"))
		listenGrenze := leseListenGrenze(r.URL.Query().Get("limit"))

		// Jeder Abschnitt steht für sich: Scheitert einer, bleibt er leer, steht im Log und in
		// nicht_gelesen, die übrigen erscheinen trotzdem. Ohne die Liste sähe ein Lesefehler aus
		// wie ein leerer Bestand; eine Teilliste zeigt keiner, weil sie wie ein Ergebnis aussähe.
		nichtGelesen := []string{}

		// 1. Beliebteste Titel (die Renner), mit den Feldern der Detailseite
		renner, err := repository.LadeRenner(ctx, t.DB.Pool, ausleihenFilter, bestandsFilter, listenGrenze)
		if err != nil {
			log.Printf("stats: Renner konnten nicht ermittelt werden: %v", err)
			renner = []repository.Renner{}
			nichtGelesen = append(nichtGelesen, "renner")
		}

		// 2. Ladenhüter (seit zwei Jahren oder nie ausgeliehen), mit den Feldern der Detailseite
		ladenhueter, err := repository.LadeLadenhueter(ctx, t.DB.Pool, bestandsFilter, listenGrenze)
		if err != nil {
			log.Printf("stats: Ladenhüter konnten nicht ermittelt werden: %v", err)
			ladenhueter = []repository.Ladenhueter{}
			nichtGelesen = append(nichtGelesen, "ladenhueter")
		}

		// 3. Verlust-, Finanz- und Zirkulationskennzahlen (ein aggregierter Scan)
		kennzahlen, err := repository.LadeBestandKennzahlen(ctx, t.DB.Pool, bestandsFilter)
		if err != nil {
			log.Printf("stats: Bestandskennzahlen konnten nicht ermittelt werden: %v", err)
			kennzahlen = &repository.BestandKennzahlen{}
			nichtGelesen = append(nichtGelesen, "kennzahlen")
		}

		// 4. Aktivitäts-Zeitreihe (Ausleihen/Rückgaben je Monat, letzte 12 Monate).
		//    Bewusst nicht vom ?zeitraum-Parameter abhängig: der Trend definiert sein
		//    eigenes 12-Monats-Fenster; der Bestand-Filter (LMF/Freihand) gilt aber.
		monatsTrend, err := repository.LadeMonatsTrend(ctx, t.DB.Pool, bestandsFilter)
		if err != nil {
			log.Printf("stats: Monatstrend konnte nicht ermittelt werden: %v", err)
			monatsTrend = []repository.MonatsTrendPunkt{}
			nichtGelesen = append(nichtGelesen, "monats_trend")
		}

		tuer.AntworteJSON(w, http.StatusOK, map[string]any{
			"filter_type":    bestandsName,
			"nicht_gelesen":  nichtGelesen,
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
