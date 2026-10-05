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

// Anliegen der Lehrkräfte: die Meldung, dass etwas nicht stimmt („falsche Bücher
// bekommen"). Die Bibliothek hakt in Ruhe ab, die Lehrkraft bekommt dabei eine Mail. Ohne
// Prioritäten, Kommentar-Threads oder Genehmigungsketten.
//
// Die Tabelle führt daneben Wünsche (art 'wunsch'), die das Portal nicht mehr anlegt: Wer
// ein Buch für eine Klasse möchte, reserviert es dort. Vorhandene Wünsche werden wie
// Meldungen gelistet und abgehakt.

// AnliegenRequest ist die Eingabe der Lehrkraft im Kollegiums-Portal.
//
// Freitext: Am Treffer gemeldet, steht der Titel des Buchs im Text; ohne Buch nennt die
// Lehrkraft selbst, worum es geht. Die Spalten titel_id und isbn in lehrer_anliegen füllt
// die Tür nicht — eine Tür, die Felder annimmt, die kein Formular schickt, sieht aus wie
// eine Funktion.
type AnliegenRequest struct {
	Art       string `json:"art" validate:"required"`
	TitelText string `json:"titel_text" validate:"required"`
	Klasse    string `json:"klasse,omitempty"`
	Kommentar string `json:"kommentar,omitempty"`
}

// CreateAnliegenHandler nimmt eine Meldung entgegen. POST /api/anliegen
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
		if req.Art != "meldung" {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("art muss 'meldung' sein — ein Buch für eine Klasse wird im Portal reserviert"))
			return
		}
		req.TitelText = strings.TrimSpace(req.TitelText)
		if req.TitelText == "" {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("worum geht es? titel_text ist leer"))
			return
		}
		// Längen kappen statt ablehnen — eine zu lange Meldung soll nicht verloren
		// gehen, nur weil jemand einen ganzen Absatz ins Titelfeld kopiert hat.
		req.TitelText = kuerze(req.TitelText, 300)
		req.Klasse = kuerze(strings.TrimSpace(req.Klasse), 50)
		req.Kommentar = kuerze(strings.TrimSpace(req.Kommentar), 1000)
		// Ohne Beschreibung nennt die Meldung nur ein Buch, und die Bibliothek muss
		// nachfragen. Das Formular verlangt den Satz; die Tür hält dieselbe Regel.
		if req.Kommentar == "" {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("was stimmt nicht? kommentar ist leer"))
			return
		}

		repo := repository.NewAnliegenRepository(s.DB.Pool)
		id, err := repo.Create(r.Context(), repository.NeuesAnliegen{
			Art:            req.Art,
			TitelText:      req.TitelText,
			Klasse:         req.Klasse,
			Kommentar:      req.Kommentar,
			AngefordertVon: claims.UserID,
		})
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

// CountOffeneAnliegenHandler speist das Badge an „Bestellungen“ — bis 24.08.2026 zählte
// niemand offene Anliegen, und wer nicht ohnehin bestellen wollte, sah sie nie.
// GET /api/anliegen/anzahl → { "anzahl": 3 }
func (s *Server) CountOffeneAnliegenHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		n, err := repository.NewAnliegenRepository(s.DB.Pool).CountOffene(r.Context())
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		RespondJSON(w, http.StatusOK, map[string]int{"anzahl": n})
	}
}

// ErledigeAnliegenHandler hakt ab und benachrichtigt die Lehrkraft.
// PUT /api/anliegen/{id}/erledigen  { "notiz": "bestellt, kommt Anfang September" }
func (s *Server) ErledigeAnliegenHandler() http.HandlerFunc {
	return s.handleErledigeAnliegen
}

// handleErledigeAnliegen hakt das Anliegen ab und benachrichtigt, wer es gestellt hat.
func (s *Server) handleErledigeAnliegen(w http.ResponseWriter, r *http.Request) {
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
		// beides kein Serverfehler, und vor allem: keine zweite Mail.
		apierrors.SendHTTPError(w, http.StatusNotFound, errors.New("anliegen ist bereits erledigt oder unbekannt"))
		return
	}

	// Das Abhaken gilt, auch wenn der Mailversand klemmt. Die Antwort trägt den
	// Mail-Status, damit die Theke weiß, ob die Lehrkraft wirklich benachrichtigt wurde.
	RespondJSON(w, http.StatusOK, map[string]string{"mail": sendeAnliegenErledigtMail(erledigt)})
}

// sendeAnliegenErledigtMail benachrichtigt, wer das Anliegen gestellt hat, und nennt, was aus
// dem Versand wurde. Ohne Adresse am Konto geht keine Mail.
func sendeAnliegenErledigtMail(erledigt *repository.AnliegenErledigt) string {
	if erledigt.AnfragendeMail == nil || *erledigt.AnfragendeMail == "" {
		return mailStatusKeineAdresse
	}
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
		return mailStatusFehlgeschlagen
	}
	return mailStatusVersendet
}
