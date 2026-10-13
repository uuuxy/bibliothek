package system

import (
	"errors"
	"net/http"
	"time"

	"bibliothek/api/tuer"
	"bibliothek/apierrors"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5"
)

// GetMailTemplatesHandler gibt alle Mail-Vorlagen zurück
func (s *Tueren) GetMailTemplatesHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		vorlagen, err := repository.LadeMailVorlagen(ctx, s.DB.Pool)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, errors.New("fehler beim Laden der Vorlagen"))
			return
		}

		type MailTemplate struct {
			ID        string `json:"id"`
			Typ       string `json:"typ"`
			Betreff   string `json:"betreff"`
			TextBody  string `json:"text_body"`
			UpdatedAt string `json:"updated_at"`
		}

		var templates []MailTemplate
		for _, v := range vorlagen {
			templates = append(templates, MailTemplate{
				ID: v.ID, Typ: v.Typ, Betreff: v.Betreff, TextBody: v.TextBody,
				UpdatedAt: v.AktualisiertAm.Format(time.RFC3339),
			})
		}

		tuer.RespondJSON(w, http.StatusOK, templates)
	}
}

// UpdateMailTemplateHandler aktualisiert eine Mail-Vorlage
func (s *Tueren) UpdateMailTemplateHandler() http.HandlerFunc {
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
		if !tuer.DecodeAndValidate(w, r, &req) {
			return
		}

		ctx := r.Context()
		// Der Stand davor sagt, was sich geändert hat: Im Protokoll stehen die Vorlage und
		// die Namen der geänderten Felder, nicht der Wortlaut.
		typ, betreffNeu, textNeu, err := repository.AendereMailVorlage(ctx, s.DB.Pool, id, req.Betreff, req.TextBody)
		// Eine unbekannte Vorlage ist ein Fehler, kein „Erfolgreich gespeichert".
		if errors.Is(err, pgx.ErrNoRows) {
			apierrors.SendHTTPError(w, http.StatusNotFound, errors.New("mail-Vorlage nicht gefunden"))
			return
		}
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, errors.New("fehler beim Aktualisieren der Vorlage"))
			return
		}
		s.ProtokolliereGeaenderteFelder(ctx, tuer.AuditMailvorlageGeaendert, map[string]any{"vorlage": typ},
			tuer.GeaenderteFelder(tuer.FeldWechsel{Name: "betreff", Geaendert: betreffNeu}, tuer.FeldWechsel{Name: "text", Geaendert: textNeu}))

		tuer.RespondJSON(w, http.StatusOK, map[string]string{"message": "Erfolgreich gespeichert"})
	}
}
