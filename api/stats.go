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

// GetStatisticsHandler returns analytical metadata details.
// Optional query parameter ?zeitraum=all|schuljahr|monat filters the loan-based
// popular_titles ranking by time period. shelf_warmers and loss_stats are global.
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

		// 1. Beliebteste Titel (Die Renner)
		popularTitles := []any{}
		qPopular := fmt.Sprintf(`
			SELECT t.id, t.titel, coalesce(t.autor, ''), coalesce(t.cover_url, ''), COUNT(a.id) AS count
			FROM buecher_titel t
			JOIN buecher_exemplare e ON t.id = e.titel_id
			JOIN ausleihen a ON e.id = a.exemplar_id
			WHERE 1=1 %s
			GROUP BY t.id, t.titel, t.autor, t.cover_url
			ORDER BY count DESC
			LIMIT 5
		`, ausleihenFilter)
		rows, err := s.DB.Pool.Query(ctx, qPopular)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var id, title, autor, coverURL string
				var count int
				if err := rows.Scan(&id, &title, &autor, &coverURL, &count); err == nil {
					popularTitles = append(popularTitles, map[string]any{
						"id":        id,
						"titel":     title,
						"autor":     autor,
						"cover_url": coverURL,
						"count":     count,
					})
				}
			}
		}

		// 2. Ladenhüter (No checkouts since 2 years or never)
		shelfWarmers := []any{}
		qWarmers := `
			SELECT t.titel, coalesce(t.autor, ''), coalesce(t.isbn, ''), MAX(a.ausgeliehen_am) AS last_loan
			FROM buecher_titel t
			LEFT JOIN buecher_exemplare e ON t.id = e.titel_id
			LEFT JOIN ausleihen a ON e.id = a.exemplar_id
			GROUP BY t.id, t.titel, t.autor, t.isbn
			HAVING MAX(a.ausgeliehen_am) < NOW() - INTERVAL '2 years'
			    OR MAX(a.ausgeliehen_am) IS NULL
			ORDER BY last_loan ASC NULLS FIRST
			LIMIT 5
		`
		rowsW, err := s.DB.Pool.Query(ctx, qWarmers)
		if err == nil {
			defer rowsW.Close()
			for rowsW.Next() {
				var title, autor, isbn string
				var lastLoan *time.Time
				if err := rowsW.Scan(&title, &autor, &isbn, &lastLoan); err == nil {
					var lastLoanStr = "Nie ausgeliehen"
					if lastLoan != nil {
						lastLoanStr = lastLoan.Format("02.01.2006")
					}
					shelfWarmers = append(shelfWarmers, map[string]any{
						"titel":      title,
						"autor":      autor,
						"isbn":       isbn,
						"letzte_aus": lastLoanStr,
					})
				}
			}
		}

		// 3. Verlustquote
		var gesamtBestand, verloreneExemplare int
		var verlustQuote float64
		qLoss := `
			SELECT 
				(SELECT COUNT(*) FROM buecher_exemplare) AS gesamt,
				(SELECT COUNT(DISTINCT exemplar_id) FROM schadensfaelle) AS verlorene,
				CASE 
				    WHEN (SELECT COUNT(*) FROM buecher_exemplare) = 0 THEN 0.0
				    ELSE ROUND(((SELECT COUNT(DISTINCT exemplar_id) FROM schadensfaelle) * 100.0) / (SELECT COUNT(*) FROM buecher_exemplare), 2)
				END AS quote
		`
		_ = s.DB.Pool.QueryRow(ctx, qLoss).Scan(&gesamtBestand, &verloreneExemplare, &verlustQuote)

		RespondJSON(w, http.StatusOK, map[string]any{
			"popular_titles": popularTitles,
			"shelf_warmers":  shelfWarmers,
			"loss_stats": map[string]any{
				"gesamt_bestand":      gesamtBestand,
				"verlorene_exemplare": verloreneExemplare,
				"verlust_quote":       verlustQuote,
			},
		})
	}
}

// queryReorders retrieves book titles below the reorder point.
func (s *Server) queryReorders(ctx context.Context) ([]ReorderTitle, error) {
	query := `
		SELECT t.id, t.titel, coalesce(t.autor, ''), coalesce(t.isbn, ''), coalesce(t.verlag, ''), coalesce(t.cover_url, ''), t.meldebestand,
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
	return results, nil
}
