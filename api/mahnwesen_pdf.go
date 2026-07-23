package api

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"bibliothek/apierrors"
	"bibliothek/pkg/httpresp"
	"bibliothek/repository"

	"github.com/jung-kurt/gofpdf"
)

// zeichneMahnMedienZeile rendert eine Tabellenzeile für ein überfälliges Medium
// (inkl. lokalem Cover, gekürztem Titel/Autor und Rot-Hervorhebung ab 14 Tagen).
func zeichneMahnMedienZeile(pdf *gofpdf.Fpdf, tr func(string) string, med repository.UeberfaelligesMedium, rowHeight float64) {
	startY := pdf.GetY()

	// Cover image (if local)
	coverPath := ""
	if strings.HasPrefix(med.CoverURL, "/uploads/") {
		localPath := strings.TrimPrefix(med.CoverURL, "/")
		if _, err := os.Stat(localPath); err == nil {
			coverPath = localPath
		}
	}
	if coverPath != "" {
		// Register image; gofpdf auto-detects type from file extension
		pdf.ImageOptions(coverPath, 18, startY+0.5, 7, rowHeight-1, false,
			gofpdf.ImageOptions{ReadDpi: true}, 0, "")
	}
	// Cover cell border (always draw border)
	pdf.SetXY(18, startY)
	pdf.CellFormat(8, rowHeight, "", "1", 0, "", false, 0, "")

	// Title cell
	titleCell := med.Titel
	if len(titleCell) > 40 {
		titleCell = titleCell[:37] + "…"
	}
	pdf.CellFormat(52, rowHeight, tr(titleCell), "1", 0, "L", false, 0, "")

	// Author
	autorCell := med.Autor
	if len(autorCell) > 20 {
		autorCell = autorCell[:18] + "…"
	}
	pdf.CellFormat(26, rowHeight, tr(autorCell), "1", 0, "L", false, 0, "")

	// Barcode-Zelle: Rahmen zeichnen, dann Barcode-Bild + darunter die Nummer einbetten —
	// damit das Buch bei der Rückgabe direkt vom Zettel gescannt werden kann.
	bcX := pdf.GetX()
	pdf.CellFormat(40, rowHeight, "", "1", 0, "", false, 0, "")
	if med.Barcode != "" {
		if pngBytes, err := GenerateBarcodePNG(med.Barcode, false, 300, 80); err == nil {
			imgName := "bc_" + med.Barcode
			opt := gofpdf.ImageOptions{ImageType: "PNG"}
			pdf.RegisterImageOptionsReader(imgName, opt, bytes.NewReader(pngBytes))
			pdf.ImageOptions(imgName, bcX+3, startY+2.5, 34, 8, false, opt, 0, "")
		}
		pdf.SetFont("Courier", "", 7)
		pdf.SetXY(bcX, startY+11.5)
		pdf.CellFormat(40, 4, tr(med.Barcode), "", 0, "C", false, 0, "")
		pdf.SetFont("Arial", "", 8)
	}

	// Zurück in die Fällig-Spalte (Barcode-Overlay hat die Position verschoben).
	pdf.SetXY(144, startY)
	pdf.CellFormat(22, rowHeight, tr(med.FaelligAm), "1", 0, "C", false, 0, "")

	// Days overdue (highlighted red if > 14)
	if med.TageUeberfaellig > 14 {
		pdf.SetTextColor(200, 30, 30)
		pdf.SetFont("Arial", "B", 8)
	}
	pdf.CellFormat(26, rowHeight, fmt.Sprintf("%d Tage", med.TageUeberfaellig), "1", 1, "C", false, 0, "")
	pdf.SetTextColor(0, 0, 0)
	pdf.SetFont("Arial", "", 8)
}

