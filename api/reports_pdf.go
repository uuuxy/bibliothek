package api

import (
	"bibliothek/apierrors"
	"bibliothek/pdf"
	"bibliothek/repository"
	"context"
	"errors"

	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/jung-kurt/gofpdf"
)

// OverdueBook represents a single overdue book in the report.
type OverdueBook struct {
	Titel         string
	BarcodeID     string
	AusgeliehenAm time.Time
	Frist         time.Time
	DaysOverdue   int
}

// OverdueStudent groups overdue books for a specific student.
type OverdueStudent struct {
	ID          string
	Vorname     string
	Nachname    string
	ElternEmail string
	Books       []OverdueBook
}

// loadMahnungTemplate lädt die Eltern-Mahnvorlage aus der Datenbank; ist keine
// konfiguriert, wird eine Standardvorlage verwendet.
func (s *Server) loadMahnungTemplate(ctx context.Context) (betreff, textBody string) {
	err := s.DB.Pool.QueryRow(ctx, "SELECT betreff, text_body FROM mail_vorlagen WHERE typ = 'MAHNUNG_ELTERN'").Scan(&betreff, &textBody)
	if err != nil {
		// Fallback template if nothing is configured
		betreff = "Mahnung: Überfällige Bücher"
		textBody = "Sehr geehrte Eltern von {{.Vorname}} {{.Nachname}},\n\nbitte geben Sie folgende Bücher umgehend in die Bibliothek zurück:\n\n{{.BuchListe}}\n\nVielen Dank."
	}
	return betreff, textBody
}

