package api

import (
	"fmt"
	"log"
	"net/http"
	"strconv"

	"bibliothek/apierrors"
	"bibliothek/pkg/schulzeit"
	"bibliothek/repository"

	"github.com/jung-kurt/gofpdf"
)

// ExportReordersPDFHandler exportiert den Bestellbedarf als PDF — dieselbe Auswahl und
// Reihenfolge wie die Ansicht (Default LMF, knappste zuerst, siehe GetReordersHandler).
func (s *Server) ExportReordersPDFHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reorders, err := s.reordersMitSettings(r.Context(), r)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		settings, err := repository.NewSystemSettingsRepository(s.DB.Pool).GetSettings(r.Context())
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		pdf := baueBestelllistePDF(reorders, settings.BestellbedarfSchwelle)

		w.Header().Set(headerContentType, contentTypePDF)
		w.Header().Set("Content-Disposition", "attachment; filename=bestellliste.pdf")
		if err := pdf.Output(w); err != nil {
			log.Printf("Bestellliste: PDF stream output failed: %v", err)
		}
	}
}

// baueBestelllistePDF setzt die Tabelle. Getrennt vom Handler, damit das Layout ohne
// HTTP-Kontext lesbar (und testbar) bleibt.
//
// schwelle ist der konfigurierte Soll-Bestand — dieselbe Zahl, die bestimmt, WELCHE
// Titel überhaupt auf der Liste stehen. Bis zum 30.07.2026 rechnete die Spalte
// „Nachbestellmenge" stattdessen mit dem Feld meldebestand aus dem Titel, das seit der
// Umstellung auf die Schwelle niemand mehr pflegt: Es steht auf der Vorgabe 5. Bei
// einer Schwelle von 30 und 10 vorhandenen Exemplaren stand auf der Bestellliste für
// den Händler „−5". Die Auswahl kam aus der einen Zahl, die Menge aus der anderen.
func baueBestelllistePDF(reorders []ReorderTitle, schwelle int) *gofpdf.Fpdf {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()
	pdf.SetMargins(15, 15, 15)
	tr := pdf.UnicodeTranslatorFromDescriptor("")

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
	for _, b := range reorders {
		pdf.CellFormat(62, 6, tr(b.Titel), "1", 0, "L", false, 0, "")
		pdf.CellFormat(35, 6, tr(b.Autor), "1", 0, "L", false, 0, "")
		pdf.CellFormat(33, 6, tr(b.ISBN), "1", 0, "L", false, 0, "")
		pdf.CellFormat(12, 6, strconv.Itoa(schwelle), "1", 0, "C", false, 0, "")
		pdf.CellFormat(12, 6, strconv.Itoa(b.VerfuegbarBestand), "1", 0, "C", false, 0, "")
		pdf.CellFormat(12, 6, strconv.Itoa(b.GesamtBestand), "1", 0, "C", false, 0, "")
		// Nachbestellmenge = fehlende EIGENE Exemplare (Soll − Gesamtbestand), nicht
		// − Verfügbar: verliehene Exemplare kommen zurück und müssen nicht ersetzt
		// werden. Sonst überbestellte man um die Zahl der gerade ausgeliehenen Bücher.
		pdf.CellFormat(14, 6, strconv.Itoa(nachbestellmenge(schwelle, b.GesamtBestand)), "1", 1, "C", false, 0, "")
	}
	return pdf
}

// nachbestellmenge liefert, wie viele Exemplare bis zum Soll-Bestand fehlen — nie
// weniger als null. Eine negative Menge auf einem Dokument, das an einen Händler geht,
// ist keine Zahl, sondern eine Blamage: Sie entsteht, sobald ein Titel mehr Exemplare
// hat als die Schwelle verlangt.
func nachbestellmenge(schwelle, gesamtBestand int) int {
	if fehlend := schwelle - gesamtBestand; fehlend > 0 {
		return fehlend
	}
	return 0
}

// schreibeBestelllisteKopf setzt die Kopfzeile der Tabelle.
func schreibeBestelllisteKopf(pdf *gofpdf.Fpdf, tr func(string) string) {
	pdf.SetFont("Arial", "B", 9)
	pdf.SetFillColor(230, 230, 230)
	pdf.CellFormat(62, 7, tr("Buchtitel"), "1", 0, "L", true, 0, "")
	pdf.CellFormat(35, 7, tr("Autor"), "1", 0, "L", true, 0, "")
	pdf.CellFormat(33, 7, tr("ISBN"), "1", 0, "L", true, 0, "")
	pdf.CellFormat(12, 7, tr("Soll"), "1", 0, "C", true, 0, "")
	pdf.CellFormat(12, 7, tr("Verf."), "1", 0, "C", true, 0, "")
	// Gesamt neben Verfügbar: Ein verliehener Klassensatz ist kein Bestellgrund.
	pdf.CellFormat(12, 7, tr("Ges."), "1", 0, "C", true, 0, "")
	pdf.CellFormat(14, 7, tr("Nachb."), "1", 1, "C", true, 0, "")
}