// zeichneMahnSeite rendert die komplette Mahnseite eines Schülers (Kopf,
// Infobox, Medien-Tabelle, Fußzeile).
func zeichneMahnSeite(pdf *gofpdf.Fpdf, tr func(string) string, sch repository.UeberfaelligerSchueler) {
	// ─── Page header ─────────────────────────────────────────────
	pdf.SetFont("Arial", "B", 14)
	pdf.Cell(0, 9, tr("Mahnung – Schulbibliothek"))
	pdf.Ln(7)

	pdf.SetFont("Arial", "", 9)
	pdf.SetTextColor(120, 120, 120)
	pdf.Cell(0, 5, tr(fmt.Sprintf("Erstellt am %s", time.Now().Format(dateFormatDE))))
	pdf.SetTextColor(0, 0, 0)
	pdf.Ln(10)

	// ─── Student info box ─────────────────────────────────────────
	pdf.SetFont("Arial", "B", 11)
	pdf.SetFillColor(240, 245, 255)
	pdf.SetDrawColor(180, 195, 230)
	pdf.RoundedRect(18, pdf.GetY(), 174, 22, 3, "1234", "FD")
	pdf.SetXY(24, pdf.GetY()+4)
	pdf.Cell(60, 7, tr("Schüler/in:"))
	pdf.SetFont("Arial", "", 11)
	pdf.Cell(0, 7, tr(sch.Name))
	pdf.SetXY(24, pdf.GetY()+8)
	pdf.SetFont("Arial", "B", 11)
	pdf.Cell(60, 7, tr("Klasse:"))
	pdf.SetFont("Arial", "", 11)
	pdf.Cell(0, 7, tr(sch.Klasse))
	pdf.SetXY(18, pdf.GetY()+12)

	pdf.Ln(6)
	pdf.SetFont("Arial", "I", 9)
	pdf.SetTextColor(160, 60, 60)
	pdf.Cell(0, 5, tr(fmt.Sprintf(
		"Bitte gib die folgenden %d %s umgehend in der Schulbibliothek ab.",
		len(sch.Medien),
		pluralMedium(len(sch.Medien)),
	)))
	pdf.SetTextColor(0, 0, 0)
	pdf.Ln(8)

	// ─── Table header ─────────────────────────────────────────────
	pdf.SetFont("Arial", "B", 8)
	pdf.SetFillColor(220, 225, 240)
	pdf.CellFormat(8, 8, "", "1", 0, "C", true, 0, "") // cover placeholder col
	pdf.CellFormat(52, 8, tr("Buchtitel"), "1", 0, "L", true, 0, "")
	pdf.CellFormat(26, 8, tr("Autor"), "1", 0, "L", true, 0, "")
	pdf.CellFormat(40, 8, tr("Barcode"), "1", 0, "C", true, 0, "")
	pdf.CellFormat(22, 8, tr("Fällig"), "1", 0, "C", true, 0, "")
	pdf.CellFormat(26, 8, tr("Tage überfällig"), "1", 1, "C", true, 0, "")

	// ─── Table rows ───────────────────────────────────────────────
	pdf.SetFont("Arial", "", 8)
	rowHeight := 18.0
	for _, med := range sch.Medien {
		zeichneMahnMedienZeile(pdf, tr, med, rowHeight)
	}

	// ─── Footer line ──────────────────────────────────────────────
	pdf.Ln(10)
	pdf.SetFont("Arial", "I", 8)
	pdf.SetTextColor(130, 130, 130)
	pdf.Cell(0, 5, tr("Schulbibliothek – Bei Fragen wende dich bitte an das Bibliotheksteam."))
	pdf.SetTextColor(0, 0, 0)
}

// generateMahnPDF creates an A4 PDF reminder list.
// Layout: exactly one page per student (page break after every student).
// Each page shows: student name, class, and a table of their overdue media with covers.
func generateMahnPDF(klassen []repository.MahnwesenKlasse) ([]byte, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(18, 18, 18)
	tr := pdf.UnicodeTranslatorFromDescriptor("")

	printedFirst := false
	for _, kl := range klassen {
		for _, sch := range kl.Schueler {
			// Every student gets their own page
			pdf.AddPage()
			printedFirst = true
			zeichneMahnSeite(pdf, tr, sch)
		}
	}

	if !printedFirst {
		// Produce an empty page if there are no overdue items
		pdf.AddPage()
		pdf.SetFont("Arial", "", 12)
		pdf.SetTextColor(130, 130, 130)
		pdf.Cell(0, 10, tr("Keine überfälligen Ausleihen vorhanden."))
	}

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// pluralMedium returns singular or plural German word for "medium".
func pluralMedium(n int) string {
	if n == 1 {
		return "Medium"
	}
	return "Medien"
}

// GetMahnwesenPDFHandler generates and streams the full overdue PDF.
// GET /api/mahnwesen/pdf
func (s *Server) GetMahnwesenPDFHandler(mahnRepo *repository.MahnwesenRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		isFerien, ferienName, err := mahnRepo.CheckFerienAktiv(ctx)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		if isFerien {
			apierrors.SendHTTPError(w, http.StatusForbidden, fmt.Errorf("mahnwesen ist derzeit pausiert (Ferien/Schließzeit: %s)", ferienName))
			return
		}

		klassen, err := mahnRepo.QueryUeberfaelligeNachKlasse(ctx, "")
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		pdfBytes, err := generateMahnPDF(klassen)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		w.Header().Set(headerContentType, contentTypePDF)
		w.Header().Set(headerContentDisposition,
			fmt.Sprintf("attachment; filename=mahnliste_%s.pdf", time.Now().Format(dateFormatISO)))
		httpresp.Write(w, pdfBytes)
	}
}
