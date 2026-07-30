package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/mail"
	"strings"

	"bibliothek/apierrors"
	"bibliothek/auth"
	"bibliothek/mailservice"
	"bibliothek/repository"
)

type MailSettingsResponse struct {
	SMTPHost    string `json:"smtp_host"`
	SMTPPort    string `json:"smtp_port"`
	SMTPUser    string `json:"smtp_user"`
	SenderEmail string `json:"sender_email"`
	HasPassword bool   `json:"has_password"`
}

type MailSettingsRequest struct {
	SMTPHost     string `json:"smtp_host"`
	SMTPPort     string `json:"smtp_port"`
	SMTPUser     string `json:"smtp_user"`
	SMTPPassword string `json:"smtp_password"` // Optional, if empty will retain old password
	SenderEmail  string `json:"sender_email"`
}

// GetMailSettingsHandler gibt die Mail-Konfiguration zurück
func (s *Server) GetMailSettingsHandler(mailRepo *repository.MailSettingsRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		config, err := mailRepo.GetConfig(ctx)
		if err != nil {
			// Falls noch keine Konfiguration existiert, geben wir leere Werte zurück (bzw. Defaults)
			config = &repository.MailSettings{
				SMTPHost:    "localhost",
				SMTPPort:    "1025",
				SMTPUser:    "",
				SenderEmail: "noreply@bibliothek-schule.de",
			}
		}

		resp := MailSettingsResponse{
			SMTPHost:    config.SMTPHost,
			SMTPPort:    config.SMTPPort,
			SMTPUser:    config.SMTPUser,
			SenderEmail: config.SenderEmail,
			HasPassword: len(config.SMTPPasswordEncrypted) > 0,
		}

		RespondJSON(w, http.StatusOK, resp)
	}
}

// UpdateMailSettingsHandler speichert die Mail-Konfiguration
func (s *Server) UpdateMailSettingsHandler(mailRepo *repository.MailSettingsRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req MailSettingsRequest
		if !DecodeAndValidate(w, r, &req) {
			return
		}

		// Absender beim Speichern prüfen, nicht erst beim Versand: Eine Adresse mit
		// Tippfehler ließ sich bisher speichern und fiel erst im Testversand auf — dort
		// als 500, die der Admin als Datenbankfehler zu lesen bekam. Leer ist erlaubt,
		// dafür setzt der Versand seinen Standardabsender ein.
		if sender := strings.TrimSpace(req.SenderEmail); sender != "" {
			parsed, err := mail.ParseAddress(sender)
			if err != nil {
				apierrors.SendHTTPError(w, http.StatusBadRequest, fmt.Errorf("ungültige Absender-E-Mail-Adresse: %w", err))
				return
			}
			req.SenderEmail = parsed.Address
		}

		ctx := r.Context()
		err := mailRepo.UpdateConfig(ctx, req.SMTPHost, req.SMTPPort, req.SMTPUser, req.SMTPPassword, req.SenderEmail)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		// Admin audit log
		if claims, ok := auth.GetClaims(r.Context()); ok {
			detailsBytes, merr := json.Marshal(map[string]interface{}{
				"smtp_host":        req.SMTPHost,
				"smtp_port":        req.SMTPPort,
				"smtp_user":        req.SMTPUser,
				"sender_email":     req.SenderEmail,
				"password_changed": req.SMTPPassword != "",
			})
			if merr != nil {
				log.Printf("audit: Mail-Settings-Details konnten nicht serialisiert werden: %v", merr)
			} else {
				logExec(s.DB.Pool.Exec(ctx, "INSERT INTO audit_logs (admin_id, aktion, details) VALUES ($1, $2, $3::jsonb)", claims.UserID, "UPDATE_MAIL_SETTINGS", string(detailsBytes)))
			}
		}

		RespondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}
}

// PostTestMailSettingsHandler sendet eine Test-E-Mail mit der aktuellen Konfiguration
func (s *Server) PostTestMailSettingsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			To string `json:"to"`
		}
		if !DecodeAndValidate(w, r, &req) {
			return
		}

		to := strings.TrimSpace(req.To)
		if to == "" {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("empfänger-E-Mail-Adresse fehlt"))
			return
		}

		// Die Adresse hier prüfen, nicht erst im Versand: Eine unbrauchbare Eingabe
		// ist ein Client-Fehler (400) und keine Serverstörung (500) — nur so kann das
		// Formular sie von einem echten SMTP-Ausfall unterscheiden. Die Hürde in
		// SendTestMail bleibt trotzdem stehen, sie schützt die anderen Aufrufer.
		parsed, err := mail.ParseAddress(to)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusBadRequest, fmt.Errorf("ungültige Empfänger-E-Mail-Adresse: %w", err))
			return
		}

		ctx := r.Context()
		err = mailservice.SendTestMail(ctx, s.DB.Pool, parsed.Address)
		if err != nil {
			// mailFehlerStatus: 502, wenn der Zielserver versagt hat — nur so erreicht
			// die Diagnose (falscher Port, abgelehnte Zugangsdaten, Zertifikatsname)
			// das Formular, für das sie geschrieben wurde.
			apierrors.SendHTTPError(w, mailFehlerStatus(err), err)
			return
		}

		RespondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}
}
