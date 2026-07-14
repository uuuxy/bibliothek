package api

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"bibliothek/apierrors"
	"bibliothek/auth"
	"bibliothek/repository"
)

// bulkOverdueResponse ist die Antwort von POST /api/mail/send-bulk-overdue.
type bulkOverdueResponse struct {
	SentCount    int    `json:"sent_count"`
	SkippedCount int    `json:"skipped_count"`
	Message      string `json:"message"`
}

// SendBulkOverdueHandler verschickt die Mahnliste JEDER überfälligen Klasse an die
// jeweilige Klassenleitung — eine E-Mail pro Klasse an genau eine Adresse.
//
// Datenschutz by design (bewusst identisch zum Einzelversand /api/mahnwesen/senden):
// Es wird NICHT an einzelne, i.d.R. minderjährige Schüler gemailt, sondern an die
// Lehrkraft, die die Schüler informiert. Jede Lehrkraft erhält ausschließlich die
// eigene Klassenliste — es gibt also keine klassenübergreifende Offenlegung von
// Empfängern oder Mahn-Status (kein TO/CC über mehrere Betroffene). Während
// Ferien-/Schließzeiten ist der Versand gesperrt, und der Massenversand wird
// auditiert (Rechenschaftspflicht, Art. 5 (2) DSGVO).
// POST /api/mail/send-bulk-overdue
func (s *Server) SendBulkOverdueHandler(mahnRepo *repository.MahnwesenRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// 1. Ferien-/Schließzeit-Sperre — identisch zum Einzelversand.
		isFerien, ferienName, err := mahnRepo.CheckFerienAktiv(ctx)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		if isFerien {
			apierrors.SendHTTPError(w, http.StatusForbidden, fmt.Errorf("mahnwesen ist derzeit pausiert (Ferien/Schließzeit: %s)", ferienName))
			return
		}

		// 2. Ohne konfigurierten Mailserver kein Massenversand.
		if os.Getenv("SMTP_HOST") == "" {
			apierrors.SendHTTPError(w, http.StatusServiceUnavailable, fmt.Errorf("SMTP nicht konfiguriert – Massenversand nicht möglich"))
			return
		}

		// 3. Alle überfälligen Klassen laden (leerer Filter = alle Klassen).
		klassen, err := mahnRepo.QueryUeberfaelligeNachKlasse(ctx, "")
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		// 4. Je Klasse eine eigene Mahnliste an die Klassenleitung senden.
		//    generateMahnPDF/SendEmail werden injiziert, damit die Skip- und
		//    Adressierungslogik ohne echten PDF-/Mailversand testbar bleibt.
		sent, skipped := versendeKlassenMahnungen(klassen, generateMahnPDF, SendEmail)

		// 5. Massenversand protokollieren.
		s.logBulkOverdueAudit(r, sent, skipped)

		RespondJSON(w, http.StatusOK, bulkOverdueResponse{
			SentCount:    sent,
			SkippedCount: skipped,
			Message:      fmt.Sprintf("%d Klassen-Mahnliste(n) an die Klassenleitungen versendet, %d übersprungen (keine E-Mail hinterlegt oder keine Fälle).", sent, skipped),
		})
	}
}

// versendeKlassenMahnungen erzeugt je Klasse das Mahn-PDF und mailt es an die
// hinterlegte Klassenleitung. Klassen ohne E-Mail oder ohne überfällige Schüler
// werden übersprungen. Ein Fehler bei einer einzelnen Klasse bricht den Lauf NICHT
// ab (Best-Effort), wird aber protokolliert und als "skipped" gezählt.
//
// generatePDF und sendMail sind injiziert (Produktion: generateMahnPDF/SendEmail),
// damit die Skip- und Adressierungslogik ohne echten PDF-/Mailversand testbar ist.
func versendeKlassenMahnungen(
	klassen []repository.MahnwesenKlasse,
	generatePDF func([]repository.MahnwesenKlasse) ([]byte, error),
	sendMail func(MailRequest) error,
) (sent, skipped int) {
	for _, kl := range klassen {
		if kl.LehrerEmail == "" || len(kl.Schueler) == 0 {
			skipped++
			continue
		}

		einzelKlasse := []repository.MahnwesenKlasse{kl}

		pdfBytes, err := generatePDF(einzelKlasse)
		if err != nil {
			log.Printf("bulk-overdue: PDF für Klasse %s fehlgeschlagen: %v", kl.Klasse, err)
			skipped++
			continue
		}

		totalSchueler, totalMedien := zaehleMahnStatistik(einzelKlasse)
		mailReq := baueMahnMailRequest(
			mahnwesenSendenRequest{Klasse: kl.Klasse, Email: kl.LehrerEmail},
			pdfBytes, totalSchueler, totalMedien,
		)

		if err := sendMail(mailReq); err != nil {
			log.Printf("bulk-overdue: Versand an Klasse %s (%s) fehlgeschlagen: %v", kl.Klasse, kl.LehrerEmail, err)
			skipped++
			continue
		}
		sent++
	}
	return sent, skipped
}

// logBulkOverdueAudit protokolliert den Massenversand (auslösender Admin, Anzahl)
// im audit_logs — analog zum Import-Audit.
func (s *Server) logBulkOverdueAudit(r *http.Request, sent, skipped int) {
	claims, ok := auth.GetClaims(r.Context())
	if !ok {
		return
	}
	details := fmt.Sprintf(`{"sent":%d,"skipped":%d}`, sent, skipped)
	logExec(s.DB.Pool.Exec(r.Context(), "INSERT INTO audit_logs (admin_id, aktion, details, ip_adresse) VALUES ($1, $2, $3::jsonb, $4)", claims.UserID, "BULK_OVERDUE_MAIL", details, getIP(r)))
}
