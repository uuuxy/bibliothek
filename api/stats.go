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

	"bibliothek/apierrors"

	"github.com/jung-kurt/gofpdf"
)

// ReorderTitle represents a book title that has fallen below its reorder point.
type ReorderTitle struct {
	ID                string `json:"id"`
	Titel             string `json:"titel"`
	Autor             string `json:"autor"`
	ISBN              string `json:"isbn"`
	Verlag            string `json:"verlag"`
	CoverURL          string `json:"cover_url,omitempty"`
	Meldebestand      int    `json:"meldebestand"`
	VerfuegbarBestand int    `json:"verfuegbarer_bestand"`
}

// GetReordersHandler lists all book titles below their reorder threshold.
func (s *Server) GetReordersHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		reorders, err := s.queryReorders(ctx)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		RespondJSON(w, http.StatusOK, reorders)
	}
}

// ExportReordersPDFHandler exports the reorder list as a PDF.
func (s *Server) ExportReordersPDFHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		reorders, err := s.queryReorders(ctx)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		pdf := gofpdf.New("P", "mm", "A4", "")
		pdf.AddPage()
		pdf.SetMargins(15, 15, 15)
		tr := pdf.UnicodeTranslatorFromDescriptor("")

		// PDF Title
		pdf.SetFont("Arial", "B", 16)
		pdf.Cell(0, 10, tr("Schulbibliothek - Bestellliste"))
		pdf.Ln(6)
		pdf.SetFont("Arial", "I", 9)
		pdf.SetTextColor(100, 100, 100)
		pdf.Cell(0, 5, tr(fmt.Sprintf("Generiert am %s", time.Now().Format("02.01.2006 (15:04)"))))
		pdf.SetTextColor(0, 0, 0)
		pdf.Ln(12)

		// Table Headers
		pdf.SetFont("Arial", "B", 9)
		pdf.SetFillColor(230, 230, 230)
		pdf.CellFormat(70, 7, tr("Buchtitel"), "1", 0, "L", true, 0, "")
		pdf.CellFormat(40, 7, tr("Autor"), "1", 0, "L", true, 0, "")
		pdf.CellFormat(35, 7, tr("ISBN"), "1", 0, "L", true, 0, "")
		pdf.CellFormat(12, 7, tr("Melde."), "1", 0, "C", true, 0, "")
		pdf.CellFormat(12, 7, tr("Verf."), "1", 0, "C", true, 0, "")
		pdf.CellFormat(11, 7, tr("Nach."), "1", 1, "C", true, 0, "")

		pdf.SetFont("Arial", "", 8)
		for _, b := range reorders {
			pdf.CellFormat(70, 6, tr(b.Titel), "1", 0, "L", false, 0, "")
			pdf.CellFormat(40, 6, tr(b.Autor), "1", 0, "L", false, 0, "")
			pdf.CellFormat(35, 6, tr(b.ISBN), "1", 0, "L", false, 0, "")
			pdf.CellFormat(12, 6, strconv.Itoa(b.Meldebestand), "1", 0, "C", false, 0, "")
			pdf.CellFormat(12, 6, strconv.Itoa(b.VerfuegbarBestand), "1", 0, "C", false, 0, "")
			pdf.CellFormat(11, 6, strconv.Itoa(b.Meldebestand-b.VerfuegbarBestand), "1", 1, "C", false, 0, "")
		}

		w.Header().Set("Content-Type", "application/pdf")
		w.Header().Set("Content-Disposition", "attachment; filename=bestellliste.pdf")
		if err := pdf.Output(w); err != nil {
			log.Printf("Stats: PDF stream output failed: %v", err)
		}
	}
}

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
	VerloreneExemplare int
	// Wiederbeschaffungswert: Summe der Einkaufspreise aller Exemplare, die
	// ausgesondert sind ODER einen Schadensfall haben (verloren/defekt).
	WiederbeschaffungswertDefekt float64
	VerlustQuote                 float64 // verlorene / Gesamtbestand (Def. wie bisher)
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
// SQL-Fragment. LMF-Bestand ist per Projekt-Konvention am Titel-Präfix „lmf-"
// erkennbar (dieselbe Regel wie im Ausleih-Limit, loan_checkout.go).
func resolveBestandsFilter(typ string) (fragment string, normalized string) {
	switch typ {
	case "lmf":
		return "AND LOWER(t.titel) LIKE 'lmf-%'", "lmf"
	case "freihand":
		return "AND LOWER(t.titel) NOT LIKE 'lmf-%'", "freihand"
	default:
		return "", "alle"
	}
}

