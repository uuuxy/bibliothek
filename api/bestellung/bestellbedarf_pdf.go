package bestellung

import (
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"bibliothek/api/tuer"
	"bibliothek/apierrors"
	"bibliothek/pkg/pdfzeichen"
	"bibliothek/pkg/schulzeit"
	"bibliothek/repository"

	"github.com/phpdave11/gofpdf"
)

// BestellbedarfPDF exportiert den Bestellbedarf als PDF — dieselbe Auswahl und
// Reihenfolge wie die Ansicht (Vorgabe LMF, knappste zuerst, siehe Bestellbedarf).
func (t *Tueren) BestellbedarfPDF() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		bedarf, _, err := t.bestellbedarfMitEinstellungen(r.Context(), schulzeit.Jetzt())
		if err != nil {
			apierrors.AntworteFehler(w, http.StatusInternalServerError, err)
			return
		}

		pdf := baueBestelllistePDF(bedarf)

		tuer.SetzeDateiKopf(w, tuer.ContentTypePDF, tuer.Speichern, "bestellliste.pdf")
		if err := pdf.Output(w); err != nil {
			slog.Error("Bestellliste: PDF stream output failed", "fehler", err)
		}
	}
}

// baueBestelllistePDF setzt die Tabelle. Getrennt von der Tür, damit das Layout ohne
// HTTP-Kontext lesbar (und testbar) bleibt. Die Menge ist das Fehlen der Zeile, dieselbe Zahl,
// die der Warenkorb übernimmt.
func baueBestelllistePDF(bedarf []repository.Bestellbedarf) *gofpdf.Fpdf {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()
	pdf.SetMargins(15, 15, 15)
	tr := pdfzeichen.Uebersetzer(pdf.UnicodeTranslatorFromDescriptor(""))

	pdf.SetFont("Arial", "B", 16)
	pdf.Cell(0, 10, tr("Schulbibliothek - Bestellliste"))
	pdf.Ln(6)
	pdf.SetFont("Arial", "I", 9)
	pdf.SetTextColor(100, 100, 100)
	pdf.Cell(0, 5, tr(fmt.Sprintf("Generiert am %s", schulzeit.Jetzt().Format("02.01.2006 (15:04)"))))
	pdf.SetTextColor(0, 0, 0)
	pdf.Ln(12)

	schreibeBestelllisteKopf(pdf, tr)

	pdf.SetFont("Arial", "", 8)
	for _, b := range bedarf {
		pdf.CellFormat(62, 6, tr(b.Titel), "1", 0, "L", false, 0, "")
		pdf.CellFormat(35, 6, tr(b.Autor), "1", 0, "L", false, 0, "")
		pdf.CellFormat(33, 6, tr(b.ISBN), "1", 0, "L", false, 0, "")
		pdf.CellFormat(16, 6, strconv.Itoa(b.Bedarf), "1", 0, "C", false, 0, "")
		pdf.CellFormat(16, 6, strconv.Itoa(b.GesamtBestand), "1", 0, "C", false, 0, "")
		pdf.CellFormat(18, 6, strconv.Itoa(b.Fehlen), "1", 1, "C", false, 0, "")
	}
	return pdf
}

// schreibeBestelllisteKopf setzt die Kopfzeile der Tabelle.
func schreibeBestelllisteKopf(pdf *gofpdf.Fpdf, tr func(string) string) {
	pdf.SetFont("Arial", "B", 9)
	pdf.SetFillColor(230, 230, 230)
	pdf.CellFormat(62, 7, tr("Buchtitel"), "1", 0, "L", true, 0, "")
	pdf.CellFormat(35, 7, tr("Autor"), "1", 0, "L", true, 0, "")
	pdf.CellFormat(33, 7, tr("ISBN"), "1", 0, "L", true, 0, "")
	pdf.CellFormat(16, 7, tr("Bedarf"), "1", 0, "C", true, 0, "")
	pdf.CellFormat(16, 7, tr("Bestand"), "1", 0, "C", true, 0, "")
	pdf.CellFormat(18, 7, tr("Menge"), "1", 1, "C", true, 0, "")
}
