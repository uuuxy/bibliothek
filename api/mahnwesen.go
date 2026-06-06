package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"bibliothek/apierrors"

	"github.com/jung-kurt/gofpdf"
)

// UeberfaelligesMedium holds data for one overdue book copy belonging to a student.
type UeberfaelligesMedium struct {
	Titel            string `json:"titel"`
	Autor            string `json:"autor"`
	ISBN             string `json:"isbn"`
	CoverURL         string `json:"cover_url,omitempty"`
	FaelligAm        string `json:"faellig_am"`
	TageUeberfaellig int    `json:"tage_ueberfaellig"`
}

// UeberfaelligerSchueler groups overdue books by student.
type UeberfaelligerSchueler struct {
	SchuelerID string                 `json:"schueler_id"`
	Name       string                 `json:"name"`
	Klasse     string                 `json:"klasse"`
	Medien     []UeberfaelligesMedium `json:"medien"`
}

// MahnwesenKlasse groups students by class for the overview response.
type MahnwesenKlasse struct {
	Klasse      string                   `json:"klasse"`
	LehrerEmail string                   `json:"lehrer_email"` // autofill from mapping; may be empty
	Schueler    []UeberfaelligerSchueler `json:"schueler"`
}

// mahnwesenSendenRequest is the payload for POST /api/mahnwesen/senden.
type mahnwesenSendenRequest struct {
	Klasse string `json:"klasse"`
	Email  string `json:"email"`
}

// queryUeberfaelligeNachKlasse returns overdue loans grouped by class → student.
func (s *Server) queryUeberfaelligeNachKlasse(ctx context.Context, klasseFilter string) ([]MahnwesenKlasse, error) {
	q := `
		SELECT s.id, s.vorname || ' ' || s.nachname, s.klasse,
		       t.titel, coalesce(t.autor,''), coalesce(t.isbn,''), coalesce(t.cover_url,''),
		       a.rueckgabe_frist,
		       GREATEST(0, EXTRACT(DAY FROM (CURRENT_TIMESTAMP - a.rueckgabe_frist))::int) AS tage_ueberfaellig
		FROM ausleihen a
		JOIN buecher_exemplare e ON a.exemplar_id = e.id
		JOIN buecher_titel t    ON e.titel_id = t.id
		JOIN schueler s         ON a.schueler_id = s.id
		WHERE a.rueckgabe_am IS NULL
		  AND a.rueckgabe_frist < CURRENT_TIMESTAMP
	`
	args := []any{}
	if klasseFilter != "" {
		q += " AND s.klasse = $1"
		args = append(args, klasseFilter)
	}
	q += " ORDER BY s.klasse, s.nachname, s.vorname, a.rueckgabe_frist"

	rows, err := s.DB.Pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Build nested structure: klasse → schueler → medien
	klassenMap := map[string]*MahnwesenKlasse{}
	schuelerMap := map[string]*UeberfaelligerSchueler{}
	klassen := make([]MahnwesenKlasse, 0) // preserve insertion order

	for rows.Next() {
		var schuelerID, name, klasse string
		var titel, autor, isbn, coverURL string
		var frist time.Time
		var tage int
		if err := rows.Scan(&schuelerID, &name, &klasse,
			&titel, &autor, &isbn, &coverURL,
			&frist, &tage); err != nil {
			continue
		}

		if _, ok := klassenMap[klasse]; !ok {
			klassen = append(klassen, MahnwesenKlasse{Klasse: klasse})
			klassenMap[klasse] = &klassen[len(klassen)-1]
		}

		schuelerKey := klasse + "|" + schuelerID
		if _, ok := schuelerMap[schuelerKey]; !ok {
			sch := UeberfaelligerSchueler{
				SchuelerID: schuelerID,
				Name:       name,
				Klasse:     klasse,
			}
			k := klassenMap[klasse]
			k.Schueler = append(k.Schueler, sch)
			schuelerMap[schuelerKey] = &k.Schueler[len(k.Schueler)-1]
		}

		schuelerMap[schuelerKey].Medien = append(schuelerMap[schuelerKey].Medien, UeberfaelligesMedium{
			Titel:            titel,
			Autor:            autor,
			ISBN:             isbn,
			CoverURL:         coverURL,
			FaelligAm:        frist.Format("02.01.2006"),
			TageUeberfaellig: tage,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Enrich with teacher email from mapping
	if len(klassen) > 0 {
		mRows, err := s.DB.Pool.Query(ctx, `SELECT klasse, lehrer_email FROM klassen_lehrer_mapping`)
		if err == nil {
			defer mRows.Close()
			emailMap := map[string]string{}
			for mRows.Next() {
				var k, e string
				if err := mRows.Scan(&k, &e); err == nil {
					emailMap[k] = e
				}
			}
			for i := range klassen {
				klassen[i].LehrerEmail = emailMap[klassen[i].Klasse]
			}
		}
	}

	return klassen, nil
}

// GetMahnwesenHandler returns overdue loans grouped by class and student.
// GET /api/mahnwesen
func (s *Server) GetMahnwesenHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
		defer cancel()

		klassen, err := s.queryUeberfaelligeNachKlasse(ctx, "")
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"klassen": klassen})
	}
}

