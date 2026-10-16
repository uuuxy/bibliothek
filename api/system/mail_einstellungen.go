package system

import (
	"encoding/json"
	"errors"
	"log"
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
	SMTPHost      string `json:"smtp_host"`
	SMTPPort      string `json:"smtp_port"`
	SMTPBenutzer  string `json:"smtp_user"`
	AbsenderEmail string `json:"sender_email"`
	HatPasswort   bool   `json:"has_password"`
}

// mailEinstellungenAnfrage ist die eingehende Mail-Konfiguration. Ein leeres SMTPPasswort
// bedeutet „unverändert lassen": Sonst würde jedes Speichern der Einstellungsseite das
// Passwort löschen, weil die Oberfläche es gar nicht kennt (siehe mailEinstellungenAntwort).
type mailEinstellungenAnfrage struct {
	SMTPHost      string `json:"smtp_host"`
	SMTPPort      string `json:"smtp_port"`
	SMTPBenutzer  string `json:"smtp_user"`
	SMTPPasswort  string `json:"smtp_password"` // leer: das gespeicherte Passwort bleibt
	AbsenderEmail string `json:"sender_email"`
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
		}

		tuer.AntworteJSON(w, http.StatusOK, antwort)
	}
}

// MailEinstellungenSchreiben speichert die Mail-Konfiguration
func (t *Tueren) MailEinstellungenSchreiben(mailAbfragen *repository.MailEinstellungsAbfragen) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req mailEinstellungenAnfrage
		if !tuer.LeseAnfrage(w, r, &req) {
			return
		}

		// Absender beim Speichern prüfen, nicht erst beim Versand: Eine Adresse mit
		// Tippfehler fiele sonst erst im Testversand auf, als 500, die der Admin als
		// Datenbankfehler läse. Leer ist erlaubt; dann setzt der Versand seinen
		// Vorgabe-Absender ein.
		if absender := strings.TrimSpace(req.AbsenderEmail); absender != "" {
			adresse, err := mail.ParseAddress(absender)
			if err != nil {
				apierrors.AntworteFehlerMitMeldung(w, http.StatusBadRequest, "Die E-Mail-Adresse des Absenders ist ungültig", err)
				return
			}
			req.AbsenderEmail = adresse.Address
		}

		ctx := r.Context()
		err := mailAbfragen.Schreibe(ctx, req.SMTPHost, req.SMTPPort, req.SMTPBenutzer, req.SMTPPasswort, req.AbsenderEmail)
		if err != nil {
			apierrors.AntworteFehler(w, http.StatusInternalServerError, err)
			return
		}

		// Protokoll der Verwaltung
		if claims, ok := auth.ClaimsAusKontext(r.Context()); ok {
			details, merr := json.Marshal(map[string]interface{}{
				"smtp_host":        req.SMTPHost,
				"smtp_port":        req.SMTPPort,
				"smtp_user":        req.SMTPBenutzer,
				"sender_email":     req.AbsenderEmail,
				"password_changed": req.SMTPPasswort != "",
			})
			if merr != nil {
				log.Printf("audit: Mail-Settings-Details konnten nicht serialisiert werden: %v", merr)
			} else {
				t.SchreibeAdminProtokoll(ctx, claims.KontoID, "UPDATE_MAIL_SETTINGS", "", string(details))
			}
		}

		tuer.AntworteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}
}

// TestmailSenden sendet eine Test-E-Mail mit der aktuellen Konfiguration
func (t *Tueren) TestmailSenden() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			To string `json:"to"`
		}
		if !tuer.LeseAnfrage(w, r, &req) {
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
		err = mailservice.SendeTestmail(ctx, t.DB.Pool, adresse.Address)
		if err != nil {
			// mailFehlerStatus: 502, wenn der Zielserver versagt hat — nur so erreicht
			// die Diagnose (falscher Port, abgelehnte Zugangsdaten, Zertifikatsname)
			// das Formular, für das sie geschrieben wurde.
			apierrors.AntworteFehler(w, mailFehlerStatus(err), err)
			return
		}

		tuer.AntworteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}
}

// mailFehlerStatus ordnet einem Versandfehler den HTTP-Status zu. Versagt der
// Zielserver, ist das keine Störung dieser Anwendung — und nur unterhalb von 500
// reicht apierrors die Meldung durch: Als 500 bekäme der Admin für jede
// SMTP-Fehlkonfiguration "Ein interner Serverfehler ist aufgetreten" zu lesen.
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
