package pdf

import (
	"bibliothek/pkg/pdfzeichen"

	"github.com/jung-kurt/gofpdf"
)

// kuerzeAufZelle kürzt den Text, bis er in der gerade gesetzten Schrift in eine Zelle der
// Breite passt. gofpdf lässt links und rechts vom Text einen Rand und druckt Überlanges über
// die Nachbarzelle.
func kuerzeAufZelle(pdf *gofpdf.Fpdf, tr func(string) string, text string, breite float64) string {
	return pdfzeichen.KuerzeAufBreite(pdf, tr, text, breite-2*pdf.GetCellMargin())
}
