package api

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"bibliothek/apierrors"

	"github.com/jackc/pgx/v5"
	"github.com/jung-kurt/gofpdf"
)

// GenerateDamagePDFHandler generates a formal PDF notification letter ("Elternbrief")
// for a student responsible for library book damage, marking the record in the DB.
func (s *Server) GenerateDamagePDFHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if id == "" {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("missing damage case ID parameter"))
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
		defer cancel()

		var beschreibung string
		var betrag float64
		var erstelltAm time.Time
		var sVorname, sNachname, sKlasse string
		var tTitel, eBarcode string

		query := `
			SELECT 
				sf.beschreibung, sf.betrag, sf.erstellt_am,
				s.vorname, s.nachname, s.klasse,
				t.titel, e.barcode_id
			FROM schadensfaelle sf
			JOIN schueler s ON sf.schueler_id = s.id
			JOIN buecher_exemplare e ON sf.exemplar_id = e.id
			JOIN buecher_titel t ON e.titel_id = t.id
			WHERE sf.id = $1
		`

		err := s.DB.Pool.QueryRow(ctx, query, id).Scan(
			&beschreibung, &betrag, &erstelltAm,
			&sVorname, &sNachname, &sKlasse,
			&tTitel, &eBarcode,
		)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				apierrors.SendHTTPError(w, http.StatusNotFound, err)
				return
			}
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		// Create new A4 PDF page in portrait mode
		pdf := gofpdf.New("P", "mm", "A4", "")
		pdf.AddPage()
		pdf.SetMargins(20, 20, 20)

		// UTF-8 to ISO-8859-1 conversion to support German umlauts (ä, ö, ü, ß) in standard PDF fonts
		tr := pdf.UnicodeTranslatorFromDescriptor("")

		// Letter Header
		pdf.SetFont("Arial", "B", 14)
		pdf.Cell(0, 10, tr("Städtisches Gymnasium Musterstadt - Schulbibliothek"))
		pdf.Ln(6)
		pdf.SetFont("Arial", "", 8)
		pdf.SetTextColor(100, 100, 100)
		pdf.Cell(0, 4, tr("Schulbibliothek · Lindenallee 4 · 12345 Musterstadt"))
		pdf.SetTextColor(0, 0, 0)
		pdf.Ln(15)

		// Date line (Right-aligned)
		pdf.SetFont("Arial", "", 10)
		pdf.CellFormat(0, 6, fmt.Sprintf("Musterstadt, den %s", time.Now().Format("02.01.2006")), "", 0, "R", false, 0, "")
		pdf.Ln(12)

		// Recipient Address Block
		pdf.SetFont("Arial", "B", 9)
		pdf.Cell(0, 4, tr("An die Erziehungsberechtigten von:"))
		pdf.Ln(5)
		pdf.SetFont("Arial", "", 11)
		pdf.Cell(0, 6, tr(fmt.Sprintf("%s %s", sVorname, sNachname)))
		pdf.Ln(5)
		pdf.Cell(0, 6, tr(fmt.Sprintf("Klasse %s", sKlasse)))
		pdf.Ln(25)

		// Letter Subject
		pdf.SetFont("Arial", "B", 12)
		pdf.Cell(0, 8, tr("Ersatzforderung für ein beschädigtes oder verlorenes Bibliotheksbuch"))
		pdf.Ln(12)

		// Introduction Body
		pdf.SetFont("Arial", "", 10)
		introText := fmt.Sprintf("Sehr geehrte Erziehungsberechtigte,\n\n"+
			"bei der Rückgabe bzw. Überprüfung des von Ihrem Kind ausgeliehenen Schulbuches "+
			"\"%s\" (Barcode: %s) wurde am %s folgende Beschädigung festgestellt:\n\n",
			tTitel, eBarcode, erstelltAm.Format("02.01.2006"))
		pdf.MultiCell(0, 5, tr(introText), "", "L", false)

		// Damage Description Box
		pdf.SetFillColor(245, 245, 245)
		pdf.SetFont("Arial", "I", 10)
		pdf.CellFormat(0, 10, tr(fmt.Sprintf("   Schadensfall: %s", beschreibung)), "1", 1, "L", true, 0, "")
		pdf.Ln(6)

		// Resolution guidelines and payment request
		pdf.SetFont("Arial", "", 10)
		dueTime := time.Now().AddDate(0, 0, 14).Format("02.01.2006")
		instructions := fmt.Sprintf("Gemäß der Schulbibliotheksordnung bitten wir Sie, für den entstandenen Schaden "+
			"einen Ersatzbetrag von %.2f EUR bis spätestens zum %s im Schulsekretariat bar zu entrichten "+
			"oder auf das Schulkonto zu überweisen.\n\n"+
			"Sollten Sie Fragen zum Schadensfall haben, können Sie sich gerne zu den Öffnungszeiten "+
			"an das Bibliotheksteam wenden.\n\n"+
			"Vielen Dank für Ihr Verständnis und Ihre Kooperation.",
			betrag, dueTime)
		pdf.MultiCell(0, 5, tr(instructions), "", "L", false)
		pdf.Ln(15)

		// Signatures
		pdf.Cell(0, 6, tr("Mit freundlichen Grüßen,"))
		pdf.Ln(15)
		pdf.SetFont("Arial", "B", 10)
		pdf.Cell(0, 6, tr("Die Bibliotheksleitung"))

		// Update database flag indicating that the letter was generated
		updateQuery := `
			UPDATE schadensfaelle
			SET elternbrief_generiert = true,
			    elternbrief_generiert_am = CURRENT_TIMESTAMP,
			    aktualisiert_am = CURRENT_TIMESTAMP
			WHERE id = $1
		`
		_, dbErr := s.DB.Pool.Exec(ctx, updateQuery, id)
		if dbErr != nil {
			log.Printf("PDF Generator: Database status update failed for case %s: %v", id, dbErr)
		}

		// Stream the generated PDF
		w.Header().Set("Content-Type", "application/pdf")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=elternbrief_%s.pdf", sNachname))

		if err := pdf.Output(w); err != nil {
			log.Printf("PDF Generator: Output error: %v", err)
			return
		}
	}
}
