package api

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"

	"bibliothek/apierrors"
	"bibliothek/auth"
	"bibliothek/repository"
)

// Anliegen der Lehrkräfte (Betreiber-Entscheidung 18.08.2026): Wunsch
// ("Markl 2 für die 8G3") und Meldung ("falsche Bücher bekommen") als EIN
// schlanker Mechanismus. Wünschen geht IMMER (kein Stichtag), die LMF hakt in
// Ruhe ab, die Lehrkraft bekommt dabei eine Mail. Bewusst ohne Prioritäten,
// Kommentar-Threads oder Genehmigungsketten.

// AnliegenRequest ist die Eingabe der Lehrkraft im Kollegiums-Portal.
type AnliegenRequest struct {
	Art       string `json:"art" validate:"required"`
	TitelText string `json:"titel_text" validate:"required"`
	TitelID   string `json:"titel_id,omitempty"`
	ISBN      string `json:"isbn,omitempty"`
	Klasse    string `json:"klasse,omitempty"`
	Kommentar string `json:"kommentar,omitempty"`
}

// CreateAnliegenHandler nimmt Wunsch/Meldung entgegen. POST /api/anliegen
func (s *Server) CreateAnliegenHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := auth.GetClaims(r.Context())
		if !ok {
			apierrors.SendHTTPError(w, http.StatusUnauthorized, errors.New("Sitzungs-Information fehlt"))
			return
		}
		var req AnliegenRequest
		if !DecodeAndValidate(w, r, &req) {
			return
		}
		req.Art = strings.ToLower(strings.TrimSpace(req.Art))
		if req.Art != "wunsch" && req.Art != "meldung" {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("art muss 'wunsch' oder 'meldung' sein"))
			return
		}
		req.TitelText = strings.TrimSpace(req.TitelText)
		if req.TitelText == "" {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("worum geht es? titel_text ist leer"))
			return
		}
		// Längen kappen statt ablehnen — ein zu langer Wunsch soll nicht verloren
		// gehen, nur weil jemand einen ganzen Absatz ins Titelfeld kopiert hat.
		req.TitelText = kuerze(req.TitelText, 300)
		req.Klasse = kuerze(strings.TrimSpace(req.Klasse), 50)
		req.Kommentar = kuerze(strings.TrimSpace(req.Kommentar), 1000)
		req.ISBN = kuerze(strings.TrimSpace(req.ISBN), 20)

		repo := repository.NewAnliegenRepository(s.DB.Pool)
		id, err := repo.Create(r.Context(), req.Art, req.TitelText, strings.TrimSpace(req.TitelID),
			req.ISBN, req.Klasse, req.Kommentar, claims.UserID)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		RespondJSON(w, http.StatusCreated, map[string]string{"id": id})
	}
}

// kuerze schneidet einen String auf höchstens n Runen.
func kuerze(s string, n int) string {
	runen := []rune(s)
	if len(runen) <= n {
		return s
	}
	return string(runen[:n])
}

// ListEigeneAnliegenHandler zeigt der Lehrkraft ihre Anliegen samt Status.
// GET /api/anliegen/eigene
func (s *Server) ListEigeneAnliegenHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := auth.GetClaims(r.Context())
		if !ok {
			apierrors.SendHTTPError(w, http.StatusUnauthorized, errors.New("Sitzungs-Information fehlt"))
			return
		}
		anliegen, err := repository.NewAnliegenRepository(s.DB.Pool).ListEigene(r.Context(), claims.UserID)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		RespondJSON(w, http.StatusOK, anliegen)
	}
}

// ListOffeneAnliegenHandler ist die Arbeitsliste der LMF. GET /api/anliegen/offen
func (s *Server) ListOffeneAnliegenHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		anliegen, err := repository.NewAnliegenRepository(s.DB.Pool).ListOffene(r.Context())
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		RespondJSON(w, http.StatusOK, anliegen)
	}
}

// ErledigeAnliegenHandler hakt ab und benachrichtigt die Lehrkraft.
// PUT /api/anliegen/{id}/erledigen  { "notiz": "bestellt, kommt Anfang September" }
func (s *Server) ErledigeAnliegenHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		var body struct {
			Notiz string `json:"notiz"`
		}
		if !DecodeAndValidate(w, r, &body) {
			return
		}

		erledigt, err := repository.NewAnliegenRepository(s.DB.Pool).Erledige(r.Context(), id, kuerze(strings.TrimSpace(body.Notiz), 500))
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		if erledigt == nil {
			// Schon abgehakt (Doppelklick an zwei Arbeitsplätzen) oder unbekannt —
			// beides kein Serverfehler, und vor allem: KEINE zweite Mail.
			apierrors.SendHTTPError(w, http.StatusNotFound, errors.New("anliegen ist bereits erledigt oder unbekannt"))
			return
		}

		// Best effort wie die Klassensatz-Bereit-Mail: Das Abhaken gilt, auch wenn der
		// Mailversand klemmt. Aber nicht mehr spurlos (Ausfallmatrix 20.08.2026): Die
		// Antwort traegt den Mail-Status, damit die Theke weiss, ob die Lehrkraft
		// wirklich benachrichtigt wurde. Ohne Konto-Adresse keine Mail.
		mailStatus := mailStatusKeineAdresse
		if erledigt.AnfragendeMail != nil && *erledigt.AnfragendeMail != "" {
			mailStatus = mailStatusVersendet
			betreff := fmt.Sprintf("Ihr Wunsch ist erledigt: %s", erledigt.TitelText)
			if erledigt.Art == "meldung" {
				betreff = fmt.Sprintf("Ihre Meldung ist erledigt: %s", erledigt.TitelText)
			}
			text := fmt.Sprintf("Die Bibliothek hat Ihr Anliegen erledigt.\n\n  Betreff: %s\n", erledigt.TitelText)
			if erledigt.Klasse != "" {
				text += fmt.Sprintf("  Klasse:  %s\n", erledigt.Klasse)
			}
			if erledigt.ErledigtNotiz != "" {
				text += fmt.Sprintf("  Notiz:   %s\n", erledigt.ErledigtNotiz)
			}
			text += "\nDiese Mail wurde automatisch beim Abhaken verschickt."
			if err := SendEmail(MailRequest{To: *erledigt.AnfragendeMail, Subject: betreff, Body: text}); err != nil {
				log.Printf("Anliegen-Mail an %s fehlgeschlagen: %v", *erledigt.AnfragendeMail, err)
				mailStatus = mailStatusFehlgeschlagen
			}
		}
		RespondJSON(w, http.StatusOK, map[string]string{"mail": mailStatus})
	}
}
