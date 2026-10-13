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

// kuerzeAufEtikett kürzt den Text einer Zeile des Etiketts für die gerade gesetzte Schrift:
// auf höchstens zeichen Zeichen und auf die Breite der Zelle. Die Zeichenzahl begrenzt die
// Zeile auf breiten Bögen, die Breite hält sie auf schmalen im Etikett; gofpdf druckt
// Überlanges über den Rand der Zelle, am Buchetikett also auf das Nachbaretikett.
func kuerzeAufEtikett(pdf *gofpdf.Fpdf, tr func(string) string, text string, zeichen int, breite float64) string {
	return pdfzeichen.KuerzeAufBreite(pdf, tr, pdfzeichen.KuerzeAufZeichen(text, zeichen), breite)
}

// zeichneQRLabel setzt ein Etikett mit QR-Code: Titel, Autor, Code und darunter die Nummer.
func zeichneQRLabel(pdf *gofpdf.Fpdf, tr func(string) string, format LabelFormat, item BuchEtikett, pos labelPos) {
	breite := format.LabelWidth - 4
	pdf.SetFont("Arial", "B", 8)
	pdf.SetXY(pos.X+2, pos.Y+3)
	pdf.Cell(breite, 4, tr(kuerzeAufEtikett(pdf, tr, item.Titel, 40, breite)))

	pdf.SetFont("Arial", "", 7)
	pdf.SetXY(pos.X+2, pos.Y+7)
	pdf.Cell(breite, 4, tr(kuerzeAufEtikett(pdf, tr, item.Autor, 30, breite)))

	barcodeImg, err := strichcode.PNG(item.BarcodeID, true, 200, 200)
	if err == nil {
		imgReader := bytes.NewReader(barcodeImg)
		pdf.RegisterImageOptionsReader(item.BarcodeID, gofpdf.ImageOptions{ImageType: "PNG"}, imgReader)
		qrSize := 16.0
		if format.LabelHeight < 30 {
			qrSize = 12.0 // niedrige Etiketten wie standard_52 fassen den großen Code nicht
		}
		qrX := pos.X + (format.LabelWidth-qrSize)/2
		qrY := pos.Y + 11.0
		if format.LabelHeight < 30 {
			qrY = pos.Y + 8.0
		}
		pdf.Image(item.BarcodeID, qrX, qrY, qrSize, qrSize, false, "", 0, "")
	}

	pdf.SetFont("Arial", "B", 8)
	textY := pos.Y + 28
	if format.LabelHeight < 30 {
		textY = pos.Y + 21
	}
	pdf.SetXY(pos.X+2, textY)
	pdf.CellFormat(format.LabelWidth-4, 4, tr(item.BarcodeID), "", 0, "C", false, 0, "")
}

// zweiteZeile setzt Anschaffungsjahr und Signatur in eine Zeile ("Ansch.J. 2016 ·
// LMF-Deutsch 5"). Für eine eigene Zeile je Angabe sind die Bögen zu niedrig: Der übrige
// Inhalt füllt rund 35 mm, die Etiketten sind 38,1 mm (Zweckform L4760) und 37 mm (Avery
// 3475) hoch.
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

// zeichneBarcodeLabel setzt das Buchetikett mit Strichcode nach der Vorlage, die in der
// Bibliothek an den Büchern klebt:
//
//	Philipp-Reis-Schule, Friedrichsdorf   ← fett, Schulname aus den Einstellungen
//	Seydlitz - Geographie Gymnasium       ← fett, Titel
//	Ansch.J. 2016 · LMF-Deutsch 5         ← Anschaffungsjahr · Signatur
//	[ Barcode ]
//	Exemplar-Nr.: 82347
//	Eigentum des Landes Hessen
//
// Der Eigentumsvermerk ist bei einem Buch, das dem Land gehört und über Jahre durch
// Schülerhände geht, der Grund, warum es zurückkommt. Auf dem kleinen Format (21,2 mm) ist
// für sechs Zeilen kein Platz: Dort entfallen Anschaffungsjahr mit Signatur und der
// Eigentumsvermerk, sonst stünden Zeilen übereinander.
func zeichneBarcodeLabel(pdf *gofpdf.Fpdf, tr func(string) string, format LabelFormat, item BuchEtikett, pos labelPos) {
	grossesEtikett := format.LabelHeight >= 30

	y := pos.Y + 2.5
	pdf.SetFont("Arial", "B", 8)
	pdf.SetXY(pos.X, y)
	pdf.CellFormat(format.LabelWidth, 3.5, tr(kuerzeAufEtikett(pdf, tr, item.Schulname, 42, format.LabelWidth)), "", 0, "C", false, 0, "")

	y += 3.5
	pdf.SetXY(pos.X, y)
	pdf.CellFormat(format.LabelWidth, 3.5, tr(kuerzeAufEtikett(pdf, tr, item.Titel, 40, format.LabelWidth)), "", 0, "C", false, 0, "")

	zeile2 := zweiteZeile(item.AnschaffungsJahr, item.Signatur)
	if grossesEtikett && zeile2 != "" {
		y += 3.5
		pdf.SetFont("Arial", "", 7)
		pdf.SetXY(pos.X, y)
		pdf.CellFormat(format.LabelWidth, 3, tr(kuerzeAufEtikett(pdf, tr, zeile2, 45, format.LabelWidth)), "", 0, "C", false, 0, "")
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

	// Die Nummer unter dem Strichcode ist die, die er trägt; die Beschriftung benennt sie,
	// damit sie ohne Scanner ablesbar ist (etwa für die Mahnliste).
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
		pdf.CellFormat(format.LabelWidth, 3, tr(kuerzeAufEtikett(pdf, tr, vermerk, 45, format.LabelWidth)), "", 0, "C", false, 0, "")
	}
}

// GenerateLabelsPDF erzeugt A4-Bögen mit Buchetiketten. formatId nennt den Bogen (etwa
// "zweckform_l4760"), startPosition zählt ab 1 und überspringt die schon abgezogenen
// Etiketten eines angebrochenen Bogens, isQR druckt einen QR-Code statt des Strichcodes.
func GenerateLabelsPDF(formatId string, startPosition int, isQR bool, items []BuchEtikett) (*gofpdf.Fpdf, error) {
	format, _ := GetLabelFormat(formatId)

	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(format.MarginLeft, format.MarginTop, format.MarginLeft)
	pdf.SetAutoPageBreak(false, 0)
	pdf.AddPage()

	tr := pdfzeichen.Uebersetzer(pdf.UnicodeTranslatorFromDescriptor(""))

	zeichneRaster(pdf, format, startPosition, len(items), func(i int, pos labelPos) {
		if isQR {
			zeichneQRLabel(pdf, tr, format, items[i], pos)
		} else {
			zeichneBarcodeLabel(pdf, tr, format, items[i], pos)
		}
	})

	return pdf, nil
}

// zeichneRaster legt anzahl Etiketten in das Raster des Bogens und ruft je Etikett zeichne
// mit seiner linken oberen Ecke auf. Startposition, Spalten- und Zeilenrechnung und
// Seitenumbruch stehen damit an einer Stelle, für Buch- und Schüler-Etiketten: Zwei
// Rechnungen wichen irgendwann um eine halbe Zeile voneinander ab, und das zeigt erst ein
// verdruckter Bogen.
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