// generateMahnPDF creates an A4 PDF reminder list.
// Layout: exactly one page per student (page break after every student).
// Each page shows: student name, class, and a table of their overdue media with covers.
func generateMahnPDF(klassen []MahnwesenKlasse) ([]byte, error) {
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
func (s *Server) GetMahnwesenPDFHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
		defer cancel()

		klassen, err := s.queryUeberfaelligeNachKlasse(ctx, "")
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

// SendMahnwesenHandler generates the class-specific PDF and e-mails it to the teacher.
// POST /api/mahnwesen/senden  { "klasse": "5b", "email": "teacher@example.com" }
func (s *Server) SendMahnwesenHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req mahnwesenSendenRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			apierrors.SendHTTPError(w, http.StatusBadRequest, err)
			return
		}
		if req.Klasse == "" {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("klasse ist erforderlich"))
			return
		}
		if req.Email == "" {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("email ist erforderlich"))
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
		defer cancel()

		klassen, err := s.queryUeberfaelligeNachKlasse(ctx, req.Klasse)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		pdfBytes, err := generateMahnPDF(klassen)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		// Count total overdue items for email body
		totalMedien := 0
		totalSchueler := 0
		for _, kl := range klassen {
			totalSchueler += len(kl.Schueler)
			for _, sch := range kl.Schueler {
				totalMedien += len(sch.Medien)
			}
		}

		emailBody := fmt.Sprintf(
			"Sehr geehrte Damen und Herren,\n\n"+
				"anbei erhalten Sie die aktuelle Mahntliste der Schulbibliothek für die Klasse %s (Stand: %s).\n\n"+
				"Betroffene Schüler/innen: %d\n"+
				"Überfällige Medien gesamt: %d\n\n"+
				"Bitte informieren Sie die betroffenen Schüler/innen über die ausstehenden Rückgaben.\n\n"+
				"Mit freundlichen Grüßen,\nSchulbibliothek",
			req.Klasse,
			time.Now().Format("02.01.2006"),
			totalSchueler,
			totalMedien,
		)

		mailReq := MailRequest{
			To:      req.Email,
			Subject: fmt.Sprintf("Mahnliste Schulbibliothek – Klasse %s – %s", req.Klasse, time.Now().Format("02.01.2006")),
			Body:    emailBody,
			Attachments: []MailAttachment{
				{
					Name:        fmt.Sprintf("mahnliste_%s_%s.pdf", req.Klasse, time.Now().Format("2006-01-02")),
					ContentType: "application/pdf",
					Data:        pdfBytes,
				},
			},
		}

		if os.Getenv("SMTP_HOST") == "" {
			log.Printf("MAHNWESEN: SMTP_HOST not set – skipping email dispatch for class %s", req.Klasse)
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]string{
				"status":  "pdf_only",
				"message": "SMTP nicht konfiguriert – E-Mail wurde nicht gesendet",
			})
			return
		}

		if err := SendEmail(mailReq); err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, fmt.Errorf("E-Mail-Versand fehlgeschlagen: %w", err))
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"status":  "sent",
			"message": fmt.Sprintf("Mahnliste für Klasse %s an %s gesendet.", req.Klasse, req.Email),
		})
	}
}

