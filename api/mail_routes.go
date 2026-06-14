package api

import (
	"bibliothek/apierrors"
	"errors"

	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"bibliothek/mailservice"
)

// PostSendOverdueNotificationHandler versendet eine Mahnung an die Eltern eines Schülers
func (s *Server) PostSendOverdueNotificationHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		schuelerID := r.PathValue("schuelerID")
		if schuelerID == "" {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("schuelerID fehlt"))
			return
		}

		ctx := r.Context()

		// 1. Schülerdaten & E-Mail abrufen
		var vorname, nachname string
		var elternEmail *string
		err := s.DB.Pool.QueryRow(ctx, "SELECT vorname, nachname, eltern_email FROM schueler WHERE id = $1", schuelerID).Scan(&vorname, &nachname, &elternEmail)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusNotFound, errors.New("Schüler nicht gefunden"))
			return
		}

		if elternEmail == nil || strings.TrimSpace(*elternEmail) == "" {
			apierrors.SendHTTPError(w, http.StatusUnprocessableEntity, errors.New("Keine Eltern-E-Mail hinterlegt"))
			return
		}

		// 2. Überfällige Ausleihen des Schülers abrufen
		rows, err := s.DB.Pool.Query(ctx, `
			SELECT bt.titel 
			FROM ausleihen a
			JOIN buecher_exemplare be ON a.exemplar_id = be.id
			JOIN buecher_titel bt ON be.titel_id = bt.id
			WHERE a.schueler_id = $1 AND a.rueckgabe_am IS NULL AND a.rueckgabe_frist < NOW()
		`, schuelerID)

		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, errors.New("Fehler beim Laden der Ausleihen"))
			return
		}
		defer rows.Close()

		var buecherListe []string
		for rows.Next() {
			var titel string
			if err := rows.Scan(&titel); err == nil {
				buecherListe = append(buecherListe, titel)
			}
		}

		if len(buecherListe) == 0 {
			apierrors.SendHTTPError(w, http.StatusUnprocessableEntity, errors.New("Keine überfälligen Bücher für diesen Schüler"))
			return
		}

		buecherString := strings.Join(buecherListe, "\n- ")

		// 3. Daten für das Template vorbereiten
		data := map[string]interface{}{
			"Name":    fmt.Sprintf("%s %s", vorname, nachname),
			"Vorname": vorname,
			"Buecher": "- " + buecherString,
		}

		// 4. Mail über das mailservice Package versenden
		err = mailservice.SendTemplateMail(ctx, s.DB.Pool, *elternEmail, "MAHNUNG_ELTERN", data)
		if err != nil {
			log.Printf("Fehler beim E-Mail-Versand (Schüler %s): %v", schuelerID, err)
			apierrors.SendHTTPError(w, http.StatusInternalServerError, errors.New("Fehler beim E-Mail-Versand"))
			return
		}

		// 5. Erfolgreiche Antwort
		RespondJSON(w, http.StatusOK, map[string]string{
			"message": "Mail an Eltern wurde verschickt",
		})
	}
}

// PostSendNotificationHandler versendet eine E-Mail an die Eltern, basierend auf dem im Body übergebenen templateType
func (s *Server) PostSendNotificationHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		schuelerID := r.PathValue("schuelerID")
		if schuelerID == "" {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("schuelerID fehlt"))
			return
		}

		var req struct {
			TemplateType string `json:"templateType"`
		}
		if !DecodeJSON(w, r, &req) {
			return
		}
		if req.TemplateType == "" {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("templateType fehlt"))
			return
		}

		ctx := r.Context()

		var vorname, nachname string
		var elternEmail *string
		err := s.DB.Pool.QueryRow(ctx, "SELECT vorname, nachname, eltern_email FROM schueler WHERE id = $1", schuelerID).Scan(&vorname, &nachname, &elternEmail)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusNotFound, errors.New("Schüler nicht gefunden"))
			return
		}

		if elternEmail == nil || strings.TrimSpace(*elternEmail) == "" {
			apierrors.SendHTTPError(w, http.StatusUnprocessableEntity, errors.New("Keine Eltern-E-Mail hinterlegt"))
			return
		}

		// Überfällige Ausleihen (für Mahnungen oder Bestellungen etc.)
		rows, err := s.DB.Pool.Query(ctx, `
			SELECT bt.titel 
			FROM ausleihen a
			JOIN buecher_exemplare be ON a.exemplar_id = be.id
			JOIN buecher_titel bt ON be.titel_id = bt.id
			WHERE a.schueler_id = $1 AND a.rueckgabe_am IS NULL
		`, schuelerID)

		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, errors.New("Fehler beim Laden der Ausleihen"))
			return
		}
		defer rows.Close()

		var buecherListe []string
		for rows.Next() {
			var titel string
			if err := rows.Scan(&titel); err == nil {
				buecherListe = append(buecherListe, titel)
			}
		}

		buecherString := ""
		if len(buecherListe) > 0 {
			buecherString = "- " + strings.Join(buecherListe, "\n- ")
		}

		// Daten für das Template vorbereiten (BuchListe statt Buecher wie vom User gewünscht)
		data := map[string]interface{}{
			"Name":      fmt.Sprintf("%s %s", vorname, nachname),
			"Vorname":   vorname,
			"BuchListe": buecherString,
		}

		err = mailservice.SendTemplateMail(ctx, s.DB.Pool, *elternEmail, req.TemplateType, data)
		if err != nil {
			log.Printf("Fehler beim E-Mail-Versand (Schüler %s): %v", schuelerID, err)
			apierrors.SendHTTPError(w, http.StatusInternalServerError, errors.New("Fehler beim E-Mail-Versand"))
			return
		}

		RespondJSON(w, http.StatusOK, map[string]string{
			"message": "Mail erfolgreich versendet",
		})
	}
}

