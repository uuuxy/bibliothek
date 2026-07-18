package api

// stats.go — Handlers for library statistics, reorder reporting and PDF export.
// Inventory scanning and Fehlbestand (missing copies) live in inventory.go.

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"bibliothek/pkg/lmf"
)

// PopularTitle ist ein Eintrag der „Renner"-Liste inkl. Drill-Down-Feldern
// (Fachbereich/Systematik/Erscheinungsjahr) für Frontend-Filter.
type PopularTitle struct {
	ID               string `json:"id"`
	Titel            string `json:"titel"`
	Autor            string `json:"autor"`
	CoverURL         string `json:"cover_url"`
	Fachbereich      string `json:"fachbereich,omitempty"`
	Systematik       string `json:"systematik,omitempty"`
	Erscheinungsjahr int    `json:"erscheinungsjahr,omitempty"`
	Count            int    `json:"count"`
}

// ShelfWarmer ist ein Eintrag der „Ladenhüter"-Liste inkl. Drill-Down-Feldern.
type ShelfWarmer struct {
	// ID ist Pflicht, auch wenn die Liste sie nicht anzeigt: Sie ist der einzige
	// eindeutige Schlüssel. Zwei Titel dürfen legitim gleich heissen und beide keine
	// ISBN haben (etwa zwei Ausgaben desselben Werks) — ohne ID kollidierten sie im
	// Frontend zu einem doppelten each-Key und rissen die Statistik-Ansicht ab.
	ID               string `json:"id"`
	Titel            string `json:"titel"`
	Autor            string `json:"autor"`
	ISBN             string `json:"isbn"`
	LetzteAusleihe   string `json:"letzte_aus"`
	Fachbereich      string `json:"fachbereich,omitempty"`
	Systematik       string `json:"systematik,omitempty"`
	Erscheinungsjahr int    `json:"erscheinungsjahr,omitempty"`
}

// bestandKennzahlen bündelt alle aggregierten Zahlen aus EINEM Scan über
// buecher_exemplare (statt mehrerer Subselects pro Kennzahl).
type bestandKennzahlen struct {
	GesamtBestand      int
	AktiverBestand     int // physisch vorhanden = nicht ausgesondert
	AktuellVerliehen   int
	// VerloreneExemplare: nur echte Abgänge (aussonderung_grund VERLUST oder
	// BESCHAEDIGUNG). Bewusst Aussortiertes und Bestandskorrekturen zählen nicht.
	VerloreneExemplare int
	// Wiederbeschaffungswert: Summe der Einkaufspreise der echten Verluste
	// (VERLUST/BESCHAEDIGUNG) — also der Bücher, die tatsächlich nachgekauft werden
	// müssen. Kuratiert Aussortiertes und Bestandskorrekturen bleiben aussen vor.
	WiederbeschaffungswertDefekt float64
	VerlustQuote                 float64 // echte Verluste / Gesamtbestand
	Zirkulationsquote            float64 // verliehen / aktiver Bestand
}

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

// resolveBestandsFilter mappt den ?type=-Parameter auf ein serverkontrolliertes
// SQL-Fragment. LMF-Bestand ist am LMF-Kennzeichen im Titel erkennbar — zentral und
// schreibvarianten-robust über pkg/lmf, dieselbe Regel wie im Ausleih-Limit.
func resolveBestandsFilter(typ string) (fragment string, normalized string) {
	switch typ {
	case "lmf":
		return "AND " + lmf.SQLBedingung("t.titel"), "lmf"
	case "freihand":
		return "AND NOT (" + lmf.SQLBedingung("t.titel") + ")", "freihand"
	default:
		return "", "alle"
	}
}

