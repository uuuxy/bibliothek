package api

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"time"

	"github.com/jung-kurt/gofpdf"
)

// OrderedItem represents a single book title ordered with its quantity.
type OrderedItem struct {
	Titel  string
	Autor  string
	ISBN   string
	Verlag string
	Menge  int
}

// BarcodeLabelDetail holds data needed to print a barcode label.
type BarcodeLabelDetail struct {
	BarcodeID string
	Titel     string
	Autor     string
	ISBN      string
}

// GenerateOrderSummaryPDF generates a PDF cover letter ("Bestellanschreiben") containing the table of ordered book titles.
func GenerateOrderSummaryPDF(items []OrderedItem) ([]byte, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()
	pdf.SetMargins(20, 20, 20)
	tr := pdf.UnicodeTranslatorFromDescriptor("")

	// Letter Header / Sender info
	pdf.SetFont("Arial", "B", 12)
	pdf.Cell(0, 8, tr("Städtisches Gymnasium Musterstadt - Schulbibliothek"))
	pdf.Ln(5)
	pdf.SetFont("Arial", "", 8)
	pdf.SetTextColor(100, 100, 100)
	pdf.Cell(0, 4, tr("Schulbibliothek · Lindenallee 4 · 12345 Musterstadt"))
	pdf.SetTextColor(0, 0, 0)
	pdf.Ln(15)

	// Date (Right-aligned)
	pdf.SetFont("Arial", "", 10)
	pdf.CellFormat(0, 6, fmt.Sprintf("Musterstadt, den %s", time.Now().Format("02.01.2006")), "", 0, "R", false, 0, "")
	pdf.Ln(10)

	// Recipient Block
	pdf.SetFont("Arial", "B", 9)
	pdf.Cell(0, 4, tr("An den Buchlieferanten"))
	pdf.Ln(20)

	// Subject
	pdf.SetFont("Arial", "B", 12)
	pdf.Cell(0, 8, tr("Buchbestellung für die Schulbibliothek"))
	pdf.Ln(10)

	// Letter Body Text
	pdf.SetFont("Arial", "", 10)
	bodyText := "Sehr geehrte Damen und Herren,\n\n" +
		"hiermit bestellen wir für unsere Schulbibliothek die nachfolgend aufgeführten Buchtitel zur Lieferung.\n" +
		"Bitte versehen Sie die gelieferten Exemplare vorab mit den Barcode/QR-Code-Aufklebern aus dem beigefügten Bogen.\n" +
		"Die Rechnung senden Sie bitte an die oben angegebene Anschrift.\n\n" +
		"Bestellte Titel:"
	pdf.MultiCell(0, 5, tr(bodyText), "", "L", false)
	pdf.Ln(6)

	// Table headers
	pdf.SetFont("Arial", "B", 9)
	pdf.SetFillColor(230, 230, 230)
	pdf.CellFormat(75, 8, tr("Buchtitel"), "1", 0, "L", true, 0, "")
	pdf.CellFormat(40, 8, tr("Autor"), "1", 0, "L", true, 0, "")
	pdf.CellFormat(35, 8, tr("ISBN"), "1", 0, "L", true, 0, "")
	pdf.CellFormat(20, 8, tr("Menge"), "1", 1, "C", true, 0, "")

	pdf.SetFont("Arial", "", 9)
	for _, item := range items {
		pdf.CellFormat(75, 7, tr(item.Titel), "1", 0, "L", false, 0, "")
		pdf.CellFormat(40, 7, tr(item.Autor), "1", 0, "L", false, 0, "")
		pdf.CellFormat(35, 7, tr(item.ISBN), "1", 0, "L", false, 0, "")
		pdf.CellFormat(20, 7, fmt.Sprintf("%d", item.Menge), "1", 1, "C", false, 0, "")
	}
	pdf.Ln(15)

	// Sign-off
	pdf.SetFont("Arial", "", 10)
	pdf.Cell(0, 6, tr("Mit freundlichen Grüßen,"))
	pdf.Ln(12)
	pdf.SetFont("Arial", "B", 10)
	pdf.Cell(0, 6, tr("Das Bibliotheksteam"))

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// GenerateBarcodeSheetPDF generates a grid of QR code labels on A4 PDF format.
func GenerateBarcodeSheetPDF(labels []BarcodeLabelDetail) ([]byte, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()
	pdf.SetMargins(10, 15, 10)
	tr := pdf.UnicodeTranslatorFromDescriptor("")

	pdf.SetFont("Arial", "B", 14)
	pdf.Cell(0, 10, tr("QR-Code-Aufkleber für Buchlieferung (Vorab-Beklebung)"))
	pdf.Ln(6)
	pdf.SetFont("Arial", "", 9)
	pdf.SetTextColor(100, 100, 100)
	pdf.Cell(0, 4, tr(fmt.Sprintf("Generiert am %s · Gesamtanzahl: %d", time.Now().Format("02.01.2006"), len(labels))))
	pdf.SetTextColor(0, 0, 0)
	pdf.Ln(10)

	colWidth := 60.0
	rowHeight := 35.0
	cols := 3
	margin := 10.0

	for idx, label := range labels {
		colIdx := idx % cols
		rowIdx := (idx / cols) % 7

		if idx > 0 && colIdx == 0 && idx%21 == 0 {
			pdf.AddPage()
		}

		x := margin + float64(colIdx)*(colWidth+5)
		y := 25.0 + float64(rowIdx)*(rowHeight+5)

		pdf.Rect(x, y, colWidth, rowHeight, "D")

		pdf.SetFont("Arial", "B", 8)
		pdf.SetXY(x+2, y+3)
		pdf.Cell(colWidth-4, 4, tr(label.Titel))

		pdf.SetFont("Arial", "", 7)
		pdf.SetXY(x+2, y+7)
		pdf.Cell(colWidth-4, 4, tr(label.Autor))

		// Generate dynamic QR code PNG in memory
		barcodeImg, err := GenerateBarcodePNG(label.BarcodeID, true, 200, 200)
		if err == nil {
			imgReader := bytes.NewReader(barcodeImg)
			pdf.RegisterImageOptionsReader(label.BarcodeID, gofpdf.ImageOptions{ImageType: "PNG"}, imgReader)
			qrSize := 16.0
			qrX := x + (colWidth-qrSize)/2
			qrY := y + 11.0
			pdf.Image(label.BarcodeID, qrX, qrY, qrSize, qrSize, false, "", 0, "")
		}

		pdf.SetFont("Arial", "B", 8)
		pdf.SetXY(x+2, y+27)
		pdf.CellFormat(colWidth-4, 4, tr(label.BarcodeID), "", 0, "C", false, 0, "")
	}

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// GenerateBarcodeCSV creates a CSV containing the barcode to ISBN mapping for the supplier.
func GenerateBarcodeCSV(labels []BarcodeLabelDetail) ([]byte, error) {
	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)
	writer.Comma = ';' // European CSV format

	// Header
	if err := writer.Write([]string{"ISBN", "Titel", "Autor", "Barcode"}); err != nil {
		return nil, err
	}

	for _, l := range labels {
		if err := writer.Write([]string{l.ISBN, l.Titel, l.Autor, l.BarcodeID}); err != nil {
			return nil, err
		}
	}
	writer.Flush()
	return buf.Bytes(), writer.Error()
}
