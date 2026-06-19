package api

import (
	"bytes"
	"fmt"

	"github.com/jung-kurt/gofpdf"
)

// GenerateLabelsPDF creates a standardized A4 PDF label sheet.
// formatId: identifies the label sheet (e.g. "zweckform_l4760")
// startPosition: 1-based index to start printing on the first page (to skip used labels)
// isQR: if true, a QR code is generated instead of a 1D Code39 barcode.
// items: the labels to print.
func GenerateLabelsPDF(formatId string, startPosition int, isQR bool, items []BarcodeLabelDetail) (*gofpdf.Fpdf, error) {
	format, _ := GetLabelFormat(formatId)

	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(format.MarginLeft, format.MarginTop, format.MarginLeft)
	pdf.SetAutoPageBreak(false, 0)
	pdf.AddPage()

	tr := pdf.UnicodeTranslatorFromDescriptor("")

	// Adjust start position (1-based to 0-based offset)
	offset := startPosition - 1
	if offset < 0 {
		offset = 0
	}

	labelsPerPage := format.Cols * format.Rows

	for i, item := range items {
		currentPos := offset + i

		// Page break logic
		if currentPos > 0 && currentPos%labelsPerPage == 0 {
			pdf.AddPage()
		}

		posInPage := currentPos % labelsPerPage
		colIdx := posInPage % format.Cols
		rowIdx := posInPage / format.Cols

		x := format.MarginLeft + float64(colIdx)*(format.LabelWidth+format.GapX)
		y := format.MarginTop + float64(rowIdx)*(format.LabelHeight+format.GapY)

		// Draw border (for debugging/cutting) - optional but useful for standard sheets, maybe toggleable?
		// To match the old A4 generation style exactly, we won't draw borders by default unless it's supplier order?
		// The old SupplierOrderHandler drew borders: pdf.Rect(x, y, colWidth, rowHeight, "D")
		// Let's NOT draw borders here to be clean, except if we want to. The old Avery LabelsHandler didn't draw borders.

		// Truncate title and author if they are too long
		titel := item.Titel
		if len(titel) > 40 {
			titel = titel[:37] + "..."
		}
		autor := item.Autor
		if len(autor) > 30 {
			autor = autor[:27] + "..."
		}

		// Print text and barcode inside label
		// To adapt dynamically, we do some proportional sizing
		if isQR {
			pdf.SetFont("Arial", "B", 8)
			pdf.SetXY(x+2, y+3)
			pdf.Cell(format.LabelWidth-4, 4, tr(titel))

			pdf.SetFont("Arial", "", 7)
			pdf.SetXY(x+2, y+7)
			pdf.Cell(format.LabelWidth-4, 4, tr(autor))

			// Generate dynamic QR code PNG
			barcodeImg, err := GenerateBarcodePNG(item.BarcodeID, true, 200, 200)
			if err == nil {
				imgReader := bytes.NewReader(barcodeImg)
				pdf.RegisterImageOptionsReader(item.BarcodeID, gofpdf.ImageOptions{ImageType: "PNG"}, imgReader)
				qrSize := 16.0
				if format.LabelHeight < 30 {
					qrSize = 12.0 // scale down for smaller labels like standard_52
				}
				qrX := x + (format.LabelWidth-qrSize)/2
				qrY := y + 11.0
				if format.LabelHeight < 30 {
					qrY = y + 8.0
				}
				pdf.Image(item.BarcodeID, qrX, qrY, qrSize, qrSize, false, "", 0, "")
			}

			// Barcode text
			pdf.SetFont("Arial", "B", 8)
			textY := y + 28
			if format.LabelHeight < 30 {
				textY = y + 21
			}
			pdf.SetXY(x+2, textY)
			pdf.CellFormat(format.LabelWidth-4, 4, tr(item.BarcodeID), "", 0, "C", false, 0, "")
		} else {
			// Code39 / Code128 layout
			pdf.SetFont("Arial", "B", 8)
			pdf.SetXY(x, y+4)
			pdf.CellFormat(format.LabelWidth, 4, tr("Schulbibliothek"), "", 0, "C", false, 0, "")

			pdf.SetFont("Arial", "", 8)
			pdf.SetXY(x, y+8)
			pdf.CellFormat(format.LabelWidth, 4, tr(titel), "", 0, "C", false, 0, "")

			bcWidth := 40.0
			bcHeight := 10.0
			if format.LabelWidth < 50 {
				bcWidth = 35.0
				bcHeight = 8.0
			}

			barcodeImg, err := GenerateBarcodePNG(item.BarcodeID, false, 250, 70)
			if err == nil {
				imgReader := bytes.NewReader(barcodeImg)
				imgName := fmt.Sprintf("1d_%s", item.BarcodeID)
				opt := gofpdf.ImageOptions{ImageType: "PNG"}
				pdf.RegisterImageOptionsReader(imgName, opt, imgReader)

				bcX := x + (format.LabelWidth-bcWidth)/2
				bcY := y + 14
				pdf.ImageOptions(imgName, bcX, bcY, bcWidth, bcHeight, false, opt, 0, "")
			}

			pdf.SetFont("Courier", "B", 10)
			textY := y + 26
			if format.LabelHeight < 30 {
				textY = y + 23
			}
			pdf.SetXY(x, textY)
			pdf.CellFormat(format.LabelWidth, 4, tr(item.BarcodeID), "", 0, "C", false, 0, "")
		}
	}

	return pdf, nil
}
