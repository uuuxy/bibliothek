package portal

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"bibliothek/api/tuer"
	"bibliothek/apierrors"
	"bibliothek/auth"
	"bibliothek/mailservice"
	"bibliothek/repository"
)

// Anliegen der Lehrkräfte: die Meldung, dass etwas nicht stimmt („falsche Bücher
// bekommen"). Die Bibliothek hakt in Ruhe ab, die Lehrkraft bekommt dabei eine Mail. Ohne
// Prioritäten, Kommentar-Threads oder Genehmigungsketten.
//
// Die Tabelle führt daneben Wünsche (art 'wunsch'), die das Portal nicht mehr anlegt: Wer
// ein Buch für eine Klasse möchte, reserviert es dort. Vorhandene Wünsche werden wie
// Meldungen gelistet und abgehakt.

// Was aus der Mail beim Erledigen wurde. Die Türen, die danach eine Lehrkraft benachrichtigen
// (Klassensatz bereit, Anliegen erledigt), nennen es in der Antwort: Ein Ausfall des Versands
// stünde sonst nur im Log, und die Theke hielte die Lehrkraft für benachrichtigt. Bei
// „fehlgeschlagen" warnt die Oberfläche.
const (
	mailStatusVersendet      = "versendet"
	mailStatusFehlgeschlagen = "fehlgeschlagen"
	mailStatusKeineAdresse   = "keine_adresse"
)

// anliegenAnfrage ist die Eingabe der Lehrkraft im Kollegiums-Portal.
//
// Freitext: Am Treffer gemeldet, steht der Titel des Buchs im Text; ohne Buch nennt die
// Lehrkraft selbst, worum es geht. Die Spalten titel_id und isbn in lehrer_anliegen füllt
// die Tür nicht — eine Tür, die Felder annimmt, die kein Formular schickt, sieht aus wie
// eine Funktion.
type anliegenAnfrage struct {
	Art       string `json:"art" validate:"required"`
	TitelText string `json:"titel_text" validate:"required"`
	Klasse    string `json:"klasse,omitempty"`
	Kommentar string `json:"kommentar,omitempty"`
}

// AnliegenAnlegen nimmt eine Meldung entgegen. POST /api/anliegen
func (t *Tueren) AnliegenAnlegen() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := auth.ClaimsAusKontext(r.Context())
		if !ok {
			apierrors.AntworteFehlerMitMeldung(w, http.StatusUnauthorized, auth.MeldungNichtAngemeldet, nil)
			return
		}
		var req anliegenAnfrage
		if !tuer.LeseAnfrage(w, r, &req) {
			return
		}
		req.Art = strings.ToLower(strings.TrimSpace(req.Art))
		if req.Art != "meldung" {
			apierrors.AntworteFehlerMitMeldung(w, http.StatusBadRequest, "Hier geht nur eine Meldung. Ein Buch für eine Klasse wird im Portal reserviert.", nil)
			return
		}
		req.TitelText = strings.TrimSpace(req.TitelText)
		if req.TitelText == "" {
			apierrors.AntworteFehlerMitMeldung(w, http.StatusBadRequest, "Bitte angeben, worum es geht", nil)
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
			apierrors.AntworteFehlerMitMeldung(w, http.StatusBadRequest, "Bitte beschreiben, was nicht stimmt", nil)
			return
		}

		abfragen := repository.NewAnliegenAbfragen(t.DB.Pool)
		id, err := abfragen.LegeAn(r.Context(), repository.NeuesAnliegen{
			Art:            req.Art,
			TitelText:      req.TitelText,
			Klasse:         req.Klasse,
			Kommentar:      req.Kommentar,
			AngefordertVon: claims.KontoID,
		})
		if err != nil {
			apierrors.AntworteFehler(w, http.StatusInternalServerError, err)
			return
		}
		tuer.AntworteJSON(w, http.StatusCreated, map[string]string{"id": id})
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