// queryBestandKennzahlen liefert Verlust-, Finanz- und Zirkulationszahlen in
// einem einzigen aggregierten Statement.
func (s *Server) queryBestandKennzahlen(ctx context.Context, typeFilter string) (*bestandKennzahlen, error) {
	// Als "Verlust" zählen ausschliesslich unfreiwillige Abgänge: VERLUST (nicht
	// auffindbar) und BESCHAEDIGUNG (Schadensfall). Bewusst NICHT enthalten sind
	// AUSSORTIERT (veraltet/verschlissen, kuratierte Entfernung) und BESTANDSKORREKTUR
	// (Import-/Sync-Anpassung, laut Migration 043 "kein echter Abgang") — sie würden
	// Verlustquote und Wiederbeschaffungswert künstlich aufblähen. aussonderung_grund
	// ist per chk_aussonderung_grund (Migration 043) nur bei ist_ausgesondert gesetzt,
	// deshalb impliziert dieser Filter bereits die Aussonderung.
	const istVerlust = "e.aussonderung_grund IN ('VERLUST', 'BESCHAEDIGUNG')"

	q := fmt.Sprintf(`
		SELECT
			COUNT(*)::int AS gesamt,
			COUNT(*) FILTER (WHERE NOT e.ist_ausgesondert)::int AS aktiv,
			COUNT(*) FILTER (WHERE al.exemplar_id IS NOT NULL AND NOT e.ist_ausgesondert)::int AS verliehen,
			COUNT(*) FILTER (WHERE %[1]s)::int AS verlorene,
			COALESCE(SUM(e.einkaufspreis) FILTER (WHERE %[1]s), 0)::float8 AS wiederbeschaffung,
			CASE WHEN COUNT(*) = 0 THEN 0.0
				 ELSE ROUND(COUNT(*) FILTER (WHERE %[1]s) * 100.0 / COUNT(*), 2)
			END::float8 AS verlust_quote,
			CASE WHEN COUNT(*) FILTER (WHERE NOT e.ist_ausgesondert) = 0 THEN 0.0
				 ELSE ROUND(COUNT(*) FILTER (WHERE al.exemplar_id IS NOT NULL AND NOT e.ist_ausgesondert) * 100.0
					   / COUNT(*) FILTER (WHERE NOT e.ist_ausgesondert), 2)
			END::float8 AS zirkulationsquote
		FROM buecher_exemplare e
		JOIN buecher_titel t ON t.id = e.titel_id
		LEFT JOIN (SELECT DISTINCT exemplar_id FROM ausleihen WHERE rueckgabe_am IS NULL) al ON al.exemplar_id = e.id
		WHERE 1=1 %[2]s
	`, istVerlust, typeFilter)

	k := &bestandKennzahlen{}
	err := s.DB.Pool.QueryRow(ctx, q).Scan(
		&k.GesamtBestand, &k.AktiverBestand, &k.AktuellVerliehen, &k.VerloreneExemplare,
		&k.WiederbeschaffungswertDefekt, &k.VerlustQuote, &k.Zirkulationsquote,
	)
	if err != nil {
		return nil, err
	}
	return k, nil
}

// resolveZeitraumFilter mappt den ?zeitraum=-Parameter auf ein serverkontrolliertes
// SQL-Fragment für das Renner-Ranking (Werte sind nie nutzergesteuertes SQL).
func resolveZeitraumFilter(zeitraum string) string {
	switch zeitraum {
	case "schuljahr":
		// Current school year starts August 1st.
		return `AND a.ausgeliehen_am >= (
			CASE WHEN EXTRACT(MONTH FROM CURRENT_DATE) >= 8
				THEN make_date(EXTRACT(YEAR FROM CURRENT_DATE)::int, 8, 1)
				ELSE make_date(EXTRACT(YEAR FROM CURRENT_DATE)::int - 1, 8, 1)
			END
		)`
	case "monat":
		return "AND a.ausgeliehen_am >= CURRENT_DATE - INTERVAL '30 days'"
	default:
		return ""
	}
}

// queryPopularTitles liefert die meistausgeliehenen Titel (best-effort: bei einem
// Query- oder Iterationsfehler wird eine leere Liste statt eines Fehlers geliefert).
func (s *Server) queryPopularTitles(ctx context.Context, ausleihenFilter, typeFilter string, limit int) []PopularTitle {
	popularTitles := []PopularTitle{}
	q := fmt.Sprintf(`
		SELECT t.id, t.titel, coalesce(t.autor, ''), coalesce(t.cover_url, ''),
		       coalesce(t.subject, ''), coalesce(t.signatur, ''), coalesce(t.erscheinungsjahr, 0),
		       COUNT(a.id) AS count
		FROM buecher_titel t
		JOIN buecher_exemplare e ON t.id = e.titel_id
		JOIN ausleihen a ON e.id = a.exemplar_id
		WHERE 1=1 %s %s
		GROUP BY t.id, t.titel, t.autor, t.cover_url, t.subject, t.signatur, t.erscheinungsjahr
		ORDER BY count DESC
		LIMIT %d
	`, ausleihenFilter, typeFilter, limit)
	rows, err := s.DB.Pool.Query(ctx, q)
	if err != nil {
		return popularTitles
	}
	defer rows.Close()
	for rows.Next() {
		var p PopularTitle
		// Scan-Fehler nicht stillschweigend überspringen: Laufen Query und Struct
		// auseinander (Spalte ergänzt/entfernt), wäre die Liste sonst einfach leer —
		// nicht von "keine Treffer" zu unterscheiden.
		if err := rows.Scan(&p.ID, &p.Titel, &p.Autor, &p.CoverURL, &p.Fachbereich, &p.Systematik, &p.Erscheinungsjahr, &p.Count); err != nil {
			log.Printf("stats: Renner-Zeile unlesbar: %v", err)
			continue
		}
		popularTitles = append(popularTitles, p)
	}
	// Bei einem Abbruch mitten in der Iteration keine irreführende Teil-Top-Liste
	// zeigen (best-effort-Sektion, daher verwerfen statt 500).
	if err := rows.Err(); err != nil {
		return []PopularTitle{}
	}
	return popularTitles
}

