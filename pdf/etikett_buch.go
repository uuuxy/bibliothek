package pdf

import (
	"bibliothek/pkg/pdfzeichen"
	"bibliothek/pkg/strichcode"
	"bytes"
	"math"

	"github.com/phpdave11/gofpdf"
)

// etikettPosition ist die linke obere Ecke eines Etiketts auf der Seite.
type etikettPosition struct{ X, Y float64 }

// rechteck ist Lage und Größe eines Felds auf dem Etikett: linke obere Ecke, Breite, Höhe.
type rechteck struct{ X, Y, Breite, Hoehe float64 }

// BuchEtikett ist, was auf dem Etikett eines Exemplars steht. Die Tür füllt es aus dem
// Druckauftrag, aus dem, was der Server über das Exemplar weiß, und aus den Einstellungen.
type BuchEtikett struct {
	// Schulname steht als erste Zeile auf jedem Etikett; er zeigt einem gefundenen Buch den
	// Weg zurück.
	Schulname string
	BarcodeID string
	Titel     string
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

// zeichneNummer setzt die Nummer des Strichcodes in die Zelle, zentriert und in der gerade
// gesetzten Schrift. Sie steht immer ganz da, weil eine gekürzte Nummer beim Ablesen zu keinem
// Buch führt: Passt sie mit der Beschriftung nicht, steht sie allein da, passt sie auch so nicht,
// wird die Schrift kleiner.
func zeichneNummer(pdf *gofpdf.Fpdf, tr func(string) string, beschriftung, nummer string, breite, hoehe float64) {
	text := beschriftung + nummer
	if beschriftung != "" && pdf.GetStringWidth(tr(text)) > breite {
		text = nummer
	}
	if gemessen := pdf.GetStringWidth(tr(text)); gemessen > breite {
		groesse, _ := pdf.GetFontSize()
		// Abgerundet auf ein Zehntel Punkt, weil gofpdf die Größe gerundet ins PDF schreibt.
		pdf.SetFontSize(math.Floor(groesse*breite/gemessen*10) / 10)
	}
	pdf.CellFormat(breite, hoehe, tr(text), "", 0, "C", false, 0, "")
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

// etikettMasse sind die senkrechten Maße eines Buchetiketts in mm: wo die erste Zeile beginnt,
// wie hoch eine Zeile ist, die Luft über und unter dem Code und die Höhe des Codes.
type etikettMasse struct {
	oben, zeile, vorCode, nachCode, strichHoehe, qrSeite float64
}

// masseFuer liefert die Maße für das Format. Auf dem kleinen Format (21,2 mm) rücken die Zeilen
// enger, und der Code wird kleiner, damit die Nummer über der Unterkante bleibt; mit den Maßen
// des großen stand sie halb auf dem Etikett darunter.
func masseFuer(format EtikettFormat) etikettMasse {
	if format.istGross() {
		return etikettMasse{oben: 2.5, zeile: 3.5, vorCode: 1, nachCode: 1.5, strichHoehe: 10, qrSeite: 12}
	}
	return etikettMasse{oben: 1.5, zeile: 3, vorCode: 0.5, nachCode: 0.5, strichHoehe: 7, qrSeite: 10}
}

// Neben dem QR-Code des kleinen Formats: Abstand des Codes zum Rand und der Nummer zum Code, in mm.
const (
	qrRandKlein    = 3.0
	qrAbstandKlein = 2.0
)

// zeichneEtikett setzt das Buchetikett nach der Vorlage, die in der Bibliothek an den Büchern
// klebt:
//
//	Philipp-Reis-Schule, Friedrichsdorf   ← fett, Schulname aus den Einstellungen
//	Seydlitz - Geographie Gymnasium       ← fett, Titel
//	Ansch.J. 2016 · LMF-Deutsch 5         ← Anschaffungsjahr · Signatur
//	[ Strichcode oder QR-Code ]
//	Exemplar-Nr.: 82347
//	Eigentum des Landes Hessen
//
// Der Eigentumsvermerk ist bei einem Buch, das dem Land gehört und über Jahre durch
// Schülerhände geht, der Grund, warum es zurückkommt; deshalb trägt das Etikett mit QR-Code
// dieselben Zeilen. Auf dem kleinen Format ist für sechs Zeilen kein Platz: Dort entfallen
// Anschaffungsjahr mit Signatur und der Eigentumsvermerk, und neben dem QR-Code steht die
// Nummer, weil unter ihm kein Platz ist.
func zeichneEtikett(pdf *gofpdf.Fpdf, tr func(string) string, format EtikettFormat, etikett BuchEtikett, alsQR bool, pos etikettPosition) {
	gross := format.istGross()
	m := masseFuer(format)

	y := pos.Y + m.oben
	pdf.SetFont("Arial", "B", 8)
	zeichneZeile(pdf, tr, etikett.Schulname, 42, rechteck{pos.X, y, format.Breite, m.zeile})
	y += m.zeile
	zeichneZeile(pdf, tr, etikett.Titel, 40, rechteck{pos.X, y, format.Breite, m.zeile})
	y += m.zeile
	if zeile2 := zweiteZeile(etikett.AnschaffungsJahr, etikett.Signatur); gross && zeile2 != "" {
		pdf.SetFont("Arial", "", 7)
		zeichneZeile(pdf, tr, zeile2, 45, rechteck{pos.X, y, format.Breite, 3})
		y += 3
	}

	// Die Nummer unter dem Code ist die, die er trägt; die Beschriftung benennt sie, damit sie
	// ohne Scanner ablesbar ist (etwa für die Mahnliste).
	pdf.SetFont("Courier", "B", 9)
	y += m.vorCode
	if alsQR && !gross {
		zeichneCode(pdf, etikett.BarcodeID, true, rechteck{pos.X + qrRandKlein, y, m.qrSeite, m.qrSeite})
		x := pos.X + qrRandKlein + m.qrSeite + qrAbstandKlein
		pdf.SetXY(x, y+(m.qrSeite-m.zeile)/2)
		zeichneNummer(pdf, tr, "", etikett.BarcodeID, pos.X+format.Breite-qrRandKlein-x, m.zeile)
		return
	}
	codeBreite, codeHoehe := 40.0, m.strichHoehe
	if format.Breite < 50 {
		codeBreite = 35.0
	}
	if alsQR {
		codeBreite, codeHoehe = m.qrSeite, m.qrSeite
	}
	zeichneCode(pdf, etikett.BarcodeID, alsQR, rechteck{pos.X + (format.Breite-codeBreite)/2, y, codeBreite, codeHoehe})
	y += codeHoehe + m.nachCode

	pdf.SetXY(pos.X, y)
	beschriftung := ""
	if gross {
		beschriftung = "Exemplar-Nr.: "
	}
	zeichneNummer(pdf, tr, beschriftung, etikett.BarcodeID, format.Breite, m.zeile)

	if vermerk := etikett.Eigentumsvermerk; gross && vermerk != "" {
		pdf.SetFont("Arial", "", 7)
		zeichneZeile(pdf, tr, vermerk, 45, rechteck{pos.X, y + m.zeile + 1, format.Breite, 3})
	}
}

// zeichneZeile setzt eine Zeile des Etiketts zentriert in das Rechteck, in der gerade gesetzten
// Schrift und gekürzt wie kuerzeAufEtikett.
func zeichneZeile(pdf *gofpdf.Fpdf, tr func(string) string, text string, zeichen int, r rechteck) {
	pdf.SetXY(r.X, r.Y)
	pdf.CellFormat(r.Breite, r.Hoehe, tr(kuerzeAufEtikett(pdf, tr, text, zeichen, r.Breite)), "", 0, "C", false, 0, "")
}

// zeichneCode setzt den Strichcode oder den QR-Code der Nummer als Bild in das Rechteck. Lässt
// sich die Nummer nicht kodieren, bleibt das Rechteck leer; die Nummer darunter steht trotzdem da.
func zeichneCode(pdf *gofpdf.Fpdf, nummer string, alsQR bool, r rechteck) {
	pixelBreite, pixelHoehe, name := 250, 70, "1d_"+nummer
	if alsQR {
		pixelBreite, pixelHoehe, name = 200, 200, "qr_"+nummer
	}
	bild, err := strichcode.PNG(nummer, alsQR, pixelBreite, pixelHoehe)
	if err != nil {
		return
	}
	optionen := gofpdf.ImageOptions{ImageType: "PNG"}
	pdf.RegisterImageOptionsReader(name, optionen, bytes.NewReader(bild))
	pdf.ImageOptions(name, r.X, r.Y, r.Breite, r.Hoehe, false, optionen, 0, "")
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
		zeichneEtikett(pdf, tr, format, etiketten[i], mitQR, pos)
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
