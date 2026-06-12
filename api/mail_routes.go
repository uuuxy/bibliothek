package api

import (
	"context"
	"encoding/json"
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
			http.Error(w, "schuelerID fehlt", http.StatusBadRequest)
			return
		}

		ctx := r.Context()

		// 1. Schülerdaten & E-Mail abrufen
		var vorname, nachname string
		var elternEmail *string
		err := s.DB.Pool.QueryRow(ctx, "SELECT vorname, nachname, eltern_email FROM schueler WHERE id = $1", schuelerID).Scan(&vorname, &nachname, &elternEmail)
		if err != nil {
			http.Error(w, "Schüler nicht gefunden", http.StatusNotFound)
			return
		}

		if elternEmail == nil || strings.TrimSpace(*elternEmail) == "" {
			http.Error(w, "Keine Eltern-E-Mail hinterlegt", http.StatusUnprocessableEntity)
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
			http.Error(w, "Fehler beim Laden der Ausleihen", http.StatusInternalServerError)
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
			http.Error(w, "Keine überfälligen Bücher für diesen Schüler", http.StatusUnprocessableEntity)
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
			http.Error(w, "Fehler beim E-Mail-Versand", http.StatusInternalServerError)
			return
		}

		// 5. Erfolgreiche Antwort
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"message": "Mail an Eltern wurde verschickt",
		})
	}
}

// PostSendNotificationHandler versendet eine E-Mail an die Eltern, basierend auf dem im Body übergebenen templateType
func (s *Server) PostSendNotificationHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		schuelerID := r.PathValue("schuelerID")
		if schuelerID == "" {
			http.Error(w, "schuelerID fehlt", http.StatusBadRequest)
			return
		}

		var req struct {
			TemplateType string `json:"templateType"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "ungültiger Body", http.StatusBadRequest)
			return
		}
		if req.TemplateType == "" {
			http.Error(w, "templateType fehlt", http.StatusBadRequest)
			return
		}

		ctx := r.Context()

		var vorname, nachname string
		var elternEmail *string
		err := s.DB.Pool.QueryRow(ctx, "SELECT vorname, nachname, eltern_email FROM schueler WHERE id = $1", schuelerID).Scan(&vorname, &nachname, &elternEmail)
		if err != nil {
			http.Error(w, "Schüler nicht gefunden", http.StatusNotFound)
			return
		}

		if elternEmail == nil || strings.TrimSpace(*elternEmail) == "" {
			http.Error(w, "Keine Eltern-E-Mail hinterlegt", http.StatusUnprocessableEntity)
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
			http.Error(w, "Fehler beim Laden der Ausleihen", http.StatusInternalServerError)
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
			http.Error(w, "Fehler beim E-Mail-Versand", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
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
			http.Error(w, "Fehler beim Laden der überfälligen Ausleihen", http.StatusInternalServerError)
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

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
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
			http.Error(w, "Fehler beim Laden der Vorlagen", http.StatusInternalServerError)
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

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(templates)
	}
}

// UpdateMailTemplateHandler aktualisiert eine Mail-Vorlage
func (s *Server) UpdateMailTemplateHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if id == "" {
			http.Error(w, "ID fehlt", http.StatusBadRequest)
			return
		}

		var req struct {
			Betreff  string `json:"betreff"`
			TextBody string `json:"text_body"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "ungültiger Body", http.StatusBadRequest)
			return
		}

		ctx := r.Context()
		_, err := s.DB.Pool.Exec(ctx, "UPDATE mail_vorlagen SET betreff = $1, text_body = $2 WHERE id = $3", req.Betreff, req.TextBody, id)
		if err != nil {
			http.Error(w, "Fehler beim Aktualisieren der Vorlage", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"message": "Erfolgreich gespeichert"})
	}
}
