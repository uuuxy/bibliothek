package api

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"bibliothek/apierrors"
)

// parseLabelParams liest Format, Startposition und QR-Flag aus den Query-Parametern
// (mit denselben Defaults wie bisher).
func parseLabelParams(r *http.Request) (formatId string, startPos int, isQR bool) {
	formatId = r.URL.Query().Get("format")
	if formatId == "" {
		formatId = "avery_3475" // default as before
	}

	startPos = 1
	if startParam := r.URL.Query().Get("start"); startParam != "" {
		if parsed, err := strconv.Atoi(startParam); err == nil && parsed > 0 {
			startPos = parsed
		}
	}

	isQR = r.URL.Query().Get("qr") == "true"
	return formatId, startPos, isQR
}

// queryLabelItems lädt alle Exemplare (Barcode, Titel, Autor) eines Titels.
func (s *Server) queryLabelItems(ctx context.Context, id string) ([]BarcodeLabelDetail, error) {
	query := `
		SELECT e.barcode_id, t.titel, coalesce(t.autor, '')
		FROM buecher_exemplare e
		JOIN buecher_titel t ON e.titel_id = t.id
		WHERE e.titel_id = $1
		ORDER BY e.barcode_id
	`
	rows, err := s.DB.Pool.Query(ctx, query, id)
	if err != nil {
		return nil, fmt.Errorf("fehler beim laden der exemplare: %w", err)
	}
	defer rows.Close()

	var items []BarcodeLabelDetail
	for rows.Next() {
		var item BarcodeLabelDetail
		if err := rows.Scan(&item.BarcodeID, &item.Titel, &item.Autor); err == nil {
			items = append(items, item)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("datenbankfehler: %w", err)
	}
	return items, nil
}

// LabelsHandler returns a handler that generates an A4 PDF containing 3x8 Avery labels
// for all copies of a given book title.
func (s *Server) LabelsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if id == "" {
			apierrors.SendHTTPError(w, http.StatusBadRequest, fmt.Errorf("id is required"))
			return
		}

		ctx := r.Context()
		formatId, startPos, isQR := parseLabelParams(r)

		items, err := s.queryLabelItems(ctx, id)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		if len(items) == 0 {
			apierrors.SendHTTPError(w, http.StatusBadRequest, fmt.Errorf("keine exemplare für diesen titel vorhanden"))
			return
		}

		pdf, err := GenerateLabelsPDF(formatId, startPos, isQR, items)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, fmt.Errorf("fehler bei der pdf generierung: %w", err))
			return
		}

		w.Header().Set(headerContentType, contentTypePDF)
		w.Header().Set(headerContentDisposition, fmt.Sprintf("inline; filename=\"etiketten_%s.pdf\"", id))

		if err := pdf.Output(w); err != nil {
			log.Printf("Fehler beim Senden des PDFs: %v", err)
		}
	}
}

// PrintLabelsRequest represents a request to generate a PDF label sheet.
type PrintLabelsRequest struct {
	FormatID      string               `json:"formatId"`
	StartPosition int                  `json:"startPosition"`
	IsQR          bool                 `json:"isQR"`
	Items         []BarcodeLabelDetail `json:"items"`
}

// PrintLabelsHandler generates an A4 PDF containing labels dynamically.
func (s *Server) PrintLabelsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req PrintLabelsRequest
		if !DecodeAndValidate(w, r, &req) {
			return
		}

		if len(req.Items) == 0 {
			apierrors.SendHTTPError(w, http.StatusBadRequest, fmt.Errorf("keine exemplare angegeben"))
			return
		}

		pdf, err := GenerateLabelsPDF(req.FormatID, req.StartPosition, req.IsQR, req.Items)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, fmt.Errorf("fehler bei der pdf generierung: %w", err))
			return
		}

		w.Header().Set(headerContentType, contentTypePDF)
		w.Header().Set(headerContentDisposition, "inline; filename=\"etiketten_custom.pdf\"")

		if err := pdf.Output(w); err != nil {
			log.Printf("Fehler beim Senden des PDFs: %v", err)
		}
	}
}
