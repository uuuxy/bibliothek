package pdf

import (
	"bibliothek/pkg/pdfzeichen"
	"bibliothek/pkg/strichcode"
	"bytes"
	"fmt"

	"github.com/jung-kurt/gofpdf"
)

// labelPos ist die linke obere Ecke eines Etiketts auf der Seite.
type labelPos struct{ X, Y float64 }

// BuchEtikett ist, was auf dem Etikett eines Exemplars steht. Die Tür füllt es aus dem
// Druckauftrag, aus dem, was der Server über das Exemplar weiß, und aus den Einstellungen.
type BuchEtikett struct {
	// Schulname steht als erste Zeile auf jedem Etikett; er zeigt einem gefundenen Buch den
	// Weg zurück.
	Schulname string
	BarcodeID string
	Titel     string
	Autor     string
	// AnschaffungsJahr und Signatur stehen zusammen in einer Zeile unter dem Titel; ein
	// leerer Wert lässt seine Angabe weg.
	AnschaffungsJahr string
	Signatur         string
	// Eigentumsvermerk ist der Vermerk dieses Exemplars. Welcher gilt, hängt am Topf des
	// Exemplars und ist entschieden, bevor das Etikett hier ankommt; leer druckt keinen.
	Eigentumsvermerk string
}

// zeichneQRLabel rendert ein Etikett mit QR-Code (Titel, Autor, QR, Barcode-Text).
func zeichneQRLabel(pdf *gofpdf.Fpdf, tr func(string) string, format LabelFormat, item BuchEtikett, titel, autor string, pos labelPos) {
	pdf.SetFont("Arial", "B", 8)
	pdf.SetXY(pos.X+2, pos.Y+3)
	pdf.Cell(format.LabelWidth-4, 4, tr(titel))

	pdf.SetFont("Arial", "", 7)
	pdf.SetXY(pos.X+2, pos.Y+7)
	pdf.Cell(format.LabelWidth-4, 4, tr(autor))

	// Generate dynamic QR code PNG
	barcodeImg, err := strichcode.PNG(item.BarcodeID, true, 200, 200)
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

// zweiteZeile kombiniert Anschaffungsjahr und Signatur auf EINER Zeile
// ("Ansch.J. 2016 · LMF-Deutsch 5"), statt für die Signatur eine eigene siebte Zeile zu
// verlangen. Die physischen Formate (Zweckform L4760: 38,1 mm, Avery 3475: 37 mm) sind
// randvoll — der bestehende Inhalt füllt schon rund 35 mm, eine zusätzliche volle Zeile
// hätte das Etikett über seine Klebefläche hinaus wachsen lassen.
func zweiteZeile(jahr, signatur string) string {
	switch {
	case jahr != "" && signatur != "":
		return "Ansch.J. " + jahr + " · " + signatur
	case jahr != "":
		return "Ansch.J. " + jahr
	case signatur != "":
		return signatur
	default:
		return ""
	}
}

// zeichneBarcodeLabel rendert ein Etikett mit 1D-Barcode (Code39/Code128).
// zeichneBarcodeLabel setzt das Buchetikett nach der Vorlage, die in der Bibliothek
// seit Jahren an den Büchern klebt:
//
//	Philipp-Reis-Schule, Friedrichsdorf   ← fett, Schulname aus den Einstellungen
//	Seydlitz - Geographie Gymnasium       ← fett, Titel
//	Ansch.J. 2016 · LMF-Deutsch 5         ← Anschaffungsjahr · Signatur
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
// Anschaffungsjahr/Signatur und Eigentumsvermerk — lieber weniger Angaben als
// übereinander gedruckte.
func zeichneBarcodeLabel(pdf *gofpdf.Fpdf, tr func(string) string, format LabelFormat, item BuchEtikett, titel string, pos labelPos) {
	grossesEtikett := format.LabelHeight >= 30

	y := pos.Y + 2.5
	pdf.SetFont("Arial", "B", 8)
	pdf.SetXY(pos.X, y)
	pdf.CellFormat(format.LabelWidth, 3.5, tr(pdfzeichen.KuerzeAufZeichen(item.Schulname, 42)), "", 0, "C", false, 0, "")

	y += 3.5
	pdf.SetXY(pos.X, y)
	pdf.CellFormat(format.LabelWidth, 3.5, tr(titel), "", 0, "C", false, 0, "")

	zeile2 := zweiteZeile(item.AnschaffungsJahr, item.Signatur)
	if grossesEtikett && zeile2 != "" {
		y += 3.5
		pdf.SetFont("Arial", "", 7)
		pdf.SetXY(pos.X, y)
		pdf.CellFormat(format.LabelWidth, 3, tr(pdfzeichen.KuerzeAufZeichen(zeile2, 45)), "", 0, "C", false, 0, "")
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

	barcodeImg, err := strichcode.PNG(item.BarcodeID, false, 250, 70)
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

	if vermerk := item.Eigentumsvermerk; grossesEtikett && vermerk != "" {
		y += 4.5
		pdf.SetFont("Arial", "", 7)
		pdf.SetXY(pos.X, y)
		pdf.CellFormat(format.LabelWidth, 3, tr(pdfzeichen.KuerzeAufZeichen(vermerk, 45)), "", 0, "C", false, 0, "")
	}
}

// GenerateLabelsPDF creates a standardized A4 PDF label sheet.
// formatId: identifies the label sheet (e.g. "zweckform_l4760")
// startPosition: 1-based index to start printing on the first page (to skip used labels)
// isQR: if true, a QR code is generated instead of a 1D Code39 barcode.
// items: the labels to print.
func GenerateLabelsPDF(formatId string, startPosition int, isQR bool, items []BuchEtikett) (*gofpdf.Fpdf, error) {
	format, _ := GetLabelFormat(formatId)

	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(format.MarginLeft, format.MarginTop, format.MarginLeft)
	pdf.SetAutoPageBreak(false, 0)
	pdf.AddPage()

	tr := pdfzeichen.Uebersetzer(pdf.UnicodeTranslatorFromDescriptor(""))

	zeichneRaster(pdf, format, startPosition, len(items), func(i int, pos labelPos) {
		item := items[i]
		// Titel und Autor auf die Etikettenbreite bringen (zeichen-, nicht byteweise).
		titel := pdfzeichen.KuerzeAufZeichen(item.Titel, 40)
		autor := pdfzeichen.KuerzeAufZeichen(item.Autor, 30)

		if isQR {
			zeichneQRLabel(pdf, tr, format, item, titel, autor, pos)
		} else {
			zeichneBarcodeLabel(pdf, tr, format, item, titel, pos)
		}
	})

	return pdf, nil
}

// zeichneRaster legt `anzahl` Etiketten in das Raster des Bogens und ruft je Etikett
// `zeichne` mit seiner linken oberen Ecke auf. Startposition (angebrochener Bogen),
// Spalten-/Zeilenrechnung und Seitenumbruch stehen damit an EINER Stelle.
//
// Es gibt die Funktion, seit die Schüler-Etiketten dazugekommen sind: Sie brauchen
// dasselbe Raster mit anderem Inhalt. Eine zweite Kopie der Rechnung wäre der Anfang
// zweier Bögen, die sich irgendwann um eine halbe Zeile unterscheiden — und das merkt
// niemand am Bildschirm, sondern erst an einem verdruckten Bogen Klebeetiketten.
func zeichneRaster(pdf *gofpdf.Fpdf, format LabelFormat, startPosition, anzahl int, zeichne func(i int, pos labelPos)) {
	// 1-basierte Startposition auf einen 0-basierten Versatz bringen.
	versatz := startPosition - 1
	if versatz < 0 {
		versatz = 0
	}
	proSeite := format.Cols * format.Rows

	for i := 0; i < anzahl; i++ {
		stelle := versatz + i
		if stelle > 0 && stelle%proSeite == 0 {
			pdf.AddPage()
		}

		aufSeite := stelle % proSeite
		x := format.MarginLeft + float64(aufSeite%format.Cols)*(format.LabelWidth+format.GapX)
		y := format.MarginTop + float64(aufSeite/format.Cols)*(format.LabelHeight+format.GapY)

		zeichne(i, labelPos{X: x, Y: y})
	}
}