// EigeneAnliegen zeigt der Lehrkraft ihre Anliegen samt Status.
// GET /api/anliegen/eigene
func (t *Tueren) EigeneAnliegen() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := auth.ClaimsAusKontext(r.Context())
		if !ok {
			apierrors.AntworteFehlerMitMeldung(w, http.StatusUnauthorized, auth.MeldungNichtAngemeldet, nil)
			return
		}
		anliegen, err := repository.NewAnliegenAbfragen(t.DB.Pool).LadeEigene(r.Context(), claims.KontoID)
		if err != nil {
			apierrors.AntworteFehler(w, http.StatusInternalServerError, err)
			return
		}
		tuer.AntworteJSON(w, http.StatusOK, anliegen)
	}
}

// OffeneAnliegen ist die Arbeitsliste der LMF. GET /api/anliegen/offen
func (t *Tueren) OffeneAnliegen() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		anliegen, err := repository.NewAnliegenAbfragen(t.DB.Pool).LadeOffene(r.Context())
		if err != nil {
			apierrors.AntworteFehler(w, http.StatusInternalServerError, err)
			return
		}
		tuer.AntworteJSON(w, http.StatusOK, anliegen)
	}
}

// OffeneAnliegenAnzahl speist den Zähler am Menüpunkt „Bestellungen“: Ohne ihn sähe offene
// Anliegen nur, wer ohnehin bestellen will.
// GET /api/anliegen/anzahl → { "anzahl": 3 }
func (t *Tueren) OffeneAnliegenAnzahl() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		n, err := repository.NewAnliegenAbfragen(t.DB.Pool).ZaehleOffene(r.Context())
		if err != nil {
			apierrors.AntworteFehler(w, http.StatusInternalServerError, err)
			return
		}
		tuer.AntworteJSON(w, http.StatusOK, map[string]int{"anzahl": n})
	}
}

// AnliegenErledigen hakt ab und benachrichtigt die Lehrkraft.
// PUT /api/anliegen/{id}/erledigen  { "notiz": "bestellt, kommt Anfang September" }
func (t *Tueren) AnliegenErledigen() http.HandlerFunc {
	return t.erledigeAnliegen
}

// erledigeAnliegen hakt das Anliegen ab und benachrichtigt, wer es gestellt hat.
func (t *Tueren) erledigeAnliegen(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var body struct {
		Notiz string `json:"notiz"`
	}
	if !tuer.LeseAnfrage(w, r, &body) {
		return
	}

	erledigt, err := repository.NewAnliegenAbfragen(t.DB.Pool).Erledige(r.Context(), id, kuerze(strings.TrimSpace(body.Notiz), 500))
	if err != nil {
		apierrors.AntworteFehler(w, http.StatusInternalServerError, err)
		return
	}
	if erledigt == nil {
		// Schon abgehakt (Doppelklick an zwei Arbeitsplätzen) oder unbekannt —
		// beides kein Serverfehler, und vor allem: keine zweite Mail.
		apierrors.AntworteFehlerMitMeldung(w, http.StatusNotFound, "Das Anliegen ist bereits erledigt oder unbekannt", nil)
		return
	}

	// Das Abhaken gilt, auch wenn der Mailversand klemmt. Die Antwort trägt den
	// Mail-Status, damit die Theke weiß, ob die Lehrkraft wirklich benachrichtigt wurde.
	tuer.AntworteJSON(w, http.StatusOK, map[string]string{"mail": t.sendeAnliegenErledigtMail(erledigt)})
}

// sendeAnliegenErledigtMail benachrichtigt, wer das Anliegen gestellt hat, und nennt, was aus
// dem Versand wurde. Ohne Adresse am Konto geht keine Mail.
func (t *Tueren) sendeAnliegenErledigtMail(erledigt *repository.AnliegenErledigt) string {
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
	if err := t.Post().Sende(mailservice.Nachricht{An: *erledigt.AnfragendeMail, Betreff: betreff, Text: text}); err != nil {
		log.Printf("Anliegen-Mail an %s fehlgeschlagen: %v", *erledigt.AnfragendeMail, err)
		return mailStatusFehlgeschlagen
	}
	return mailStatusVersendet
}
