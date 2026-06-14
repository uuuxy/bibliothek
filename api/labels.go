package api

import (
	"bytes"
	"context"
	"fmt"
	"image/png"
	"log"
	"net/http"
	"time"

	"bibliothek/apierrors"

	"github.com/boombuler/barcode"
	"github.com/boombuler/barcode/code128"
	"github.com/jung-kurt/gofpdf"
)

// LabelsHandler returns a handler that generates an A4 PDF containing 3x8 Avery labels
// for all copies of a given book title.
func (s *Server) LabelsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if id == "" {
			apierrors.SendHTTPError(w, http.StatusBadRequest, fmt.Errorf("id is required"))
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		// Fetch the book title
		var titel string
		err := s.DB.Pool.QueryRow(ctx, "SELECT titel FROM buecher_titel WHERE id = $1", id).Scan(&titel)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusNotFound, fmt.Errorf("titel nicht gefunden: %w", err))
			return
		}

		// Truncate title if it's too long
		if len(titel) > 40 {
			titel = titel[:37] + "..."
		}

		// Fetch all barcodes for this title
		rows, err := s.DB.Pool.Query(ctx, "SELECT barcode_id FROM buecher_exemplare WHERE titel_id = $1 ORDER BY barcode_id", id)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, fmt.Errorf("fehler beim laden der exemplare: %w", err))
			return
		}
		defer rows.Close()

		var barcodes []string
		for rows.Next() {
			var b string
			if err := rows.Scan(&b); err == nil {
				barcodes = append(barcodes, b)
			}
		}

		if len(barcodes) == 0 {
			apierrors.SendHTTPError(w, http.StatusBadRequest, fmt.Errorf("keine exemplare für diesen titel vorhanden"))
			return
		}

		pdf := gofpdf.New("P", "mm", "A4", "")
		pdf.SetMargins(0, 0, 0)
		pdf.SetAutoPageBreak(false, 0)

		// Avery 3475/3490 (70mm x 36mm or 37mm) Grid Variables
		columns := 3
		rowsPerPage := 8
		labelWidth := 70.0
		labelHeight := 37.0
		marginTop := 0.5 // Standard upper margin for 37mm rows is ~0-0.5mm
		marginLeft := 0.0

		pdf.AddPage()

		for i, bcText := range barcodes {
			// Page break if necessary
			if i > 0 && i%(columns*rowsPerPage) == 0 {
				pdf.AddPage()
			}

			// Position on the grid
			posInPage := i % (columns * rowsPerPage)
			col := posInPage % columns
			row := posInPage / columns

			x := marginLeft + (float64(col) * labelWidth)
			y := marginTop + (float64(row) * labelHeight)

			// Generate Barcode image
			bcGen, err := code128.Encode(bcText)
			if err != nil {
				continue
			}
			// Scale barcode (e.g. 200x50 pixels)
			bcScaled, err := barcode.Scale(bcGen, 200, 50)
			if err != nil {
				continue
			}

			// Encode PNG to buffer
			var buf bytes.Buffer
			_ = png.Encode(&buf, bcScaled)

			// Register Image
			imgName := fmt.Sprintf("bc_%s", bcText)
			opt := gofpdf.ImageOptions{ImageType: "PNG", ReadDpi: true}
			pdf.RegisterImageOptionsReader(imgName, opt, &buf)

			// Draw Label
			pdf.SetFont("Arial", "B", 8)
			pdf.SetXY(x, y+4)
			pdf.CellFormat(labelWidth, 4, "Schulbibliothek", "", 0, "C", false, 0, "")

			pdf.SetFont("Arial", "", 8)
			pdf.SetXY(x, y+8)
			pdf.CellFormat(labelWidth, 4, titel, "", 0, "C", false, 0, "")

			// Draw Barcode Image
			// Center barcode: Width = 40mm, Height = 10mm
			bcWidth := 40.0
			bcHeight := 10.0
			bcX := x + (labelWidth-bcWidth)/2
			bcY := y + 14
			pdf.ImageOptions(imgName, bcX, bcY, bcWidth, bcHeight, false, opt, 0, "")

			// Draw Barcode Text
			pdf.SetFont("Courier", "B", 10)
			pdf.SetXY(x, y+26)
			pdf.CellFormat(labelWidth, 4, bcText, "", 0, "C", false, 0, "")
		}

		w.Header().Set("Content-Type", "application/pdf")
		w.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=\"etiketten_%s.pdf\"", id))

		err = pdf.Output(w)
		if err != nil {
			log.Printf("Fehler beim Senden des PDFs: %v", err)
		}
	}
}
