package api

import (
	"encoding/json"
	"errors"
	"net/http"

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

		ctx := r.Context()
		err := mailRepo.UpdateConfig(ctx, req.SMTPHost, req.SMTPPort, req.SMTPUser, req.SMTPPassword, req.SenderEmail)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		// Admin audit log
		if claims, ok := auth.GetClaims(r.Context()); ok {
			detailsBytes, _ := json.Marshal(map[string]interface{}{
				"smtp_host": req.SMTPHost,
				"smtp_port": req.SMTPPort,
				"smtp_user": req.SMTPUser,
				"sender_email": req.SenderEmail,
				"password_changed": req.SMTPPassword != "",
			})
			_, _ = s.DB.Pool.Exec(ctx, "INSERT INTO audit_logs (admin_id, aktion, details) VALUES ($1, $2, $3::jsonb)", claims.UserID, "UPDATE_MAIL_SETTINGS", string(detailsBytes))
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

		if req.To == "" {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("empfänger-E-Mail-Adresse fehlt"))
			return
		}

		ctx := r.Context()
		err := mailservice.SendTestMail(ctx, s.DB.Pool, req.To)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		RespondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}
}