// queryShelfWarmers liefert die Ladenhüter: entweder seit >2 Jahren nicht mehr
// ausgeliehen, ODER noch nie ausgeliehen UND bereits seit >2 Jahren im Bestand. Der
// Bestandsalter-Filter (MIN(e.erstellt_am)) verhindert, dass frisch gekaufte Neuzugänge
// — nie ausgeliehen, weil brandneu — sofort auf der Aussonderungsliste landen und
// versehentlich entsorgt werden. Best-effort mit leerer Liste bei Fehlern.
func (s *Server) queryShelfWarmers(ctx context.Context, typeFilter string, limit int) []ShelfWarmer {
	shelfWarmers := []ShelfWarmer{}
	// t.id mitliefern: Es wird ohnehin danach gruppiert (eine Zeile je Titel), war aber
	// nicht Teil der Projektion — dem Client fehlte damit der eindeutige Schlüssel.
	q := fmt.Sprintf(`
		SELECT t.id, t.titel, coalesce(t.autor, ''), coalesce(t.isbn, ''),
		       coalesce(t.subject, ''), coalesce(t.signatur, ''), coalesce(t.erscheinungsjahr, 0),
		       MAX(a.ausgeliehen_am) AS last_loan
		FROM buecher_titel t
		LEFT JOIN buecher_exemplare e ON t.id = e.titel_id
		LEFT JOIN ausleihen a ON e.id = a.exemplar_id
		WHERE 1=1 %s
		GROUP BY t.id, t.titel, t.autor, t.isbn, t.subject, t.signatur, t.erscheinungsjahr
		HAVING MAX(a.ausgeliehen_am) < NOW() - INTERVAL '2 years'
		    OR (MAX(a.ausgeliehen_am) IS NULL AND MIN(e.erstellt_am) < NOW() - INTERVAL '2 years')
		ORDER BY last_loan ASC NULLS FIRST
		LIMIT %d
	`, typeFilter, limit)
	rows, err := s.DB.Pool.Query(ctx, q)
	if err != nil {
		return shelfWarmers
	}
	defer rows.Close()
	for rows.Next() {
		var sw ShelfWarmer
		var lastLoan *time.Time
		// Wie oben: ein verschluckter Scan-Fehler sähe aus wie "keine Ladenhüter".
		if err := rows.Scan(&sw.ID, &sw.Titel, &sw.Autor, &sw.ISBN, &sw.Fachbereich, &sw.Systematik, &sw.Erscheinungsjahr, &lastLoan); err != nil {
			log.Printf("stats: Ladenhüter-Zeile unlesbar: %v", err)
			continue
		}
		sw.LetzteAusleihe = "Nie ausgeliehen"
		if lastLoan != nil {
			sw.LetzteAusleihe = lastLoan.Format(dateFormatDE)
		}
		shelfWarmers = append(shelfWarmers, sw)
	}
	// Bei Iterationsabbruch keine irreführende Teil-Ladenhüterliste zeigen.
	if err := rows.Err(); err != nil {
		return []ShelfWarmer{}
	}
	return shelfWarmers
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

		ausleihenFilter := resolveZeitraumFilter(r.URL.Query().Get("zeitraum"))
		typeFilter, typeName := resolveBestandsFilter(r.URL.Query().Get("type"))
		listLimit := resolveListLimit(r.URL.Query().Get("limit"))

		// 1. Beliebteste Titel (Die Renner) — inkl. Drill-Down-Feldern
		popularTitles := s.queryPopularTitles(ctx, ausleihenFilter, typeFilter, listLimit)

		// 2. Ladenhüter (No checkouts since 2 years or never) — inkl. Drill-Down-Feldern
		shelfWarmers := s.queryShelfWarmers(ctx, typeFilter, listLimit)

		// 3. Verlust-, Finanz- und Zirkulationskennzahlen (EIN aggregierter Scan)
		kennzahlen, err := s.queryBestandKennzahlen(ctx, typeFilter)
		if err != nil {
			log.Printf("stats: Bestandskennzahlen konnten nicht ermittelt werden: %v", err)
			kennzahlen = &bestandKennzahlen{}
		}

		RespondJSON(w, http.StatusOK, map[string]any{
			"filter_type":    typeName,
			"popular_titles": popularTitles,
			"shelf_warmers":  shelfWarmers,
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
