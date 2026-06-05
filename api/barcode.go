package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"bibliothek/apierrors"
	"bibliothek/utils"

	"github.com/boombuler/barcode"
	"github.com/boombuler/barcode/code39"
	"github.com/boombuler/barcode/qr"
	"github.com/jackc/pgx/v5"
	"github.com/jung-kurt/gofpdf"
)

// OrderRequest holds the input parameters for generating a new supplier order.
type OrderRequest struct {
	TitelID string `json:"titel_id"`
	Menge   int    `json:"menge"`
}

// GenerateBarcodePNG creates a high-resolution PNG barcode image from a string.
// Supports Code39 and QR-code. Scales the output to the specified dimensions.
func GenerateBarcodePNG(content string, isQR bool, width, height int) ([]byte, error) {
	var bc barcode.Barcode
	var err error

	if isQR {
		bc, err = qr.Encode(content, qr.M, qr.Auto)
	} else {
		// Code39 is case-sensitive, capitalize content for compatibility
		bc, err = code39.Encode(strings.ToUpper(content), true, true)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to encode barcode: %w", err)
	}

	scaled, err := barcode.Scale(bc, width, height)
	if err != nil {
		return nil, fmt.Errorf("failed to scale barcode: %w", err)
	}

	// Convert the scaled barcode to standard 8-bit RGBA image
	// to avoid 16-bit PNG depth which gofpdf PNG parser doesn't support.
	bounds := scaled.Bounds()
	rgbaImg := image.NewRGBA(bounds)
	draw.Draw(rgbaImg, bounds, scaled, bounds.Min, draw.Src)

	var buf bytes.Buffer
	if err := png.Encode(&buf, rgbaImg); err != nil {
		return nil, fmt.Errorf("failed to encode PNG: %w", err)
	}

	return buf.Bytes(), nil
}

// BarcodeHandler handles on-demand PNG barcode and QR code generation.
func (s *Server) BarcodeHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		content := r.URL.Query().Get("content")
		if content == "" {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("missing content parameter"))
			return
		}

		isQR := r.URL.Query().Get("qr") == "true"

		// Set default size metrics
		width := 300
		height := 100
		if isQR {
			width = 200
			height = 200
		}

		if wStr := r.URL.Query().Get("width"); wStr != "" {
			if parsed, err := strconv.Atoi(wStr); err == nil {
				width = parsed
			}
		}
		if hStr := r.URL.Query().Get("height"); hStr != "" {
			if parsed, err := strconv.Atoi(hStr); err == nil {
				height = parsed
			}
		}

		pngBytes, err := GenerateBarcodePNG(content, isQR, width, height)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Cache-Control", "public, max-age=31536000") // Cache for 1 year
		_, _ = w.Write(pngBytes)
	}
}

// SupplierOrderHandler processes new book orders. Generates sequential B- barcodes,
// registers the new copies in the DB, and builds a print-ready PDF containing barcode sheets.
func (s *Server) SupplierOrderHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req OrderRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			apierrors.SendHTTPError(w, http.StatusBadRequest, err)
			return
		}

		if req.Menge <= 0 || req.Menge > 200 {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("quantity must be between 1 and 200"))
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
		defer cancel()

		tx, err := s.DB.Pool.Begin(ctx)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		defer tx.Rollback(ctx)

		// 1. Resolve master title details
		var titel, autor string
		err = tx.QueryRow(ctx, "SELECT titel, coalesce(autor, '') FROM buecher_titel WHERE id = $1", req.TitelID).Scan(&titel, &autor)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				apierrors.SendHTTPError(w, http.StatusNotFound, err)
				return
			}
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		// 2. Fetch the highest B-XXXXX barcode in the system to calculate the next sequence
		startNum, err := utils.GetNextBarcodeSequence(ctx, tx, "buecher_exemplare", "B", false)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, fmt.Errorf("failed to get next barcode: %w", err))
			return
		}

		// 3. Register copies in DB (marked as not borrowable until delivery)
		newBarcodes := []string{}
		qInsert := `
			INSERT INTO buecher_exemplare (titel_id, barcode_id, zustand_notiz, ist_ausleihbar)
			VALUES ($1, $2, 'Bestellt (Lieferanten-Vorab-Barcode)', false)
		`
		for i := 0; i < req.Menge; i++ {
			barcodeID := fmt.Sprintf("B-%05d", startNum+i)
			_, err = tx.Exec(ctx, qInsert, req.TitelID, barcodeID)
			if err != nil {
				apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
				return
			}
			newBarcodes = append(newBarcodes, barcodeID)
		}

		if err := tx.Commit(ctx); err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		// 4. Generate printable PDF label sheets
		pdf := gofpdf.New("P", "mm", "A4", "")
		pdf.AddPage()
		pdf.SetMargins(10, 15, 10)
		tr := pdf.UnicodeTranslatorFromDescriptor("")

		pdf.SetFont("Arial", "B", 14)
		pdf.Cell(0, 10, tr("Barcode-Aufkleber für Buchlieferant (Vorab-Beklebung)"))
		pdf.Ln(6)
		pdf.SetFont("Arial", "", 9)
		pdf.SetTextColor(100, 100, 100)
		pdf.Cell(0, 4, tr(fmt.Sprintf("Bestellung: %d x \"%s\" · Generiert am %s", req.Menge, titel, time.Now().Format("02.01.2006"))))
		pdf.SetTextColor(0, 0, 0)
		pdf.Ln(10)

		// Define label grid metrics (A4 sheet spacing)
		colWidth := 60.0
		rowHeight := 35.0
		cols := 3
		margin := 10.0

		trTitel := tr(titel)
		trAutor := tr(autor)

		for idx, barcodeID := range newBarcodes {
			colIdx := idx % cols
			rowIdx := (idx / cols) % 7 // 7 rows per page

			// Add page if row overflows
			if idx > 0 && colIdx == 0 && idx%21 == 0 {
				pdf.AddPage()
			}

			x := margin + float64(colIdx)*(colWidth+5)
			y := 25.0 + float64(rowIdx)*(rowHeight+5)

			// Draw border box around each label
			pdf.Rect(x, y, colWidth, rowHeight, "D")

			// Print metadata inside label
			pdf.SetFont("Arial", "B", 8)
			pdf.SetXY(x+2, y+3)
			pdf.Cell(colWidth-4, 4, trTitel)

			pdf.SetFont("Arial", "", 7)
			pdf.SetXY(x+2, y+7)
			pdf.Cell(colWidth-4, 4, trAutor)

			// Generate and register barcode PNG from memory
			barcodeImg, err := GenerateBarcodePNG(barcodeID, false, 250, 70)
			if err == nil {
				imgReader := bytes.NewReader(barcodeImg)
				imgInfo := pdf.RegisterImageOptionsReader(barcodeID, gofpdf.ImageOptions{ImageType: "PNG"}, imgReader)
				if imgInfo != nil {
					// Embed barcode graphic
					pdf.Image(barcodeID, x+5, y+12, colWidth-10, 13, false, "", 0, "")
				}
			}

			// Render barcode ID string underneath image
			pdf.SetFont("Arial", "B", 8)
			pdf.SetXY(x+2, y+27)
			pdf.CellFormat(colWidth-4, 4, tr(barcodeID), "", 0, "C", false, 0, "")
		}

		w.Header().Set("Content-Type", "application/pdf")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=bestellung_barcodes_%d.pdf", startNum))
		if err := pdf.Output(w); err != nil {
			log.Printf("Barcode: PDF streaming failure: %v", err)
		}
	}
}
