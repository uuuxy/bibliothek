package api

import (
	"errors"
	"net/http"
	"time"

	"bibliothek/apierrors"

	"github.com/jackc/pgx/v5"
)

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
		// Der Stand davor sagt, was sich geändert hat: Im Protokoll stehen die Vorlage und
		// die Namen der geänderten Felder, nicht der Wortlaut.
		var typ string
		var betreffNeu, textNeu bool
		err := s.DB.Pool.QueryRow(ctx, `
			WITH alt AS (SELECT betreff, text_body FROM mail_vorlagen WHERE id = $3 FOR UPDATE)
			UPDATE mail_vorlagen v SET betreff = $1, text_body = $2
			  FROM alt
			 WHERE v.id = $3
			RETURNING v.typ, alt.betreff IS DISTINCT FROM $1, alt.text_body IS DISTINCT FROM $2
		`, req.Betreff, req.TextBody, id).Scan(&typ, &betreffNeu, &textNeu)
		// Eine unbekannte Vorlage ist ein Fehler, kein „Erfolgreich gespeichert".
		if errors.Is(err, pgx.ErrNoRows) {
			apierrors.SendHTTPError(w, http.StatusNotFound, errors.New("mail-Vorlage nicht gefunden"))
			return
		}
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, errors.New("fehler beim Aktualisieren der Vorlage"))
			return
		}
		s.protokolliereGeaenderteFelder(ctx, auditMailvorlageGeaendert, map[string]any{"vorlage": typ},
			geaenderteFelder(feldWechsel{"betreff", betreffNeu}, feldWechsel{"text", textNeu}))

		RespondJSON(w, http.StatusOK, map[string]string{"message": "Erfolgreich gespeichert"})
	}
}
