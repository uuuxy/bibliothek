package pdf

import (
	"bibliothek/pkg/pdfzeichen"
	"bibliothek/pkg/strichcode"
	"bytes"
	"fmt"

	"github.com/phpdave11/gofpdf"
)

// lernmittelTabellenSpalten sind die vier Spalten der Ausleihhistorie mit ihren Breiten in
// mm, beschriftet wie auf den Lernmittel-Etiketten, die schon in den Büchern kleben.
var lernmittelTabellenSpalten = []struct {
	Titel string
	Breit float64
}{
	{"Schuljahr", 15},
	{"Name des Schülers", 35},
	{"Klasse", 15},
	{"Zustand", 20},
}

// lernmittelTabellenZeilen ist die Zahl der leeren Zeilen zum Ausfüllen von Hand: Ein Buch
// durchläuft mehrere Schuljahre, bevor es ausgesondert wird.
const lernmittelTabellenZeilen = 6

// Maße des großen Etiketts. Ein A6-Feld ist ein Viertel einer A4-Seite (210×297 = 2×105 mal
// 2×148,5): Vier davon passen ohne Skalierung auf ein Blatt.
const (
	lernmittelFeldBreite = 105.0 // A6-Breite
	lernmittelFeldHoehe  = 148.5 // A6-Höhe
	lernmittelRand       = 10.0  // Innenrand im Feld
	lernmittelInhalt     = lernmittelFeldBreite - 2*lernmittelRand
	lernmittelProSeite   = 4
)

