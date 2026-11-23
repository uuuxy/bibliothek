package pdf

import (
	"math"

	"bibliothek/internal/pkg/pdfzeichen"

	"github.com/phpdave11/gofpdf"
)

// SignaturEtikettFormat ist das Signaturetikett für den Buchrücken im Druck-Center: die Signatur
// groß, Regaladresse und Kürzel untereinander. Gedruckt wird auf dem Bogen der kleinen Barcodes
// (Standard 52, 48,3 × 21,2 mm), den die Schule für ihre Etiketten hat.
const SignaturEtikettFormat = "signatur_52"

// signaturBogen ist der Bogen, auf dem die Signaturetiketten stehen.
const signaturBogen = "standard_52"

const (
	signaturSchriftGross = 18.0 // pt, solange die längste Zeile in die Breite passt
	signaturSchriftKlein = 8.0  // pt; kleiner wird nicht gesetzt, darunter wird gekürzt
	signaturRand         = 2.0  // mm links und rechts im Etikett
	signaturZeilenfaktor = 1.2  // Zeilenhöhe im Verhältnis zur Schriftgröße
	mmJePunkt            = 25.4 / 72
)

// ErzeugeSignaturEtiketten erzeugt Bögen mit einem Signaturetikett je Exemplar; aufschriften
// nennt je Etikett seine Zeilen (repository.SignaturZeilen). startPosition zählt ab 1 und
// überspringt wie bei den Buch-Etiketten die abgezogenen Felder.
func ErzeugeSignaturEtiketten(startPosition int, aufschriften [][]string) *gofpdf.Fpdf {
	format, _ := FindeEtikettFormat(signaturBogen)

	pdf := pdfzeichen.NewDokument()
	pdf.SetMargins(format.RandLinks, format.RandOben, format.RandLinks)
	pdf.SetAutoPageBreak(false, 0)
	pdf.AddPage()

	tr := pdfzeichen.Ersetze
	zeichneRaster(pdf, format, startPosition, len(aufschriften), func(i int, pos etikettPosition) {
		zeichneSignaturEtikett(pdf, tr, format, aufschriften[i], pos)
	})
	return pdf
}

// zeichneSignaturEtikett setzt die Zeilen der Signatur zentriert und fett in das Etikett, alle in
// einer Größe: so groß, wie die längste Zeile in die Breite passt, höchstens signaturSchriftGross.
func zeichneSignaturEtikett(pdf *gofpdf.Fpdf, tr func(string) string, format EtikettFormat, zeilen []string, pos etikettPosition) {
	breite := format.Breite - 2*signaturRand

	pdf.SetFont(pdfzeichen.Schrift, "B", signaturSchriftGross)
	groesse := signaturSchriftGross
	for _, zeile := range zeilen {
		if gemessen := pdf.GetStringWidth(tr(zeile)); gemessen > breite {
			// Abgerundet auf ein Zehntel Punkt, weil gofpdf die Größe gerundet ins PDF schreibt.
			groesse = math.Min(groesse, math.Floor(signaturSchriftGross*breite/gemessen*10)/10)
		}
	}
	pdf.SetFontSize(math.Max(groesse, signaturSchriftKlein))

	hoehe := math.Max(groesse, signaturSchriftKlein) * mmJePunkt * signaturZeilenfaktor
	y := pos.Y + (format.Hoehe-hoehe*float64(len(zeilen)))/2
	for i, zeile := range zeilen {
		zeichneZeile(pdf, tr, zeile, len([]rune(zeile)), rechteck{pos.X + signaturRand, y + float64(i)*hoehe, breite, hoehe})
	}
}