// queryUeberfaelligeNachJahrgang returns overdue loans grouped by class → student based on grade level.
func (s *Server) queryUeberfaelligeNachJahrgang(ctx context.Context, klasseFilter string) ([]MahnwesenKlasse, error) {
	q := `
		SELECT s.id, s.vorname || ' ' || s.nachname, s.klasse,
		       t.titel, coalesce(t.autor,''), coalesce(t.isbn,''), coalesce(t.cover_url,''),
		       a.ausgeliehen_am,
		       t.jahrgang_bis,
		       NULLIF(regexp_replace(s.klasse, '\D', '', 'g'), '')::int AS schueler_jahrgang,
			   s.ist_abgaenger
		FROM ausleihen a
		JOIN buecher_exemplare e ON a.exemplar_id = e.id
		JOIN buecher_titel t    ON e.titel_id = t.id
		JOIN schueler s         ON a.schueler_id = s.id
		WHERE a.rueckgabe_am IS NULL
		  AND (
		      (NULLIF(regexp_replace(s.klasse, '\D', '', 'g'), '')::int > t.jahrgang_bis)
		      OR s.ist_abgaenger = true
		  )
	`
	args := []any{}
	if klasseFilter != "" {
		q += " AND s.klasse = $1"
		args = append(args, klasseFilter)
	}
	q += " ORDER BY s.klasse, s.nachname, s.vorname, t.titel"

	rows, err := s.DB.Pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	klassenMap := map[string]*MahnwesenKlasse{}
	schuelerMap := map[string]*UeberfaelligerSchueler{}
	klassen := make([]MahnwesenKlasse, 0)

	for rows.Next() {
		var schuelerID, name, klasse string
		var titel, autor, isbn, coverURL string
		var ausgeliehenAm time.Time
		var jahrgangBis int
		var schuelerJahrgang *int
		var istAbgaenger bool

		if err := rows.Scan(&schuelerID, &name, &klasse,
			&titel, &autor, &isbn, &coverURL,
			&ausgeliehenAm, &jahrgangBis, &schuelerJahrgang, &istAbgaenger); err != nil {
			continue
		}

		if _, ok := klassenMap[klasse]; !ok {
			klassen = append(klassen, MahnwesenKlasse{Klasse: klasse})
			klassenMap[klasse] = &klassen[len(klassen)-1]
		}

		schuelerKey := klasse + "|" + schuelerID
		if _, ok := schuelerMap[schuelerKey]; !ok {
			sch := UeberfaelligerSchueler{
				SchuelerID: schuelerID,
				Name:       name,
				Klasse:     klasse,
			}
			k := klassenMap[klasse]
			k.Schueler = append(k.Schueler, sch)
			schuelerMap[schuelerKey] = &k.Schueler[len(k.Schueler)-1]
		}

		ueberschreitung := 0
		if schuelerJahrgang != nil {
			ueberschreitung = *schuelerJahrgang - jahrgangBis
		}

		schuelerMap[schuelerKey].Medien = append(schuelerMap[schuelerKey].Medien, UeberfaelligesMedium{
			Titel:            titel,
			Autor:            autor,
			ISBN:             isbn,
			CoverURL:         coverURL,
			FaelligAm:        fmt.Sprintf("bis Kl. %d", jahrgangBis),
			TageUeberfaellig: ueberschreitung, // Missbrauchen wir für die Jahre der Überschreitung
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return klassen, nil
}

// GetMahnwesenJahrgangHandler returns overdue loans based on grade level logic.
// GET /api/mahnwesen/ueberfaellig_jahrgang
func (s *Server) GetMahnwesenJahrgangHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
		defer cancel()

		klassen, err := s.queryUeberfaelligeNachJahrgang(ctx, "")
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"klassen": klassen})
	}
}
