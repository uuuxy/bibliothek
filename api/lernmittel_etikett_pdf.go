package api

import (
	"bytes"
	"fmt"

	"github.com/jung-kurt/gofpdf"
)

// lernmittelTabellenSpalten sind die vier Spalten der Ausleihhistorie — exakt die
// Beschriftung, die auf dem physischen Lernmittel-Etikett bereits verwendet wird
// (Schuljahr / Name des Schülers / Klasse / Zustand), mit Breiten in mm.
var lernmittelTabellenSpalten = []struct {
	Titel string
	Breit float64
}{
	{"Schuljahr", 15},
	{"Name des Schülers", 35},
	{"Klasse", 15},
	{"Zustand", 20},
}

// lernmittelTabellenZeilen ist die Anzahl leerer Datenzeilen zum handschriftlichen
// Ausfüllen — ein Buch durchläuft typischerweise mehrere Schuljahre, bevor es
// ausgesondert wird.
const lernmittelTabellenZeilen = 6

// Maße des großen Etiketts. Ein A6-Feld ist exakt ein Viertel einer A4-Seite
// (210×297 = 2×105 mal 2×148,5) — deshalb passen vier davon ohne jede Skalierung auf
// ein Blatt, und der Aufdruck bleibt millimetergenau so groß wie bisher.
const (
	lernmittelFeldBreite = 105.0 // A6-Breite
	lernmittelFeldHoehe  = 148.5 // A6-Höhe
	lernmittelRand       = 10.0  // Innenrand im Feld
	lernmittelInhalt     = lernmittelFeldBreite - 2*lernmittelRand
	lernmittelProSeite   = 4
)