// queryOverdueStudents lädt alle überfälligen Ausleihen (ohne Abgänger) und gruppiert
// sie je Schüler in stabiler Reihenfolge (Nachname, Vorname, Titel).
func (s *Server) queryOverdueStudents(ctx context.Context) ([]*OverdueStudent, error) {
	query := `
		SELECT
			s.id, s.vorname, s.nachname,
			a.ausgeliehen_am, a.rueckgabe_frist,
			FLOOR(EXTRACT(EPOCH FROM (CURRENT_TIMESTAMP - a.rueckgabe_frist))/86400) AS days_overdue,
			t.titel, e.barcode_id
		FROM ausleihen a
		JOIN schueler s ON a.schueler_id = s.id
		JOIN buecher_exemplare e ON a.exemplar_id = e.id
		JOIN buecher_titel t ON e.titel_id = t.id
		WHERE a.rueckgabe_am IS NULL
		  AND a.rueckgabe_frist < CURRENT_TIMESTAMP
		  AND s.ist_abgaenger = false
		ORDER BY s.nachname, s.vorname, t.titel;
	`

	rows, err := s.DB.Pool.Query(ctx, query)
	if err != nil {
		return nil, errors.New("fehler beim Abrufen der Datenbank")
	}
	defer rows.Close()

	studentMap := make(map[string]*OverdueStudent)
	var studentOrder []string

	for rows.Next() {
		var id, vorname, nachname, titel, barcode string
		var ausgeliehenAm, frist time.Time
		var days float64 // EXTRACT returns numeric/float

		if err := rows.Scan(&id, &vorname, &nachname, &ausgeliehenAm, &frist, &days, &titel, &barcode); err != nil {
			log.Printf("Scan error: %v", err)
			continue
		}

		if _, exists := studentMap[id]; !exists {
			studentMap[id] = &OverdueStudent{
				ID: id, Vorname: vorname, Nachname: nachname,
			}
			studentOrder = append(studentOrder, id)
		}

		studentMap[id].Books = append(studentMap[id].Books, OverdueBook{
			Titel:         titel,
			BarcodeID:     barcode,
			AusgeliehenAm: ausgeliehenAm,
			Frist:         frist,
			DaysOverdue:   int(days),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	students := make([]*OverdueStudent, 0, len(studentOrder))
	for _, sID := range studentOrder {
		students = append(students, studentMap[sID])
	}
	return students, nil
}

// zeichneElternMahnbrief rendert eine DIN-5008-Mahnseite (für Fensterkuvert, Formblatt A)
// für einen Schüler inkl. Adressfeld, Betreff, Fließtext und Tabelle der überfälligen Bücher.
func zeichneElternMahnbrief(pdf *gofpdf.Fpdf, tr func(string) string, student *OverdueStudent, betreff, textBody, absender string) {
	pdf.AddPage()

	// --- DIN 5008 Folding Marks ---
	pdf.SetLineWidth(0.2)
	pdf.SetDrawColor(150, 150, 150)
	pdf.Line(0, 105, 4, 105)     // Falzmarke oben (Formblatt A)
	pdf.Line(0, 148.5, 6, 148.5) // Lochmarke (Mitte)
	pdf.Line(0, 210, 4, 210)     // Falzmarke unten (Formblatt A)

	pdf.SetDrawColor(0, 0, 0) // Reset to black

	// --- Address Window ---
	// Start Y: 45mm, X: 20mm (Formblatt A)
	pdf.SetFont("Arial", "U", 7)
	pdf.SetXY(20, 45)
	pdf.Cell(85, 5, tr(absender))

	pdf.SetFont("Arial", "", 11)
	pdf.SetXY(20, 52)

	// Address block
	pdf.CellFormat(85, 5, tr(fmt.Sprintf("Eltern von %s %s", student.Vorname, student.Nachname)), "", 1, "L", false, 0, "")
	pdf.SetX(20)

	addrLine1 := "Adresse unbekannt"
	addrLine2 := ""
	pdf.CellFormat(85, 5, tr(addrLine1), "", 1, "L", false, 0, "")
	pdf.SetX(20)
	pdf.CellFormat(85, 5, tr(addrLine2), "", 1, "L", false, 0, "")

	// --- Date ---
	pdf.SetFont("Arial", "", 11)
	pdf.SetXY(150, 85)
	pdf.Cell(40, 5, "Datum: "+time.Now().Format(dateFormatDE))

	// --- Subject ---
	pdf.SetFont("Arial", "B", 12)
	pdf.SetXY(20, 100)

	// ⚡ Bolt: Use a single Replacer to prevent intermediate allocations
	replacer := strings.NewReplacer(
		"{{.Vorname}}", student.Vorname,
		"{{.Nachname}}", student.Nachname,
	)
	parsedBetreff := replacer.Replace(betreff)
	pdf.Cell(0, 5, tr(parsedBetreff))

	// --- Body Text ---
	pdf.SetFont("Arial", "", 11)
	pdf.SetXY(20, 115)

	bodyReplacer := strings.NewReplacer(
		"{{.Vorname}}", student.Vorname,
		"{{.Nachname}}", student.Nachname,
		"{{.Frist}}", time.Now().Format(dateFormatDE),
	)
	parsedText := bodyReplacer.Replace(textBody)

	// Split by the book list placeholder
	parts := strings.Split(parsedText, "{{.BuchListe}}")

	// Print text before book list
	pdf.MultiCell(170, 6, tr(parts[0]), "", "L", false)
	pdf.Ln(5)

	// --- Overdue Books Table ---
	pdf.SetFont("Arial", "B", 10)
	pdf.SetX(20)
	pdf.SetFillColor(240, 240, 240)
	pdf.CellFormat(75, 7, tr("Titel"), "1", 0, "L", true, 0, "")
	pdf.CellFormat(35, 7, tr("Barcode"), "1", 0, "L", true, 0, "")
	pdf.CellFormat(30, 7, tr("Ausgeliehen"), "1", 0, "L", true, 0, "")
	pdf.CellFormat(30, 7, tr("Tage überfällig"), "1", 1, "R", true, 0, "")

	pdf.SetFont("Arial", "", 10)
	for _, b := range student.Books {
		pdf.SetX(20)

		tTitle := b.Titel
		if len(tTitle) > 38 {
			tTitle = tTitle[:35] + "..."
		}

		pdf.CellFormat(75, 6, tr(tTitle), "1", 0, "L", false, 0, "")
		pdf.CellFormat(35, 6, tr(b.BarcodeID), "1", 0, "L", false, 0, "")
		pdf.CellFormat(30, 6, b.AusgeliehenAm.Format(dateFormatDE), "1", 0, "L", false, 0, "")
		pdf.CellFormat(30, 6, fmt.Sprintf("%d", b.DaysOverdue), "1", 1, "R", false, 0, "")
	}

	// Print text after book list (if any)
	if len(parts) > 1 {
		pdf.Ln(5)
		pdf.SetX(20)
		pdf.SetFont("Arial", "", 11)
		pdf.MultiCell(170, 6, tr(strings.TrimSpace(parts[1])), "", "L", false)
	}
}

// GetOverdueReportsPDFHandler generates a PDF containing overdue notices for all students,
// formatted according to DIN 5008 standards for window envelopes.
func (s *Server) GetOverdueReportsPDFHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// 1. Load Mail Template
		betreff, textBody := s.loadMahnungTemplate(ctx)

		// 2. Query Overdue Loans grouped by student
		students, err := s.queryOverdueStudents(ctx)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		if len(students) == 0 {
			apierrors.SendHTTPError(w, http.StatusNotFound, errors.New("keine überfälligen Ausleihen gefunden"))
			return
		}

		settingsRepo := repository.NewSystemSettingsRepository(s.DB.Pool)
		settings, _ := settingsRepo.GetSettings(ctx) //nolint:errcheck
		schule := pdf.SchuleInfo{
			Name:    settings.SchuleName,
			Strasse: settings.SchuleStrasse,
			PLZ:     settings.SchulePLZ,
			Ort:     settings.SchuleOrt,
		}
		absender := schule.Absenderzeile()

		// 3. Generate PDF Document
		doc := gofpdf.New("P", "mm", "A4", "")
		tr := doc.UnicodeTranslatorFromDescriptor("") // To correctly render German umlauts in standard fonts

		for _, student := range students {
			zeichneElternMahnbrief(doc, tr, student, betreff, textBody, absender)
		}

		filename := fmt.Sprintf("mahnlauf_%s.pdf", time.Now().Format(dateFormatISO))
		w.Header().Set(headerContentType, contentTypePDF)
		w.Header().Set(headerContentDisposition, fmt.Sprintf("attachment; filename=\"%s\"", filename))

		if err := doc.Output(w); err != nil {
			log.Printf("Error writing PDF output: %v", err)
		}
	}
}
