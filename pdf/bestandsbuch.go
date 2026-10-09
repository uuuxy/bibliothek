package pdf

import (
	"fmt"
	"time"

	"bibliothek/pkg/schulzeit"

	"github.com/jung-kurt/gofpdf"
)

// bestandsbuchKopf zeichnet den Briefkopf beider Bücher: Schule, Datum, Titel, Zeitraum.
func bestandsbuchKopf(p *gofpdf.Fpdf, tr func(string) string, titel string, von, bis time.Time, schule SchuleInfo) {
	p.SetFont("Arial", "B", 12)
	p.Cell(0, 8, tr(schule.Name))
	p.Ln(5)
	p.SetFont("Arial", "", 8)
	p.SetTextColor(100, 100, 100)
	p.Cell(0, 4, tr(schule.Absenderzeile()))
	p.SetTextColor(0, 0, 0)
	p.Ln(10)

	p.SetFont("Arial", "", 10)
	p.CellFormat(0, 6, tr(schule.OrtDatum(schulzeit.Jetzt().Format(dateFormatDE))), "", 1, "R", false, 0, "")
	p.Ln(4)

	p.SetFont("Arial", "B", 14)
	p.Cell(0, 10, tr(titel))
	p.Ln(7)
	p.SetFont("Arial", "", 10)
	p.SetTextColor(80, 80, 80)
	p.Cell(0, 6, tr(fmt.Sprintf("Zeitraum: %s bis %s", von.Format(dateFormatDE), bis.Format(dateFormatDE))))
	p.SetTextColor(0, 0, 0)
	p.Ln(12)
}
