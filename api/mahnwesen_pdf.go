package api

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"bibliothek/apierrors"
	"bibliothek/repository"

	"github.com/jung-kurt/gofpdf"
)

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
			if printedFirst {
				pdf.AddPage()
			} else {
				pdf.AddPage()
				printedFirst = true
			}

			// ─── Page header ─────────────────────────────────────────────
			pdf.SetFont("Arial", "B", 14)
			pdf.Cell(0, 9, tr("Mahnung – Schulbibliothek"))
			pdf.Ln(7)

			pdf.SetFont("Arial", "", 9)
			pdf.SetTextColor(120, 120, 120)
			pdf.Cell(0, 5, tr(fmt.Sprintf("Erstellt am %s", time.Now().Format("02.01.2006"))))
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
			pdf.CellFormat(70, 8, tr("Buchtitel"), "1", 0, "L", true, 0, "")
			pdf.CellFormat(40, 8, tr("Autor"), "1", 0, "L", true, 0, "")
			pdf.CellFormat(30, 8, tr("Fällig am"), "1", 0, "C", true, 0, "")
			pdf.CellFormat(26, 8, tr("Tage überfällig"), "1", 1, "C", true, 0, "")

			// ─── Table rows ───────────────────────────────────────────────
			pdf.SetFont("Arial", "", 8)
			rowHeight := 18.0
			for _, med := range sch.Medien {
				startY := pdf.GetY()
				startX := pdf.GetX()
				_ = startX

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

				// Title cell (MultiCell for long titles)
				pdf.SetXY(26, startY)
				titleCell := med.Titel
				if len(titleCell) > 55 {
					titleCell = titleCell[:52] + "…"
				}
				pdf.CellFormat(70, rowHeight, tr(titleCell), "1", 0, "L", false, 0, "")

				// Author
				autorCell := med.Autor
				if len(autorCell) > 32 {
					autorCell = autorCell[:29] + "…"
				}
				pdf.CellFormat(40, rowHeight, tr(autorCell), "1", 0, "L", false, 0, "")

				// Due date
				pdf.CellFormat(30, rowHeight, tr(med.FaelligAm), "1", 0, "C", false, 0, "")

				// Days overdue (highlighted red if > 14)
				if med.TageUeberfaellig > 14 {
					pdf.SetTextColor(200, 30, 30)
					pdf.SetFont("Arial", "B", 8)
				}
				pdf.CellFormat(26, rowHeight, fmt.Sprintf("%d Tage", med.TageUeberfaellig), "1", 1, "C", false, 0, "")
				pdf.SetTextColor(0, 0, 0)
				pdf.SetFont("Arial", "", 8)
			}

			// ─── Footer line ──────────────────────────────────────────────
			pdf.Ln(10)
			pdf.SetFont("Arial", "I", 8)
			pdf.SetTextColor(130, 130, 130)
			pdf.Cell(0, 5, tr("Schulbibliothek – Bei Fragen wende dich bitte an das Bibliotheksteam."))
			pdf.SetTextColor(0, 0, 0)
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
			apierrors.SendHTTPError(w, http.StatusForbidden, fmt.Errorf("Mahnwesen ist derzeit pausiert (Ferien/Schließzeit: %s)", ferienName))
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

		w.Header().Set("Content-Type", "application/pdf")
		w.Header().Set("Content-Disposition",
			fmt.Sprintf("attachment; filename=mahnliste_%s.pdf", time.Now().Format("2006-01-02")))
		_, _ = w.Write(pdfBytes)
	}
}