// PostSendBulkOverdueHandler versendet Massen-Mahnungen an alle Schüler mit überfälligen Büchern.
func (s *Server) PostSendBulkOverdueHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// 1. Alle aktuell überfälligen Ausleihen abfragen und nach Schüler gruppieren
		rows, err := s.DB.Pool.Query(ctx, `
			SELECT s.id, s.vorname, s.nachname, s.eltern_email, bt.titel
			FROM ausleihen a
			JOIN schueler s ON a.schueler_id = s.id
			JOIN buecher_exemplare be ON a.exemplar_id = be.id
			JOIN buecher_titel bt ON be.titel_id = bt.id
			WHERE a.rueckgabe_am IS NULL AND a.rueckgabe_frist < NOW()
			ORDER BY s.id
		`)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, errors.New("Fehler beim Laden der überfälligen Ausleihen"))
			return
		}
		defer rows.Close()

		type studentOverdue struct {
			Vorname     string
			Nachname    string
			ElternEmail *string
			Buecher     []string
		}

		overdues := make(map[string]*studentOverdue)

		// Schleifen-Logik 1: Ausleihen lesen und pro Schüler aggregieren
		for rows.Next() {
			var id, vorname, nachname, titel string
			var email *string
			if err := rows.Scan(&id, &vorname, &nachname, &email, &titel); err == nil {
				if _, exists := overdues[id]; !exists {
					overdues[id] = &studentOverdue{
						Vorname:     vorname,
						Nachname:    nachname,
						ElternEmail: email,
						Buecher:     []string{},
					}
				}
				overdues[id].Buecher = append(overdues[id].Buecher, titel)
			}
		}

		ohneEmail := 0

		// Error-Handling / Edge-Case: Schüler ohne E-Mail separat zählen, um sie synchron zurückzugeben
		for _, data := range overdues {
			if data.ElternEmail == nil || strings.TrimSpace(*data.ElternEmail) == "" {
				ohneEmail++
			}
		}

		// Asynchroner Massenversand im Hintergrund
		go func(tasks map[string]*studentOverdue) {
			bgCtx := context.Background() // Eigener Context, der nicht mit dem HTTP Request abbricht
			for _, data := range tasks {
				if data.ElternEmail == nil || strings.TrimSpace(*data.ElternEmail) == "" {
					continue
				}

				buecherString := "- " + strings.Join(data.Buecher, "\n- ")
				tmplData := map[string]interface{}{
					"Name":      fmt.Sprintf("%s %s", data.Vorname, data.Nachname),
					"Vorname":   data.Vorname,
					"BuchListe": buecherString,
				}

				_ = mailservice.SendTemplateMail(bgCtx, s.DB.Pool, *data.ElternEmail, "MAHNUNG_ELTERN", tmplData)
				time.Sleep(100 * time.Millisecond) // Kleines Delay zur Schonung des SMTP-Servers
			}
		}(overdues)

		RespondJSON(w, http.StatusOK, map[string]any{
			"message":    "Massen-Versand wurde im Hintergrund gestartet",
			"ohne_email": ohneEmail,
		})
	}
}

// GetMailTemplatesHandler gibt alle Mail-Vorlagen zurück
func (s *Server) GetMailTemplatesHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		rows, err := s.DB.Pool.Query(ctx, "SELECT id, typ, betreff, text_body, updated_at FROM mail_vorlagen ORDER BY typ ASC")
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, errors.New("Fehler beim Laden der Vorlagen"))
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
		if !DecodeJSON(w, r, &req) {
			return
		}

		ctx := r.Context()
		_, err := s.DB.Pool.Exec(ctx, "UPDATE mail_vorlagen SET betreff = $1, text_body = $2 WHERE id = $3", req.Betreff, req.TextBody, id)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, errors.New("Fehler beim Aktualisieren der Vorlage"))
			return
		}

		RespondJSON(w, http.StatusOK, map[string]string{"message": "Erfolgreich gespeichert"})
	}
}