// queryBestandKennzahlen liefert Verlust-, Finanz- und Zirkulationszahlen in
// einem einzigen aggregierten Statement.
func (s *Server) queryBestandKennzahlen(ctx context.Context, typeFilter string) (*bestandKennzahlen, error) {
	q := fmt.Sprintf(`
		SELECT
			COUNT(*)::int AS gesamt,
			COUNT(*) FILTER (WHERE NOT e.ist_ausgesondert)::int AS aktiv,
			COUNT(*) FILTER (WHERE al.exemplar_id IS NOT NULL AND NOT e.ist_ausgesondert)::int AS verliehen,
			COUNT(*) FILTER (WHERE sf.exemplar_id IS NOT NULL)::int AS verlorene,
			COALESCE(SUM(e.einkaufspreis) FILTER (WHERE e.ist_ausgesondert OR sf.exemplar_id IS NOT NULL), 0)::float8 AS wiederbeschaffung,
			CASE WHEN COUNT(*) = 0 THEN 0.0
				 ELSE ROUND(COUNT(*) FILTER (WHERE sf.exemplar_id IS NOT NULL) * 100.0 / COUNT(*), 2)
			END::float8 AS verlust_quote,
			CASE WHEN COUNT(*) FILTER (WHERE NOT e.ist_ausgesondert) = 0 THEN 0.0
				 ELSE ROUND(COUNT(*) FILTER (WHERE al.exemplar_id IS NOT NULL AND NOT e.ist_ausgesondert) * 100.0
					   / COUNT(*) FILTER (WHERE NOT e.ist_ausgesondert), 2)
			END::float8 AS zirkulationsquote
		FROM buecher_exemplare e
		JOIN buecher_titel t ON t.id = e.titel_id
		LEFT JOIN (SELECT DISTINCT exemplar_id FROM ausleihen WHERE rueckgabe_am IS NULL) al ON al.exemplar_id = e.id
		LEFT JOIN (SELECT DISTINCT exemplar_id FROM schadensfaelle) sf ON sf.exemplar_id = e.id
		WHERE 1=1 %s
	`, typeFilter)

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

// GetStatisticsHandler returns analytical metadata details.
// Optional query parameters:
//   - ?zeitraum=all|schuljahr|monat filtert das Renner-Ranking zeitlich.
//   - ?type=lmf|freihand filtert ALLE Kennzahlen und Listen auf den
//     LMF-Bestand (Lernmittel, Titel-Präfix „lmf-") bzw. die Schülerbücherei.
//     Ohne Parameter: Gesamtbestand.
func (s *Server) GetStatisticsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Resolve time filter for popular_titles query.
		// Values are server-controlled strings, never user-provided SQL fragments.
		var ausleihenFilter string
		switch r.URL.Query().Get("zeitraum") {
		case "schuljahr":
			// Current school year starts August 1st.
			ausleihenFilter = `AND a.ausgeliehen_am >= (
				CASE WHEN EXTRACT(MONTH FROM CURRENT_DATE) >= 8
					THEN make_date(EXTRACT(YEAR FROM CURRENT_DATE)::int, 8, 1)
					ELSE make_date(EXTRACT(YEAR FROM CURRENT_DATE)::int - 1, 8, 1)
				END
			)`
		case "monat":
			ausleihenFilter = "AND a.ausgeliehen_am >= CURRENT_DATE - INTERVAL '30 days'"
		default:
			ausleihenFilter = ""
		}

		typeFilter, typeName := resolveBestandsFilter(r.URL.Query().Get("type"))
		listLimit := resolveListLimit(r.URL.Query().Get("limit"))

		// 1. Beliebteste Titel (Die Renner) — inkl. Drill-Down-Feldern
		popularTitles := []PopularTitle{}
		qPopular := fmt.Sprintf(`
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
		`, ausleihenFilter, typeFilter, listLimit)
		rows, err := s.DB.Pool.Query(ctx, qPopular)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var p PopularTitle
				if err := rows.Scan(&p.ID, &p.Titel, &p.Autor, &p.CoverURL, &p.Fachbereich, &p.Systematik, &p.Erscheinungsjahr, &p.Count); err == nil {
					popularTitles = append(popularTitles, p)
				}
			}
			// Bei einem Abbruch mitten in der Iteration keine irreführende Teil-Top-Liste
			// zeigen (best-effort-Sektion, daher verwerfen statt 500).
			if err := rows.Err(); err != nil {
				popularTitles = []PopularTitle{}
			}
		}

		// 2. Ladenhüter (No checkouts since 2 years or never) — inkl. Drill-Down-Feldern
		shelfWarmers := []ShelfWarmer{}
		qWarmers := fmt.Sprintf(`
			SELECT t.titel, coalesce(t.autor, ''), coalesce(t.isbn, ''),
			       coalesce(t.subject, ''), coalesce(t.signatur, ''), coalesce(t.erscheinungsjahr, 0),
			       MAX(a.ausgeliehen_am) AS last_loan
			FROM buecher_titel t
			LEFT JOIN buecher_exemplare e ON t.id = e.titel_id
			LEFT JOIN ausleihen a ON e.id = a.exemplar_id
			WHERE 1=1 %s
			GROUP BY t.id, t.titel, t.autor, t.isbn, t.subject, t.signatur, t.erscheinungsjahr
			HAVING MAX(a.ausgeliehen_am) < NOW() - INTERVAL '2 years'
			    OR MAX(a.ausgeliehen_am) IS NULL
			ORDER BY last_loan ASC NULLS FIRST
			LIMIT %d
		`, typeFilter, listLimit)
		rowsW, err := s.DB.Pool.Query(ctx, qWarmers)
		if err == nil {
			defer rowsW.Close()
			for rowsW.Next() {
				var sw ShelfWarmer
				var lastLoan *time.Time
				if err := rowsW.Scan(&sw.Titel, &sw.Autor, &sw.ISBN, &sw.Fachbereich, &sw.Systematik, &sw.Erscheinungsjahr, &lastLoan); err == nil {
					sw.LetzteAusleihe = "Nie ausgeliehen"
					if lastLoan != nil {
						sw.LetzteAusleihe = lastLoan.Format("02.01.2006")
					}
					shelfWarmers = append(shelfWarmers, sw)
				}
			}
			// Bei Iterationsabbruch keine irreführende Teil-Ladenhüterliste zeigen.
			if err := rowsW.Err(); err != nil {
				shelfWarmers = []ShelfWarmer{}
			}
		}

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

