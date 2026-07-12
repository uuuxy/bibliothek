package api

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"bibliothek/apierrors"
	"bibliothek/repository"
)

// mahnwesenSendenRequest is the payload for POST /api/mahnwesen/senden.
type mahnwesenSendenRequest struct {
	Klasse string `json:"klasse"`
	Email  string `json:"email"`
}

// SendMahnwesenHandler generates the class-specific PDF and e-mails it to the teacher.
// POST /api/mahnwesen/senden  { "klasse": "5b", "email": "teacher@example.com" }
func (s *Server) SendMahnwesenHandler(mahnRepo *repository.MahnwesenRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req mahnwesenSendenRequest
		if !DecodeAndValidate(w, r, &req) {
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

		klassen, err := mahnRepo.QueryUeberfaelligeNachKlasse(ctx, req.Klasse)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		pdfBytes, err := generateMahnPDF(klassen)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		totalSchueler, totalMedien := zaehleMahnStatistik(klassen)
		mailReq := baueMahnMailRequest(req, pdfBytes, totalSchueler, totalMedien)

		if os.Getenv("SMTP_HOST") == "" {
			log.Printf("MAHNWESEN: SMTP_HOST not set – skipping email dispatch for class %s", req.Klasse)
			RespondJSON(w, http.StatusOK, map[string]string{
				"status":  "pdf_only",
				"message": "SMTP nicht konfiguriert – E-Mail wurde nicht gesendet",
			})
			return
		}

		if err := SendEmail(mailReq); err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, fmt.Errorf("E-Mail-Versand fehlgeschlagen: %w", err))
			return
		}

		RespondJSON(w, http.StatusOK, map[string]string{
			"status":  "sent",
			"message": fmt.Sprintf("Mahnliste für Klasse %s an %s gesendet.", req.Klasse, req.Email),
		})
	}
}

// zaehleMahnStatistik summiert betroffene Schüler und überfällige Medien über alle Klassen.
func zaehleMahnStatistik(klassen []repository.MahnwesenKlasse) (totalSchueler, totalMedien int) {
	for _, kl := range klassen {
		totalSchueler += len(kl.Schueler)
		for _, sch := range kl.Schueler {
			totalMedien += len(sch.Medien)
		}
	}
	return totalSchueler, totalMedien
}

// baueMahnMailRequest baut die E-Mail (Text + PDF-Anhang) für eine Klassen-Mahnliste.
func baueMahnMailRequest(req mahnwesenSendenRequest, pdfBytes []byte, totalSchueler, totalMedien int) MailRequest {
	emailBody := fmt.Sprintf(
		"Sehr geehrte Damen und Herren,\n\n"+
			"anbei erhalten Sie die aktuelle Mahntliste der Schulbibliothek für die Klasse %s (Stand: %s).\n\n"+
			"Betroffene Schüler/innen: %d\n"+
			"Überfällige Medien gesamt: %d\n\n"+
			"Bitte informieren Sie die betroffenen Schüler/innen über die ausstehenden Rückgaben.\n\n"+
			"Mit freundlichen Grüßen,\nSchulbibliothek",
		req.Klasse,
		time.Now().Format(dateFormatDE),
		totalSchueler,
		totalMedien,
	)

	return MailRequest{
		To:      req.Email,
		Subject: fmt.Sprintf("Mahnliste Schulbibliothek – Klasse %s – %s", req.Klasse, time.Now().Format(dateFormatDE)),
		Body:    emailBody,
		Attachments: []MailAttachment{
			{
				Name:        fmt.Sprintf("mahnliste_%s_%s.pdf", req.Klasse, time.Now().Format(dateFormatISO)),
				ContentType: contentTypePDF,
				Data:        pdfBytes,
			},
		},
	}
}
