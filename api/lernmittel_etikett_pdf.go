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

// GenerateLernmittelEtikettenPDF erzeugt das große Lernmittel-Etikett (A6, eine Seite
// pro Exemplar) für Lieferanten, die selbst etikettieren und dafür fertige Etiketten von
// uns brauchen (z. B. Naacher). Enthält dieselben Kopfangaben wie das kleine
// Etikett (zeichneBarcodeLabel) — Schulname, Titel, Ansch.J./Signatur, Barcode,
// Exemplar-Nr., Eigentumsvermerk —, zusätzlich eine Tabelle für die mehrjährige
// Ausleihhistorie, für die auf dem kleinen Format kein Platz ist.
func GenerateLernmittelEtikettenPDF(items []BarcodeLabelDetail, kopf EtikettKopf) ([]byte, error) {
	pdf := gofpdf.New("P", "mm", "A6", "")
	pdf.SetMargins(10, 10, 10)
	tr := pdf.UnicodeTranslatorFromDescriptor("")

	for _, item := range items {
		pdf.AddPage()
		zeichneLernmittelEtikett(pdf, tr, item, kopf)
	}

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// zeichneLernmittelEtikett rendert EIN großes Etikett auf die aktuelle Seite.
func zeichneLernmittelEtikett(pdf *gofpdf.Fpdf, tr func(string) string, item BarcodeLabelDetail, kopf EtikettKopf) {
	const breite = 85.0 // A6 (105mm) minus 2×10mm Rand

	y := 12.0
	pdf.SetFont("Arial", "B", 11)
	pdf.SetXY(10, y)
	pdf.CellFormat(breite, 5, tr(kuerzeAufZeichen(kopf.Schulname, 45)), "", 0, "C", false, 0, "")

	y += 7
	pdf.SetFont("Arial", "B", 12)
	pdf.SetXY(10, y)
	pdf.CellFormat(breite, 6, tr(kuerzeAufZeichen(item.Titel, 42)), "", 0, "C", false, 0, "")

	zeile2 := zweiteZeile(item.AnschaffungsJahr, item.Signatur)
	if zeile2 != "" {
		y += 7
		pdf.SetFont("Arial", "", 9)
		pdf.SetXY(10, y)
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
		pdf.ImageOptions(imgName, 10+(breite-bcWidth)/2, y, bcWidth, bcHeight, false, opt, 0, "")
	}
	y += 18

	pdf.SetFont("Courier", "B", 11)
	pdf.SetXY(10, y)
	pdf.CellFormat(breite, 5, tr("Exemplar-Nr.: "+item.BarcodeID), "", 0, "C", false, 0, "")

	y += 7
	if kopf.Eigentumsvermerk != "" {
		pdf.SetFont("Arial", "", 8)
		pdf.SetXY(10, y)
		pdf.CellFormat(breite, 4, tr(kuerzeAufZeichen(kopf.Eigentumsvermerk, 48)), "", 0, "C", false, 0, "")
		y += 8
	}

	zeichneLernmittelTabelle(pdf, tr, y)
}

// zeichneLernmittelTabelle rendert die Ausleihhistorie-Tabelle (Kopfzeile + leere
// Datenzeilen) ab der übergebenen Höhe — dieselbe Border-Zellen-Technik wie in
// api/mahnwesen_pdf.go und api/order_pdf.go (GenerateOrderSummaryPDF).
func zeichneLernmittelTabelle(pdf *gofpdf.Fpdf, tr func(string) string, y float64) {
	const zeilenhoehe = 7.0

	pdf.SetXY(10, y)
	pdf.SetFont("Arial", "B", 8)
	pdf.SetFillColor(230, 230, 230)
	for _, spalte := range lernmittelTabellenSpalten {
		pdf.CellFormat(spalte.Breit, zeilenhoehe, tr(spalte.Titel), "1", 0, "C", true, 0, "")
	}
	pdf.Ln(zeilenhoehe)

	pdf.SetFont("Arial", "", 8)
	for i := 0; i < lernmittelTabellenZeilen; i++ {
		pdf.SetX(10)
		for _, spalte := range lernmittelTabellenSpalten {
			pdf.CellFormat(spalte.Breit, zeilenhoehe, "", "1", 0, "C", false, 0, "")
		}
		pdf.Ln(zeilenhoehe)
	}
}
