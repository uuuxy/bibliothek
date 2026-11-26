package system

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/mail"
	"slices"
	"strings"

	"bibliothek/internal/api/tuer"
	"bibliothek/internal/apierrors"
	"bibliothek/internal/auth"
	"bibliothek/internal/mailservice"
	"bibliothek/internal/repository"
)

// mailEinstellungenAntwort ist die Mail-Konfiguration, wie die Oberfläche sie sehen darf.
// Das Passwort verlässt den Server nie; stattdessen sagt HatPasswort nur, ob eines
// hinterlegt ist.
type mailEinstellungenAntwort struct {
	SMTPHost      string            `json:"smtp_host"`
	SMTPPort      string            `json:"smtp_port"`
	SMTPBenutzer  string            `json:"smtp_user"`
	AbsenderEmail string            `json:"sender_email"`
	HatPasswort   bool              `json:"has_password"`
	Portal        mailPortalAntwort `json:"portal"`
}

// mailPortalAntwort ist der zweite Absender, ohne Passwort wie oben. Fuer sind die Arten von
// Mails, die über ihn gehen, Arten alle, die sich wählen lassen (mailservice.Mailarten).
type mailPortalAntwort struct {
	AbsenderEmail string   `json:"sender_email"`
	SMTPBenutzer  string   `json:"smtp_user"`
	HatPasswort   bool     `json:"has_password"`
	Fuer          []string `json:"fuer"`
	Arten         []string `json:"arten"`
}

// mailPortalAnfrage ist der eingehende zweite Absender. Eine leere Adresse heißt, alle Mails
// gehen über den Postausgang; ein leerer Benutzer, mit dessen Zugang. Fehlt Fuer, bleiben die
// gespeicherten Arten stehen; eine leere Liste nimmt alle heraus.
type mailPortalAnfrage struct {
	AbsenderEmail string   `json:"sender_email"`
	SMTPBenutzer  string   `json:"smtp_user"`
	SMTPPasswort  string   `json:"smtp_password"` // leer: das gespeicherte Passwort bleibt
	Fuer          []string `json:"fuer"`
}

// mailEinstellungenAnfrage ist die eingehende Mail-Konfiguration. Ein leeres SMTPPasswort
// bedeutet „unverändert lassen": Sonst würde jedes Speichern der Einstellungsseite das
// Passwort löschen, weil die Oberfläche es gar nicht kennt (siehe mailEinstellungenAntwort).
// Ein leerer Port heißt 587, wie im Versand und in der Umgebung (mailservice.ladeSMTPKonfig).
type mailEinstellungenAnfrage struct {
	SMTPHost      string `json:"smtp_host"`
	SMTPPort      string `json:"smtp_port"`
	SMTPBenutzer  string `json:"smtp_user"`
	SMTPPasswort  string `json:"smtp_password"` // leer: das gespeicherte Passwort bleibt
	AbsenderEmail string `json:"sender_email"`
	// Fehlt Portal, bleibt der gespeicherte zweite Absender unangetastet: Eine Seite, die vor dem
	// Update geladen wurde, kennt das Feld nicht und löschte ihn sonst beim Speichern.
	Portal *mailPortalAnfrage `json:"portal"`
}

// MailEinstellungen gibt die Mail-Konfiguration zurück
func (t *Tueren) MailEinstellungen(mailAbfragen *repository.MailEinstellungsAbfragen) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		konfig, err := mailAbfragen.Lade(ctx)
		if err != nil {
			// Gibt es noch keine Konfiguration, kommen die Vorgaben zurück.
			konfig = &repository.MailEinstellungen{
				SMTPHost:      "localhost",
				SMTPPort:      "1025",
				SMTPBenutzer:  "",
				AbsenderEmail: "noreply@bibliothek-schule.de",
			}
		}

		antwort := mailEinstellungenAntwort{
			SMTPHost:      konfig.SMTPHost,
			SMTPPort:      konfig.SMTPPort,
			SMTPBenutzer:  konfig.SMTPBenutzer,
			AbsenderEmail: konfig.AbsenderEmail,
			HatPasswort:   len(konfig.SMTPPasswortVerschluesselt) > 0,
			Portal: mailPortalAntwort{
				AbsenderEmail: konfig.PortalAbsenderEmail,
				SMTPBenutzer:  konfig.PortalSMTPBenutzer,
				HatPasswort:   len(konfig.PortalSMTPPasswortVerschluesselt) > 0,
				Fuer:          ordneMailarten(konfig.PortalAbsenderFuer),
				Arten:         ordneMailarten(nil),
			},
		}

		tuer.AntworteJSON(w, http.StatusOK, antwort)
	}
}