// queryReorders retrieves book titles below the reorder point.
func (s *Server) queryReorders(ctx context.Context) ([]ReorderTitle, error) {
	query := `
		SELECT t.id, t.titel, coalesce(t.autor, ''), coalesce(t.isbn, ''), coalesce(t.verlag, ''), 
		       COALESCE(NULLIF(t.cover_url, ''), CASE WHEN t.isbn IS NOT NULL AND t.isbn != '' THEN 'https://portal.dnb.de/opac/mvb/cover?isbn=' || replace(t.isbn, '-', '') ELSE '' END),
		       t.meldebestand,
			(SELECT COUNT(*) FROM buecher_exemplare e 
			 WHERE e.titel_id = t.id AND e.ist_ausleihbar = true AND e.ist_ausgesondert = false
			   AND NOT EXISTS (SELECT 1 FROM ausleihen a WHERE a.exemplar_id = e.id AND a.rueckgabe_am IS NULL)
			) AS verfuegbar
		FROM buecher_titel t
		WHERE (
			SELECT COUNT(*) FROM buecher_exemplare e 
			WHERE e.titel_id = t.id AND e.ist_ausleihbar = true AND e.ist_ausgesondert = false
			  AND NOT EXISTS (SELECT 1 FROM ausleihen a WHERE a.exemplar_id = e.id AND a.rueckgabe_am IS NULL)
		) < t.meldebestand
		ORDER BY t.titel
	`
	rows, err := s.DB.Pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := make([]ReorderTitle, 0)
	for rows.Next() {
		var r ReorderTitle
		err := rows.Scan(&r.ID, &r.Titel, &r.Autor, &r.ISBN, &r.Verlag, &r.CoverURL, &r.Meldebestand, &r.VerfuegbarBestand)
		if err != nil {
			return nil, err
		}
		results = append(results, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return results, nil
}