// ErzeugeLernmittelEtiketten erzeugt die großen Lernmittel-Etiketten für Lieferanten,
// die selbst etikettieren. Sie tragen die Kopfangaben des kleinen Etiketts
// (zeichneBarcodeEtikett) und dazu eine Tabelle für die Ausleihhistorie über mehrere Jahre,
// für die das kleine Format keinen Platz hat.
//
// Vier Etiketten stehen auf einer A4-Seite (2×2). Mit einem Etikett je Seite käme auf einem
// A4-Drucker ein Etikett je Blatt heraus, außer jemand stellte im Druckdialog „4 Seiten pro
// Blatt" ein; die Anordnung gehört in die Datei.
func ErzeugeLernmittelEtiketten(etiketten []BuchEtikett) ([]byte, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(lernmittelRand, lernmittelRand, lernmittelRand)
	// Ohne das setzt gofpdf mitten in die Tabelle des unteren Feldes einen Seitenumbruch,
	// sobald der Inhalt in die unteren 2 cm reicht. Die Lage jedes Feldes rechnet der
	// Erzeuger selbst; ein Umbruch von gofpdf verschöbe sie.
	pdf.SetAutoPageBreak(false, 0)
	tr := pdfzeichen.Uebersetzer(pdf.UnicodeTranslatorFromDescriptor(""))

	for i, etikett := range etiketten {
		if i%lernmittelProSeite == 0 {
			pdf.AddPage()
			zeichneSchnittlinien(pdf)
		}
		feld := i % lernmittelProSeite
		ox := float64(feld%2) * lernmittelFeldBreite
		oy := float64(feld/2) * lernmittelFeldHoehe
		zeichneLernmittelEtikett(pdf, tr, etikett, ox, oy)
	}

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// zeichneSchnittlinien zeichnet das Kreuz, an dem die vier Etiketten getrennt werden. Ohne
// die Linien wäre die Mitte des Blatts zu schätzen. Sie sind hellgrau und haarfein, damit
// sie beim Zuschneiden verschwinden und nicht als Rahmen auf dem Etikett stehen bleiben.
func zeichneSchnittlinien(pdf *gofpdf.Fpdf) {
	pdf.SetDrawColor(200, 200, 200)
	pdf.SetLineWidth(0.1)
	pdf.Line(lernmittelFeldBreite, 0, lernmittelFeldBreite, 2*lernmittelFeldHoehe)
	pdf.Line(0, lernmittelFeldHoehe, 2*lernmittelFeldBreite, lernmittelFeldHoehe)
	pdf.SetDrawColor(0, 0, 0)
	pdf.SetLineWidth(0.2)
}

// zeichneLernmittelEtikett setzt ein großes Etikett in das Feld mit der linken oberen Ecke
// (ox, oy). Alle Maße im Rumpf zählen ab diesem Ursprung, weil vier Felder auf einer Seite
// stehen.
func zeichneLernmittelEtikett(pdf *gofpdf.Fpdf, tr func(string) string, etikett BuchEtikett, ox, oy float64) {
	const breite = lernmittelInhalt
	x := ox + lernmittelRand

	y := oy + 12.0
	pdf.SetFont("Arial", "B", 11)
	pdf.SetXY(x, y)
	pdf.CellFormat(breite, 5, tr(kuerzeAufEtikett(pdf, tr, etikett.Schulname, 45, breite)), "", 0, "C", false, 0, "")

	y += 7
	pdf.SetFont("Arial", "B", 12)
	pdf.SetXY(x, y)
	pdf.CellFormat(breite, 6, tr(kuerzeAufEtikett(pdf, tr, etikett.Titel, 42, breite)), "", 0, "C", false, 0, "")

	zeile2 := zweiteZeile(etikett.AnschaffungsJahr, etikett.Signatur)
	if zeile2 != "" {
		y += 7
		pdf.SetFont("Arial", "", 9)
		pdf.SetXY(x, y)
		pdf.CellFormat(breite, 4, tr(kuerzeAufEtikett(pdf, tr, zeile2, 48, breite)), "", 0, "C", false, 0, "")
	}

	y += 9
	bild, err := strichcode.PNG(etikett.BarcodeID, false, 250, 70)
	if err == nil {
		bildLeser := bytes.NewReader(bild)
		bildName := fmt.Sprintf("lm_%s", etikett.BarcodeID)
		optionen := gofpdf.ImageOptions{ImageType: "PNG"}
		pdf.RegisterImageOptionsReader(bildName, optionen, bildLeser)
		strichBreite, strichHoehe := 60.0, 16.0
		pdf.ImageOptions(bildName, x+(breite-strichBreite)/2, y, strichBreite, strichHoehe, false, optionen, 0, "")
	}
	y += 18

	pdf.SetFont("Courier", "B", 11)
	pdf.SetXY(x, y)
	pdf.CellFormat(breite, 5, tr("Exemplar-Nr.: "+etikett.BarcodeID), "", 0, "C", false, 0, "")

	y += 7
	if vermerk := etikett.Eigentumsvermerk; vermerk != "" {
		pdf.SetFont("Arial", "", 8)
		pdf.SetXY(x, y)
		pdf.CellFormat(breite, 4, tr(kuerzeAufEtikett(pdf, tr, vermerk, 48, breite)), "", 0, "C", false, 0, "")
		y += 8
	}

	zeichneLernmittelTabelle(pdf, tr, x, y)
}

// zeichneLernmittelTabelle setzt die Tabelle der Ausleihhistorie (Kopfzeile und leere
// Zeilen) ab der übergebenen Ecke, mit umrandeten Zellen wie in mahnliste.go und
// bestellanschreiben.go. x kommt vom Aufrufer: Die zwei rechten Felder der Seite beginnen
// bei x = 115, ein fester linker Rand zöge ihre Tabellen unter das Nachbaretikett.
func zeichneLernmittelTabelle(pdf *gofpdf.Fpdf, tr func(string) string, x, y float64) {
	const zeilenhoehe = 7.0

	pdf.SetXY(x, y)
	pdf.SetFont("Arial", "B", 8)
	pdf.SetFillColor(230, 230, 230)
	for _, spalte := range lernmittelTabellenSpalten {
		pdf.CellFormat(spalte.Breit, zeilenhoehe, tr(spalte.Titel), "1", 0, "C", true, 0, "")
	}
	pdf.Ln(zeilenhoehe)

	pdf.SetFont("Arial", "", 8)
	for i := 0; i < lernmittelTabellenZeilen; i++ {
		pdf.SetX(x)
		for _, spalte := range lernmittelTabellenSpalten {
			pdf.CellFormat(spalte.Breit, zeilenhoehe, "", "1", 0, "C", false, 0, "")
		}
		pdf.Ln(zeilenhoehe)
	}
}