// leseMailEinstellungen liest die Mail-Konfiguration aus dem Rumpf und prüft sie. Bei einem
// Fehler hat sie mit 400 geantwortet und liefert false.
func leseMailEinstellungen(w http.ResponseWriter, r *http.Request) (mailEinstellungenAnfrage, bool) {
	// Streng, weil die Tür jedes Feld schreibt: Ein Feld, das sie nicht kennt, ginge sonst
	// verloren, und die fehlenden stünden danach leer in der Datenbank.
	var req mailEinstellungenAnfrage
	if !tuer.LeseAnfrageStreng(w, r, &req) {
		return req, false
	}

	// Ohne Server nähme der Versand still die Umgebung statt dieser Einstellung, und Port,
	// Benutzer und Absender wären verloren.
	req.SMTPHost = strings.TrimSpace(req.SMTPHost)
	if req.SMTPHost == "" {
		apierrors.AntworteFehlerMitMeldung(w, http.StatusBadRequest, "Bitte das Feld „SMTP Host“ ausfüllen", nil)
		return req, false
	}

	// Absender beim Speichern prüfen, nicht erst beim Versand: Eine Adresse mit
	// Tippfehler fiele sonst erst im Testversand auf, als 500, die der Admin als
	// Datenbankfehler läse. Leer ist erlaubt; dann setzt der Versand seinen
	// Vorgabe-Absender ein.
	if absender := strings.TrimSpace(req.AbsenderEmail); absender != "" {
		adresse, err := mail.ParseAddress(absender)
		if err != nil {
			apierrors.AntworteFehlerMitMeldung(w, http.StatusBadRequest, "Die E-Mail-Adresse des Absenders ist ungültig", err)
			return req, false
		}
		req.AbsenderEmail = adresse.Address
	}

	if req.Portal != nil && !pruefePortalAbsender(w, req.Portal) {
		return req, false
	}
	return req, true
}

// pruefePortalAbsender prüft den zweiten Absender und bringt Adresse, Benutzer und Arten in ihre
// feste Form. Bei einem Fehler hat sie mit 400 geantwortet und liefert false.
func pruefePortalAbsender(w http.ResponseWriter, p *mailPortalAnfrage) bool {
	p.SMTPBenutzer = strings.TrimSpace(p.SMTPBenutzer)
	p.AbsenderEmail = strings.TrimSpace(p.AbsenderEmail)
	if p.AbsenderEmail != "" {
		adresse, err := mail.ParseAddress(p.AbsenderEmail)
		if err != nil {
			apierrors.AntworteFehlerMitMeldung(w, http.StatusBadRequest, "Die E-Mail-Adresse des zweiten Absenders ist ungültig", err)
			return false
		}
		p.AbsenderEmail = adresse.Address
	}
	if p.Fuer == nil {
		return true
	}
	// Eine unbekannte Art fiele beim Ordnen still weg; die Tür lehnt sie deshalb vorher ab.
	for _, art := range p.Fuer {
		if !slices.Contains(mailservice.Mailarten, mailservice.Mailart(art)) {
			apierrors.AntworteFehlerMitMeldung(w, http.StatusBadRequest, "Unbekannte Art von Mail für den zweiten Absender", nil)
			return false
		}
	}
	p.Fuer = ordneMailarten(p.Fuer)
	return true
}

// ordneMailarten liefert die genannten Arten in der Reihenfolge von mailservice.Mailarten, jede
// einmal; nil nennt alle. Die Antwort trägt so nie null, und gespeichert wird eine feste Form.
func ordneMailarten(genannt []string) []string {
	geordnet := []string{}
	for _, art := range mailservice.Mailarten {
		if genannt == nil || slices.Contains(genannt, string(art)) {
			geordnet = append(geordnet, string(art))
		}
	}
	return geordnet
}

