package api

import (
	"bibliothek/apierrors"
	"errors"
	"net/http"
	"time"
)

// PostSendOverdueNotificationHandler versendet eine Mahnung an die Eltern eines Schülers
func (s *Server) PostSendOverdueNotificationHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		apierrors.SendHTTPError(w, http.StatusNotImplemented, errors.New("E-Mail-Versand aus Datenschutzgründen deaktiviert (keine Eltern-E-Mails mehr gespeichert)"))
	}
}

// PostSendNotificationHandler versendet eine E-Mail an die Eltern, basierend auf dem im Body übergebenen templateType
func (s *Server) PostSendNotificationHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		apierrors.SendHTTPError(w, http.StatusNotImplemented, errors.New("E-Mail-Versand aus Datenschutzgründen deaktiviert (keine Eltern-E-Mails mehr gespeichert)"))
	}
}

// PostSendBulkOverdueHandler versendet Massen-Mahnungen an alle Schüler mit überfälligen Büchern.
func (s *Server) PostSendBulkOverdueHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		apierrors.SendHTTPError(w, http.StatusNotImplemented, errors.New("E-Mail-Versand aus Datenschutzgründen deaktiviert (keine Eltern-E-Mails mehr gespeichert)"))
	}
}

// GetMailTemplatesHandler gibt alle Mail-Vorlagen zurück
func (s *Server) GetMailTemplatesHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		rows, err := s.DB.Pool.Query(ctx, "SELECT id, typ, betreff, text_body, updated_at FROM mail_vorlagen ORDER BY typ ASC")
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, errors.New("fehler beim Laden der Vorlagen"))
			return
		}
		defer rows.Close()

		type MailTemplate struct {
			ID        string `json:"id"`
			Typ       string `json:"typ"`
			Betreff   string `json:"betreff"`
			TextBody  string `json:"text_body"`
			UpdatedAt string `json:"updated_at"`
		}

		var templates []MailTemplate
		for rows.Next() {
			var t MailTemplate
			var ts time.Time
			if err := rows.Scan(&t.ID, &t.Typ, &t.Betreff, &t.TextBody, &ts); err == nil {
				t.UpdatedAt = ts.Format(time.RFC3339)
				templates = append(templates, t)
			}
		}
		if err := rows.Err(); err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, errors.New("fehler beim Laden der Vorlagen"))
			return
		}

		RespondJSON(w, http.StatusOK, templates)
	}
}

// UpdateMailTemplateHandler aktualisiert eine Mail-Vorlage
func (s *Server) UpdateMailTemplateHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if id == "" {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("ID fehlt"))
			return
		}

		var req struct {
			Betreff  string `json:"betreff"`
			TextBody string `json:"text_body"`
		}
		if !DecodeAndValidate(w, r, &req) {
			return
		}

		ctx := r.Context()
		_, err := s.DB.Pool.Exec(ctx, "UPDATE mail_vorlagen SET betreff = $1, text_body = $2 WHERE id = $3", req.Betreff, req.TextBody, id)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, errors.New("fehler beim Aktualisieren der Vorlage"))
			return
		}

		RespondJSON(w, http.StatusOK, map[string]string{"message": "Erfolgreich gespeichert"})
	}
}
