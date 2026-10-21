package pdf

import (
	"bibliothek/pkg/pdfzeichen"
	"bibliothek/pkg/strichcode"
	"bytes"
	"fmt"

	"github.com/phpdave11/gofpdf"
)

// etikettPosition ist die linke obere Ecke eines Etiketts auf der Seite.
type etikettPosition struct{ X, Y float64 }

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

// zeichneQREtikett setzt ein Etikett mit QR-Code: Titel, Autor, Code und darunter die Nummer.
func zeichneQREtikett(pdf *gofpdf.Fpdf, tr func(string) string, format EtikettFormat, etikett BuchEtikett, pos etikettPosition) {
	breite := format.Breite - 4
	pdf.SetFont("Arial", "B", 8)
	pdf.SetXY(pos.X+2, pos.Y+3)
	pdf.Cell(breite, 4, tr(kuerzeAufEtikett(pdf, tr, etikett.Titel, 40, breite)))

	pdf.SetFont("Arial", "", 7)
	pdf.SetXY(pos.X+2, pos.Y+7)
	pdf.Cell(breite, 4, tr(kuerzeAufEtikett(pdf, tr, etikett.Autor, 30, breite)))

	bild, err := strichcode.PNG(etikett.BarcodeID, true, 200, 200)
	if err == nil {
		bildLeser := bytes.NewReader(bild)
		pdf.RegisterImageOptionsReader(etikett.BarcodeID, gofpdf.ImageOptions{ImageType: "PNG"}, bildLeser)
		qrGroesse := 16.0
		if !format.istGross() {
			qrGroesse = 12.0 // niedrige Etiketten wie standard_52 fassen den großen Code nicht
		}
		qrX := pos.X + (format.Breite-qrGroesse)/2
		qrY := pos.Y + 11.0
		if !format.istGross() {
			qrY = pos.Y + 8.0
		}
		pdf.Image(etikett.BarcodeID, qrX, qrY, qrGroesse, qrGroesse, false, "", 0, "")
	}

	pdf.SetFont("Arial", "B", 8)
	textY := pos.Y + 28
	if !format.istGross() {
		textY = pos.Y + 21
	}
	pdf.SetXY(pos.X+2, textY)
	pdf.CellFormat(format.Breite-4, 4, tr(etikett.BarcodeID), "", 0, "C", false, 0, "")
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

// zeichneBarcodeEtikett setzt das Buchetikett mit Strichcode nach der Vorlage, die in der
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
func zeichneBarcodeEtikett(pdf *gofpdf.Fpdf, tr func(string) string, format EtikettFormat, etikett BuchEtikett, pos etikettPosition) {
	grossesEtikett := format.istGross()

	y := pos.Y + 2.5
	pdf.SetFont("Arial", "B", 8)
	pdf.SetXY(pos.X, y)
	pdf.CellFormat(format.Breite, 3.5, tr(kuerzeAufEtikett(pdf, tr, etikett.Schulname, 42, format.Breite)), "", 0, "C", false, 0, "")

	y += 3.5
	pdf.SetXY(pos.X, y)
	pdf.CellFormat(format.Breite, 3.5, tr(kuerzeAufEtikett(pdf, tr, etikett.Titel, 40, format.Breite)), "", 0, "C", false, 0, "")

	zeile2 := zweiteZeile(etikett.AnschaffungsJahr, etikett.Signatur)
	if grossesEtikett && zeile2 != "" {
		y += 3.5
		pdf.SetFont("Arial", "", 7)
		pdf.SetXY(pos.X, y)
		pdf.CellFormat(format.Breite, 3, tr(kuerzeAufEtikett(pdf, tr, zeile2, 45, format.Breite)), "", 0, "C", false, 0, "")
		y += 3
	} else {
		y += 3.5
	}

	strichBreite := 40.0
	strichHoehe := 10.0
	if format.Breite < 50 {
		strichBreite = 35.0
		strichHoehe = 8.0
	}

	bild, err := strichcode.PNG(etikett.BarcodeID, false, 250, 70)
	if err == nil {
		bildLeser := bytes.NewReader(bild)
		bildName := fmt.Sprintf("1d_%s", etikett.BarcodeID)
		optionen := gofpdf.ImageOptions{ImageType: "PNG"}
		pdf.RegisterImageOptionsReader(bildName, optionen, bildLeser)

		strichX := pos.X + (format.Breite-strichBreite)/2
		pdf.ImageOptions(bildName, strichX, y+1, strichBreite, strichHoehe, false, optionen, 0, "")
	}
	y += strichHoehe + 1.5

	// Die Nummer unter dem Strichcode ist die, die er trägt; die Beschriftung benennt sie,
	// damit sie ohne Scanner ablesbar ist (etwa für die Mahnliste).
	pdf.SetFont("Courier", "B", 9)
	pdf.SetXY(pos.X, y)
	beschriftung := etikett.BarcodeID
	if grossesEtikett {
		beschriftung = "Exemplar-Nr.: " + etikett.BarcodeID
	}
	pdf.CellFormat(format.Breite, 3.5, tr(beschriftung), "", 0, "C", false, 0, "")

	if vermerk := etikett.Eigentumsvermerk; grossesEtikett && vermerk != "" {
		y += 4.5
		pdf.SetFont("Arial", "", 7)
		pdf.SetXY(pos.X, y)
		pdf.CellFormat(format.Breite, 3, tr(kuerzeAufEtikett(pdf, tr, vermerk, 45, format.Breite)), "", 0, "C", false, 0, "")
	}
}

// ErzeugeBuchEtiketten erzeugt A4-Bögen mit Etiketten für Bücher. formatID nennt den Bogen (etwa
// "zweckform_l4760"), startPosition zählt ab 1 und überspringt die schon abgezogenen Etiketten
// eines angebrochenen Bogens, mitQR druckt einen QR-Code statt des Strichcodes.
func ErzeugeBuchEtiketten(formatID string, startPosition int, mitQR bool, etiketten []BuchEtikett) (*gofpdf.Fpdf, error) {
	format, _ := FindeEtikettFormat(formatID)

	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(format.RandLinks, format.RandOben, format.RandLinks)
	pdf.SetAutoPageBreak(false, 0)
	pdf.AddPage()

	tr := pdfzeichen.Uebersetzer(pdf.UnicodeTranslatorFromDescriptor(""))

	zeichneRaster(pdf, format, startPosition, len(etiketten), func(i int, pos etikettPosition) {
		if mitQR {
			zeichneQREtikett(pdf, tr, format, etiketten[i], pos)
		} else {
			zeichneBarcodeEtikett(pdf, tr, format, etiketten[i], pos)
		}
	})

	return pdf, nil
}

// zeichneRaster legt anzahl Etiketten in das Raster des Bogens und ruft je Etikett zeichne mit
// seiner linken oberen Ecke auf. Startposition, Spalten, Zeilen und Seitenumbruch stehen damit an
// einer Stelle für Etiketten von Büchern und von Schülern; zwei Rechnungen liefen irgendwann um
// eine halbe Zeile auseinander, und das zeigte erst ein verdruckter Bogen.
func zeichneRaster(pdf *gofpdf.Fpdf, format EtikettFormat, startPosition, anzahl int, zeichne func(i int, pos etikettPosition)) {
	// Die Startposition zählt ab 1, der Versatz ab 0.
	versatz := startPosition - 1
	if versatz < 0 {
		versatz = 0
	}
	proSeite := format.Spalten * format.Zeilen

	for i := 0; i < anzahl; i++ {
		stelle := versatz + i
		if stelle > 0 && stelle%proSeite == 0 {
			pdf.AddPage()
		}

		aufSeite := stelle % proSeite
		x := format.RandLinks + float64(aufSeite%format.Spalten)*(format.Breite+format.AbstandX)
		y := format.RandOben + float64(aufSeite/format.Spalten)*(format.Hoehe+format.AbstandY)

		zeichne(i, etikettPosition{X: x, Y: y})
	}
}