// MailEinstellungenSchreiben speichert die Mail-Konfiguration
func (t *Tueren) MailEinstellungenSchreiben(mailAbfragen *repository.MailEinstellungsAbfragen) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req, ok := leseMailEinstellungen(w, r)
		if !ok {
			return
		}

		if !pruefePasswortZumNeuenServer(r.Context(), w, mailAbfragen, req) {
			return
		}
		if err := schreibeMailEinstellungen(r.Context(), mailAbfragen, req); err != nil {
			apierrors.AntworteFehler(w, http.StatusInternalServerError, err)
			return
		}
		t.protokolliereMailEinstellungen(r.Context(), req)

		tuer.AntworteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}
}

// schreibeMailEinstellungen schreibt den Postausgang und, wenn die Anfrage ihn nennt, den
// zweiten Absender.
// Meldungen, wenn der Mailserver wechselt und ein gespeichertes Passwort mitginge.
const (
	meldungPasswortZumNeuenServer       = "Der Mailserver hat sich geändert. Bitte das Passwort neu eintragen."
	meldungPortalPasswortZumNeuenServer = "Der Mailserver hat sich geändert. Bitte auch das Passwort des zweiten Absenders neu eintragen."
)

// pruefePasswortZumNeuenServer verlangt bei einem anderen Mailserver die gespeicherten Passwörter
// neu: Sie gingen sonst beim nächsten Versand an den neuen Server, und wer Einstellungen ändern
// darf, läse sie dort mit. Bei einem Fehler hat sie geantwortet und liefert false.
func pruefePasswortZumNeuenServer(ctx context.Context, w http.ResponseWriter, mailAbfragen *repository.MailEinstellungsAbfragen, req mailEinstellungenAnfrage) bool {
	konfig, err := mailAbfragen.Lade(ctx)
	if err != nil {
		apierrors.AntworteFehler(w, http.StatusInternalServerError, err)
		return false
	}
	if strings.EqualFold(strings.TrimSpace(konfig.SMTPHost), req.SMTPHost) {
		return true
	}
	if len(konfig.SMTPPasswortVerschluesselt) > 0 && req.SMTPPasswort == "" {
		apierrors.AntworteFehlerMitMeldung(w, http.StatusBadRequest, meldungPasswortZumNeuenServer, nil)
		return false
	}
	// Das Passwort des zweiten Absenders nur, wenn er mit eigenem Zugang bleibt: Wer ihn im selben
	// Speichern entfernt, nimmt das Passwort mit (SchreibePortalAbsender).
	bleibt := konfig.PortalAbsenderEmail != "" && konfig.PortalSMTPBenutzer != ""
	if req.Portal != nil {
		bleibt = req.Portal.AbsenderEmail != "" && req.Portal.SMTPBenutzer != ""
	}
	if len(konfig.PortalSMTPPasswortVerschluesselt) > 0 && bleibt && (req.Portal == nil || req.Portal.SMTPPasswort == "") {
		apierrors.AntworteFehlerMitMeldung(w, http.StatusBadRequest, meldungPortalPasswortZumNeuenServer, nil)
		return false
	}
	return true
}

func schreibeMailEinstellungen(ctx context.Context, mailAbfragen *repository.MailEinstellungsAbfragen, req mailEinstellungenAnfrage) error {
	if err := mailAbfragen.Schreibe(ctx, req.SMTPHost, req.SMTPPort, req.SMTPBenutzer, req.SMTPPasswort, req.AbsenderEmail); err != nil {
		return err
	}
	if req.Portal == nil {
		return nil
	}
	return mailAbfragen.SchreibePortalAbsender(ctx, req.Portal.AbsenderEmail, req.Portal.SMTPBenutzer, req.Portal.SMTPPasswort, req.Portal.Fuer)
}

