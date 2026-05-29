package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"bibliothek/apierrors"
	"github.com/jackc/pgx/v5"
	"github.com/jung-kurt/gofpdf"
)

// ReorderTitle represents a book title that has fallen below its reorder point.
type ReorderTitle struct {
	ID               string `json:"id"`
	Titel            string `json:"titel"`
	Autor            string `json:"autor"`
	ISBN             string `json:"isbn"`
	Verlag           string `json:"verlag"`
	CoverURL         string `json:"cover_url,omitempty"`
	Meldebestand     int    `json:"meldebestand"`
	VerfuegbarBestand int    `json:"verfuegbarer_bestand"`
}

// InventoryScanRequest is the payload for checking in an item during inventory.
type InventoryScanRequest struct {
	BarcodeID string `json:"barcode_id"`
}

// InventoryScanResponse yields the inventory status of the physical copy.
type InventoryScanResponse struct {
	BarcodeID       string `json:"barcode_id"`
	Titel           string `json:"titel"`
	CoverURL        string `json:"cover_url,omitempty"`
	ImRegalErwartet bool   `json:"im_regal_erwartet"`
	Status          string `json:"status"` // "Geprüft"
}

// GetReordersHandler lists all book titles below their reorder threshold.
func (s *Server) GetReordersHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		reorders, err := s.queryReorders(ctx)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(reorders)
	}
}

// ExportReordersPDFHandler exports the reorder list as a PDF.
func (s *Server) ExportReordersPDFHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
		defer cancel()

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

// ScanInventoryHandler registers copy scans in active inventory lists.
func (s *Server) ScanInventoryHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req InventoryScanRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			apierrors.SendHTTPError(w, http.StatusBadRequest, err)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		var copyID, title, coverURL string
		var isLent bool

		// Check copy metadata and current loan status
		query := `
			SELECT e.id, t.titel, coalesce(t.cover_url, ''), EXISTS (
				SELECT 1 FROM ausleihen a 
				WHERE a.exemplar_id = e.id AND a.rueckgabe_am IS NULL
			) AS is_lent
			FROM buecher_exemplare e
			JOIN buecher_titel t ON e.titel_id = t.id
			WHERE e.barcode_id = $1
			LIMIT 1
		`
		err := s.DB.Pool.QueryRow(ctx, query, req.BarcodeID).Scan(&copyID, &title, &coverURL, &isLent)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				apierrors.SendHTTPError(w, http.StatusNotFound, err)
				return
			}
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		// Update inventory audit timestamp on copy
		updateQuery := `
			UPDATE buecher_exemplare
			SET inventur_geprueft_am = CURRENT_TIMESTAMP,
			    aktualisiert_am = CURRENT_TIMESTAMP
			WHERE id = $1
		`
		_, err = s.DB.Pool.Exec(ctx, updateQuery, copyID)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(InventoryScanResponse{
			BarcodeID:       req.BarcodeID,
			Titel:           title,
			CoverURL:        coverURL,
			ImRegalErwartet: !isLent, // Should be on shelf if not currently checked out
			Status:          "Geprüft",
		})
	}
}

// GetStatisticsHandler returns analytical metadata details.
func (s *Server) GetStatisticsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		// 1. Top-Ausleiher (Klassen)
		topClasses := []any{}
		qTop := `
			SELECT s.klasse, COUNT(*) AS count
			FROM ausleihen a
			JOIN schueler s ON a.schueler_id = s.id
			GROUP BY s.klasse
			ORDER BY count DESC
			LIMIT 5
		`
		rows, err := s.DB.Pool.Query(ctx, qTop)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var class string
				var count int
				if err := rows.Scan(&class, &count); err == nil {
					topClasses = append(topClasses, map[string]any{"klasse": class, "count": count})
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

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"top_classes":   topClasses,
			"shelf_warmers": shelfWarmers,
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
			 WHERE e.titel_id = t.id AND e.ist_ausleihbar = true 
			   AND NOT EXISTS (SELECT 1 FROM ausleihen a WHERE a.exemplar_id = e.id AND a.rueckgabe_am IS NULL)
			) AS verfuegbar
		FROM buecher_titel t
		WHERE (
			SELECT COUNT(*) FROM buecher_exemplare e 
			WHERE e.titel_id = t.id AND e.ist_ausleihbar = true 
			  AND NOT EXISTS (SELECT 1 FROM ausleihen a WHERE a.exemplar_id = e.id AND a.rueckgabe_am IS NULL)
		) < t.meldebestand
		ORDER BY t.titel
	`
	rows, err := s.DB.Pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []ReorderTitle
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