// GenerateLernmittelEtikettenPDF erzeugt die großen Lernmittel-Etiketten für Lieferanten,
// die selbst etikettieren und dafür fertige Etiketten von uns brauchen (z. B. Naacher).
// Enthält dieselben Kopfangaben wie das kleine Etikett (zeichneBarcodeLabel) —
// Schulname, Titel, Ansch.J./Signatur, Barcode, Exemplar-Nr., Eigentumsvermerk —,
// zusätzlich eine Tabelle für die mehrjährige Ausleihhistorie, für die auf dem kleinen
// Format kein Platz ist.
//
// VIER Etiketten je A4-Seite (2×2), seit 06.08.2026. Vorher war jede Seite ein eigenes
// A6-Blatt: ein Exemplar = eine Seite. Auf einem gewöhnlichen A4-Drucker kam damit ein
// Etikett pro Blatt heraus, außer man stellte im Druckdialog von Hand „4 Seiten pro
// Blatt" ein — was niemand tut, der einen Stapel Etiketten braucht (telefonische
// Rückmeldung Naacher). Die Anordnung gehört in die Datei, nicht in den Druckdialog.
func GenerateLernmittelEtikettenPDF(items []BarcodeLabelDetail, kopf EtikettKopf) ([]byte, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(lernmittelRand, lernmittelRand, lernmittelRand)
	// Ohne das setzt gofpdf mitten in die Tabelle des unteren Feldes einen Seitenumbruch,
	// sobald der Inhalt in die untere 2-cm-Zone reicht. Die Position jedes Feldes rechnen
	// wir selbst aus — automatische Umbrüche würden sie verschieben.
	pdf.SetAutoPageBreak(false, 0)
	tr := pdf.UnicodeTranslatorFromDescriptor("")

	for i, item := range items {
		if i%lernmittelProSeite == 0 {
			pdf.AddPage()
			zeichneSchnittlinien(pdf)
		}
		feld := i % lernmittelProSeite
		ox := float64(feld%2) * lernmittelFeldBreite
		oy := float64(feld/2) * lernmittelFeldHoehe
		zeichneLernmittelEtikett(pdf, tr, item, kopf, ox, oy)
	}

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// zeichneSchnittlinien zeichnet das Kreuz, an dem die vier Etiketten getrennt werden.
//
// Ohne die Linien muss man die Mitte des Blattes schätzen — bei einem Etikett, das
// später in ein Buch geklebt wird, sieht man jeden schiefen Schnitt. Hellgrau und
// haarfein, damit die Linie beim Zuschneiden verschwindet und nicht als Rahmen auf dem
// fertigen Etikett stehen bleibt.
func zeichneSchnittlinien(pdf *gofpdf.Fpdf) {
	pdf.SetDrawColor(200, 200, 200)
	pdf.SetLineWidth(0.1)
	pdf.Line(lernmittelFeldBreite, 0, lernmittelFeldBreite, 2*lernmittelFeldHoehe)
	pdf.Line(0, lernmittelFeldHoehe, 2*lernmittelFeldBreite, lernmittelFeldHoehe)
	pdf.SetDrawColor(0, 0, 0)
	pdf.SetLineWidth(0.2)
}

// zeichneLernmittelEtikett rendert EIN großes Etikett in das Feld mit der linken oberen
// Ecke (ox, oy). Alle Maße im Rumpf sind relativ zu diesem Ursprung — genau das ist der
// Unterschied zur Fassung davor, die absolut auf einer eigenen A6-Seite zeichnete.
func zeichneLernmittelEtikett(pdf *gofpdf.Fpdf, tr func(string) string, item BarcodeLabelDetail, kopf EtikettKopf, ox, oy float64) {
	const breite = lernmittelInhalt
	x := ox + lernmittelRand

	y := oy + 12.0
	pdf.SetFont("Arial", "B", 11)
	pdf.SetXY(x, y)
	pdf.CellFormat(breite, 5, tr(kuerzeAufZeichen(kopf.Schulname, 45)), "", 0, "C", false, 0, "")

	y += 7
	pdf.SetFont("Arial", "B", 12)
	pdf.SetXY(x, y)
	pdf.CellFormat(breite, 6, tr(kuerzeAufZeichen(item.Titel, 42)), "", 0, "C", false, 0, "")

	zeile2 := zweiteZeile(item.AnschaffungsJahr, item.Signatur)
	if zeile2 != "" {
		y += 7
		pdf.SetFont("Arial", "", 9)
		pdf.SetXY(x, y)
		pdf.CellFormat(breite, 4, tr(kuerzeAufZeichen(zeile2, 48)), "", 0, "C", false, 0, "")
	}

	y += 9
	barcodeImg, err := GenerateBarcodePNG(item.BarcodeID, false, 250, 70)
	if err == nil {
		imgReader := bytes.NewReader(barcodeImg)
		imgName := fmt.Sprintf("lm_%s", item.BarcodeID)
		opt := gofpdf.ImageOptions{ImageType: "PNG"}
		pdf.RegisterImageOptionsReader(imgName, opt, imgReader)
		bcWidth, bcHeight := 60.0, 16.0
		pdf.ImageOptions(imgName, x+(breite-bcWidth)/2, y, bcWidth, bcHeight, false, opt, 0, "")
	}
	y += 18

	pdf.SetFont("Courier", "B", 11)
	pdf.SetXY(x, y)
	pdf.CellFormat(breite, 5, tr("Exemplar-Nr.: "+item.BarcodeID), "", 0, "C", false, 0, "")

	y += 7
	if kopf.Eigentumsvermerk != "" {
		pdf.SetFont("Arial", "", 8)
		pdf.SetXY(x, y)
		pdf.CellFormat(breite, 4, tr(kuerzeAufZeichen(kopf.Eigentumsvermerk, 48)), "", 0, "C", false, 0, "")
		y += 8
	}

	zeichneLernmittelTabelle(pdf, tr, x, y)
}

// zeichneLernmittelTabelle rendert die Ausleihhistorie-Tabelle (Kopfzeile + leere
// Datenzeilen) ab der übergebenen Ecke — dieselbe Border-Zellen-Technik wie in
// api/mahnwesen_pdf.go und api/order_pdf.go (GenerateOrderSummaryPDF).
//
// x wird durchgereicht und nicht mehr fest auf 10 gesetzt: Seit vier Etiketten auf einer
// Seite stehen, liegen die beiden rechten Felder bei x = 115. Ein festes SetX(10) hätte
// deren Tabellen an den linken Blattrand gezogen — unter das Nachbaretikett.
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