// protokolliereMailEinstellungen schreibt das Speichern ins Protokoll der Verwaltung. Die
// Passwörter stehen nicht darin, nur ob eines geändert wurde.
func (t *Tueren) protokolliereMailEinstellungen(ctx context.Context, req mailEinstellungenAnfrage) {
	claims, ok := auth.ClaimsAusKontext(ctx)
	if !ok {
		return
	}
	eintrag := map[string]any{
		"smtp_host":        req.SMTPHost,
		"smtp_port":        req.SMTPPort,
		"smtp_user":        req.SMTPBenutzer,
		"sender_email":     req.AbsenderEmail,
		"password_changed": req.SMTPPasswort != "",
	}
	if req.Portal != nil {
		eintrag["portal_sender_email"] = req.Portal.AbsenderEmail
		eintrag["portal_smtp_user"] = req.Portal.SMTPBenutzer
		eintrag["portal_password_changed"] = req.Portal.SMTPPasswort != ""
		if req.Portal.Fuer != nil {
			eintrag["portal_fuer"] = req.Portal.Fuer
		}
	}
	details, err := json.Marshal(eintrag)
	if err != nil {
		slog.Error("audit: Mail-Settings-Details konnten nicht serialisiert werden", "fehler", err)
		return
	}
	t.SchreibeAdminProtokoll(ctx, claims.KontoID, "UPDATE_MAIL_SETTINGS", "", string(details))
}

// TestmailSenden sendet eine Test-E-Mail mit der aktuellen Konfiguration
func (t *Tueren) TestmailSenden() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			To string `json:"to"`
			// "portal" prüft den zweiten Absender; ohne Angabe gilt der Postausgang.
			Postfach string `json:"postfach"`
		}
		if !tuer.LeseAnfrage(w, r, &req) {
			return
		}
		postfach := mailservice.PostfachAllgemein
		switch req.Postfach {
		case "":
		case "portal":
			postfach = mailservice.PostfachPortal
		default:
			apierrors.AntworteFehlerMitMeldung(w, http.StatusBadRequest, "Unbekannter Absender für den Testversand", nil)
			return
		}

		an := strings.TrimSpace(req.To)
		if an == "" {
			apierrors.AntworteFehlerMitMeldung(w, http.StatusBadRequest, "Die E-Mail-Adresse des Empfängers fehlt", nil)
			return
		}

		// Die Adresse hier prüfen, nicht erst im Versand: Eine unbrauchbare Eingabe
		// ist ein Eingabefehler (400) und keine Störung des Servers (500) — nur so kann das
		// Formular sie von einem echten SMTP-Ausfall unterscheiden. Die Hürde in
		// SendeTestmail bleibt trotzdem stehen, sie schützt die anderen Aufrufer.
		adresse, err := mail.ParseAddress(an)
		if err != nil {
			apierrors.AntworteFehlerMitMeldung(w, http.StatusBadRequest, "Die E-Mail-Adresse des Empfängers ist ungültig", err)
			return
		}

		ctx := r.Context()
		err = mailservice.SendeTestmail(ctx, t.DB.Pool, adresse.Address, postfach)
		if err != nil {
			// mailFehlerStatus: 502, wenn der Zielserver versagt hat — nur so erreicht
			// die Diagnose (falscher Port, abgelehnte Zugangsdaten, Zertifikatsname)
			// das Formular, für das sie geschrieben wurde. Sie geht mit Absicht als Text des
			// Fehlers hinaus; der Satz von BeschreibeSMTPFehler ist für dieses Formular gebaut.
			apierrors.AntworteFehlerMitMeldung(w, mailFehlerStatus(err), err.Error(), err)
			return
		}

		tuer.AntworteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}
}

// mailFehlerStatus ordnet einem Versandfehler den HTTP-Status zu. Versagt der Zielserver, ist
// das keine Störung dieser Anwendung, und nur bei 500 ersetzt apierrors die Meldung: Als 500
// bekäme der Admin für jede SMTP-Fehlkonfiguration „Ein interner Serverfehler ist aufgetreten"
// zu lesen.
func mailFehlerStatus(err error) int {
	switch {
	case errors.Is(err, mailservice.ErrSMTPVersand):
		return http.StatusBadGateway
	case errors.Is(err, mailservice.ErrMailNichtKonfiguriert):
		// Offene Einstellung, keine Störung — dieselbe Antwort, die die Massenversände
		// schon vorher gegeben haben.
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}
