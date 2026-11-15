package system

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/mail"
	"strings"

	"bibliothek/api/tuer"
	"bibliothek/apierrors"
	"bibliothek/auth"
	"bibliothek/mailservice"
	"bibliothek/repository"
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

// mailPortalAntwort ist der Absender der Mails aus Mein Portal, ohne Passwort wie oben.
type mailPortalAntwort struct {
	AbsenderEmail string `json:"sender_email"`
	SMTPBenutzer  string `json:"smtp_user"`
	HatPasswort   bool   `json:"has_password"`
}

// mailPortalAnfrage ist der eingehende Absender der Mails aus Mein Portal. Eine leere Adresse
// heißt, sie gehen über den Postausgang; ein leerer Benutzer, mit dessen Zugang.
type mailPortalAnfrage struct {
	AbsenderEmail string `json:"sender_email"`
	SMTPBenutzer  string `json:"smtp_user"`
	SMTPPasswort  string `json:"smtp_password"` // leer: das gespeicherte Passwort bleibt
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
	// Fehlt Portal, bleibt der gespeicherte Absender für Mein Portal unangetastet: Eine Seite,
	// die vor dem Update geladen wurde, kennt das Feld nicht und löschte ihn sonst beim Speichern.
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

	if req.Portal != nil {
		req.Portal.SMTPBenutzer = strings.TrimSpace(req.Portal.SMTPBenutzer)
		if absender := strings.TrimSpace(req.Portal.AbsenderEmail); absender != "" {
			adresse, err := mail.ParseAddress(absender)
			if err != nil {
				apierrors.AntworteFehlerMitMeldung(w, http.StatusBadRequest, "Die E-Mail-Adresse des Absenders für Mein Portal ist ungültig", err)
				return req, false
			}
			req.Portal.AbsenderEmail = adresse.Address
		} else {
			req.Portal.AbsenderEmail = ""
		}
	}
	return req, true
}

// MailEinstellungenSchreiben speichert die Mail-Konfiguration
func (t *Tueren) MailEinstellungenSchreiben(mailAbfragen *repository.MailEinstellungsAbfragen) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req, ok := leseMailEinstellungen(w, r)
		if !ok {
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
// Absender für Mein Portal.
func schreibeMailEinstellungen(ctx context.Context, mailAbfragen *repository.MailEinstellungsAbfragen, req mailEinstellungenAnfrage) error {
	if err := mailAbfragen.Schreibe(ctx, req.SMTPHost, req.SMTPPort, req.SMTPBenutzer, req.SMTPPasswort, req.AbsenderEmail); err != nil {
		return err
	}
	if req.Portal == nil {
		return nil
	}
	return mailAbfragen.SchreibePortalAbsender(ctx, req.Portal.AbsenderEmail, req.Portal.SMTPBenutzer, req.Portal.SMTPPasswort)
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
			// "portal" prüft den Absender für Mein Portal; ohne Angabe gilt der Postausgang.
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
