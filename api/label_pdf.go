package api

import (
	"bytes"
	"fmt"

	"github.com/jung-kurt/gofpdf"
)

// labelPos ist die linke obere Ecke eines Etiketts auf der Seite.
type labelPos struct{ X, Y float64 }

// zeichneQRLabel rendert ein Etikett mit QR-Code (Titel, Autor, QR, Barcode-Text).
func zeichneQRLabel(pdf *gofpdf.Fpdf, tr func(string) string, format LabelFormat, item BarcodeLabelDetail, titel, autor string, pos labelPos) {
	pdf.SetFont("Arial", "B", 8)
	pdf.SetXY(pos.X+2, pos.Y+3)
	pdf.Cell(format.LabelWidth-4, 4, tr(titel))

	pdf.SetFont("Arial", "", 7)
	pdf.SetXY(pos.X+2, pos.Y+7)
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
		qrX := pos.X + (format.LabelWidth-qrSize)/2
		qrY := pos.Y + 11.0
		if format.LabelHeight < 30 {
			qrY = pos.Y + 8.0
		}
		pdf.Image(item.BarcodeID, qrX, qrY, qrSize, qrSize, false, "", 0, "")
	}

	// Barcode text
	pdf.SetFont("Arial", "B", 8)
	textY := pos.Y + 28
	if format.LabelHeight < 30 {
		textY = pos.Y + 21
	}
	pdf.SetXY(pos.X+2, textY)
	pdf.CellFormat(format.LabelWidth-4, 4, tr(item.BarcodeID), "", 0, "C", false, 0, "")
}

// zeichneBarcodeLabel rendert ein Etikett mit 1D-Barcode (Code39/Code128).
// zeichneBarcodeLabel setzt das Buchetikett nach der Vorlage, die in der Bibliothek
// seit Jahren an den Büchern klebt:
//
//	Philipp-Reis-Schule, Friedrichsdorf   ← fett, Schulname aus den Einstellungen
//	Seydlitz - Geographie Gymnasium       ← fett, Titel
//	Ansch.J. 2016                         ← Anschaffungsjahr
//	[ Barcode ]
//	Exemplar-Nr.: 82347
//	Eigentum des Landes Hessen
//
// Vorher standen dort drei Angaben: ein fest verdrahtetes "Schulbibliothek", der Titel
// und die nackte Nummer. Der Eigentumsvermerk fehlte ganz — bei einem Buch, das dem
// Land gehört und über Jahre durch Schülerhände geht, ist er der Grund, warum es
// zurückkommt.
//
// Auf dem kleinen Format (21,2 mm) ist für sechs Zeilen kein Platz. Dort entfallen
// Anschaffungsjahr und Eigentumsvermerk — lieber weniger Angaben als übereinander
// gedruckte.
func zeichneBarcodeLabel(pdf *gofpdf.Fpdf, tr func(string) string, format LabelFormat, item BarcodeLabelDetail, titel string, kopf EtikettKopf, pos labelPos) {
	grossesEtikett := format.LabelHeight >= 30

	y := pos.Y + 2.5
	pdf.SetFont("Arial", "B", 8)
	pdf.SetXY(pos.X, y)
	pdf.CellFormat(format.LabelWidth, 3.5, tr(kuerzeAufZeichen(kopf.Schulname, 42)), "", 0, "C", false, 0, "")

	y += 3.5
	pdf.SetXY(pos.X, y)
	pdf.CellFormat(format.LabelWidth, 3.5, tr(titel), "", 0, "C", false, 0, "")

	if grossesEtikett && item.AnschaffungsJahr != "" {
		y += 3.5
		pdf.SetFont("Arial", "", 7)
		pdf.SetXY(pos.X, y)
		pdf.CellFormat(format.LabelWidth, 3, tr("Ansch.J. "+item.AnschaffungsJahr), "", 0, "C", false, 0, "")
		y += 3
	} else {
		y += 3.5
	}

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

		bcX := pos.X + (format.LabelWidth-bcWidth)/2
		pdf.ImageOptions(imgName, bcX, y+1, bcWidth, bcHeight, false, opt, 0, "")
	}
	y += bcHeight + 1.5

	// Die Nummer unter dem Barcode ist DIESELBE, die der Barcode trägt — die Beschriftung
	// benennt sie nur, damit sie ohne Scanner ablesbar ist (etwa auf der Mahnliste).
	pdf.SetFont("Courier", "B", 9)
	pdf.SetXY(pos.X, y)
	beschriftung := item.BarcodeID
	if grossesEtikett {
		beschriftung = "Exemplar-Nr.: " + item.BarcodeID
	}
	pdf.CellFormat(format.LabelWidth, 3.5, tr(beschriftung), "", 0, "C", false, 0, "")

	if grossesEtikett && kopf.Eigentumsvermerk != "" {
		y += 4.5
		pdf.SetFont("Arial", "", 7)
		pdf.SetXY(pos.X, y)
		pdf.CellFormat(format.LabelWidth, 3, tr(kuerzeAufZeichen(kopf.Eigentumsvermerk, 45)), "", 0, "C", false, 0, "")
	}
}

// GenerateLabelsPDF creates a standardized A4 PDF label sheet.
// formatId: identifies the label sheet (e.g. "zweckform_l4760")
// startPosition: 1-based index to start printing on the first page (to skip used labels)
// isQR: if true, a QR code is generated instead of a 1D Code39 barcode.
// items: the labels to print.
// kopf: schulweite Angaben (Schulname, Eigentumsvermerk) aus den Systemeinstellungen.
func GenerateLabelsPDF(formatId string, startPosition int, isQR bool, items []BarcodeLabelDetail, kopf EtikettKopf) (*gofpdf.Fpdf, error) {
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

		// Titel und Autor auf die Etikettenbreite bringen (zeichen-, nicht byteweise).
		titel := kuerzeAufZeichen(item.Titel, 40)
		autor := kuerzeAufZeichen(item.Autor, 30)

		if isQR {
			zeichneQRLabel(pdf, tr, format, item, titel, autor, labelPos{X: x, Y: y})
		} else {
			zeichneBarcodeLabel(pdf, tr, format, item, titel, kopf, labelPos{X: x, Y: y})
		}
	}

	return pdf, nil
}
