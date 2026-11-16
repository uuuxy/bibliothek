package mailservice

// Eine Quelle für die Frage „womit wird verschickt?". Gäbe es zwei, bestätigte der Knopf für den
// Testversand eine Konfiguration, die kein Mahnlauf benutzt, und wer den SMTP-Server umstellte,
// änderte am Versand nichts; das fiele erst auf, wenn eine Mahnung nicht ankommt.
//
// Die gespeicherte Konfiguration gilt. Die Umgebung ist nur Rückfall, solange dort nichts steht;
// beim ersten Start übernimmt db.UebernimmMailKonfig sie in die Datenbank, damit im Formular
// steht, was gilt.

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/smtp"
	"os"
	"strings"

	"bibliothek/internal/crypto"
	"bibliothek/internal/db"
	"bibliothek/internal/repository"
)

// ErrMailNichtKonfiguriert meldet, dass kein Mailserver hinterlegt ist. Das ist kein
// Fehler des Systems, sondern eine offene Einstellung: Der Aufrufer antwortet damit 503 statt
// 500, und die Meldung erreicht den Admin (apierrors.AntworteFehler neutralisiert nur 500).
var ErrMailNichtKonfiguriert = errors.New("SMTP nicht konfiguriert")

// SMTPKonfig ist die Konfiguration, mit der verschickt wird.
type SMTPKonfig struct {
	Host     string
	Port     string
	Benutzer string
	Passwort string
	Absender string
	// portal ersetzt für PostfachPortal Absender und Zugang; nil heißt, es gibt keinen eigenen.
	portal *portalAbsender
	// portalFuer sind die Arten von Mails, die über den zweiten Absender gehen.
	portalFuer map[Mailart]bool
}

// portalAbsender ist der zweite Absender, wie die Einstellungen ihn nennen.
type portalAbsender struct {
	absender string
	// benutzer und passwort sind leer, wenn der Zugang des Postausgangs auch für diese Adresse gilt.
	benutzer string
	passwort string
	// fehler hält ein Passwort fest, das sich nicht entschlüsseln ließ. Es hält nur die Mails
	// dieses Absenders auf; die übrigen gehen weiter über den Postausgang.
	fehler error
}

// postfachFuer sagt, über welchen Absender eine Mail dieser Art hinausgeht.
func (k SMTPKonfig) postfachFuer(art Mailart) Postfach {
	if k.portalFuer[art] {
		return PostfachPortal
	}
	return PostfachAllgemein
}

// fuer liefert die Konfiguration, mit der eine Mail des Postfachs hinausgeht. Server und Port
// gelten für beide Absender.
func (k SMTPKonfig) fuer(postfach Postfach) (SMTPKonfig, error) {
	if postfach != PostfachPortal || k.portal == nil {
		return k, nil
	}
	if k.portal.fehler != nil {
		return SMTPKonfig{}, k.portal.fehler
	}
	p := k
	p.portal = nil
	p.Absender = k.portal.absender
	if k.portal.benutzer != "" {
		p.Benutzer = k.portal.benutzer
		p.Passwort = k.portal.passwort
	}
	return p, nil
}

// adresse liefert das host:port für net/smtp.
func (k SMTPKonfig) adresse() string {
	return k.Host + ":" + k.Port
}

// auth liefert die Anmeldung am SMTP-Server oder nil, wenn keine Zugangsdaten
// hinterlegt sind (Relay im Schulnetz, das nach IP zulässt).
func (k SMTPKonfig) auth() smtp.Auth {
	if k.Benutzer == "" || k.Passwort == "" {
		return nil
	}
	return smtp.PlainAuth("", k.Benutzer, k.Passwort, k.Host)
}

// istKonfiguriert meldet, ob mit dieser Konfiguration ein Versand überhaupt versucht
// werden kann. Eine nicht ausgefüllte Beispielkonfiguration führt zu „übersprungen" und nicht
// zu einem Versuch, einen Host namens "Ihr SMTP-Host" zu erreichen.
func (k SMTPKonfig) istKonfiguriert() bool {
	if strings.TrimSpace(k.Host) == "" || strings.TrimSpace(k.Port) == "" {
		return false
	}
	if k.Host == "Ihr SMTP-Host" {
		return false
	}
	return !strings.Contains(k.Passwort, "Passwort") && k.Passwort != "secret"
}

// KonfigAusUmgebung liest die Konfiguration aus den Umgebungsvariablen. Sie ist der
// Rückfall für eine Datenbank ohne gespeicherte Konfiguration und die Quelle, aus
// der db.UebernimmMailKonfig beim ersten Start übernimmt.
func KonfigAusUmgebung() SMTPKonfig {
	k := SMTPKonfig{
		Host:     strings.TrimSpace(os.Getenv("SMTP_HOST")),
		Port:     strings.TrimSpace(os.Getenv("SMTP_PORT")),
		Benutzer: os.Getenv("SMTP_USER"),
		Passwort: os.Getenv("SMTP_PASSWORD"),
		Absender: strings.TrimSpace(os.Getenv("SMTP_FROM")),
	}
	if k.Host != "" && k.Port == "" {
		k.Port = "587"
	}
	if k.Absender == "" {
		k.Absender = k.Benutzer
	}
	return k
}

// ladeSMTPKonfig liefert die Konfiguration, mit der tatsächlich versendet wird.
// Jeder Versand benutzt diese Funktion; nur so sagt der Knopf für den Testversand etwas über den
// Mahnlauf. Die gespeicherte Zeile liest sie über dieselbe Funktion
// wie die Maske der Mail-Einstellungen (repository.MailEinstellungsAbfragen.Lade).
func ladeSMTPKonfig(ctx context.Context, pool db.Pool) (SMTPKonfig, error) {
	gespeichert, err := repository.NewMailEinstellungsAbfragen(pool).Lade(ctx)
	if err != nil {
		// Keine Zeile (frische Datenbank, noch nicht gelaufene Migration): Die Umgebung
		// trägt weiter, damit ein Mahnlauf nicht an einer Konfigurationslücke scheitert.
		// Protokolliert, weil es sonst niemand merkt.
		slog.Error("mail: gespeicherte SMTP-Konfiguration nicht lesbar, Umgebung wird benutzt", "fehler", err)
		return KonfigAusUmgebung(), nil
	}
	k := SMTPKonfig{
		Host:     gespeichert.SMTPHost,
		Port:     gespeichert.SMTPPort,
		Benutzer: gespeichert.SMTPBenutzer,
		Absender: gespeichert.AbsenderEmail,
	}

	if len(gespeichert.SMTPPasswortVerschluesselt) > 0 {
		entschluesselt, err := crypto.Entschluessele(gespeichert.SMTPPasswortVerschluesselt)
		if err != nil {
			// Nicht auf die Umgebung ausweichen: Ein unlesbares Passwort heißt, dass der
			// APP_ENCRYPTION_KEY nicht mehr derselbe ist. Das muss auffallen, statt still
			// mit einer anderen Konfiguration weiterzulaufen.
			return SMTPKonfig{}, fmt.Errorf("fehler beim Entschlüsseln des SMTP-Passworts: %w", err)
		}
		k.Passwort = string(entschluesselt)
	}

	k.Host = strings.TrimSpace(k.Host)
	if k.Host == "" {
		return KonfigAusUmgebung(), nil
	}
	if strings.TrimSpace(k.Port) == "" {
		k.Port = "587"
	}
	if strings.TrimSpace(k.Absender) == "" {
		k.Absender = absenderVorgabe
	}
	k.portal = lesePortalAbsender(gespeichert)
	k.portalFuer = map[Mailart]bool{}
	for _, art := range gespeichert.PortalAbsenderFuer {
		k.portalFuer[Mailart(art)] = true
	}
	return k, nil
}

// lesePortalAbsender liest den zweiten Absender aus der gespeicherten Zeile. Ohne Adresse gibt
// es keinen, und alle Mails gehen über den Postausgang.
func lesePortalAbsender(gespeichert *repository.MailEinstellungen) *portalAbsender {
	adresse := strings.TrimSpace(gespeichert.PortalAbsenderEmail)
	if adresse == "" {
		return nil
	}
	p := &portalAbsender{absender: adresse, benutzer: strings.TrimSpace(gespeichert.PortalSMTPBenutzer)}
	if p.benutzer == "" || len(gespeichert.PortalSMTPPasswortVerschluesselt) == 0 {
		return p
	}
	entschluesselt, err := crypto.Entschluessele(gespeichert.PortalSMTPPasswortVerschluesselt)
	if err != nil {
		p.fehler = fmt.Errorf("fehler beim Entschlüsseln des SMTP-Passworts des zweiten Absenders: %w", err)
		return p
	}
	p.passwort = string(entschluesselt)
	return p
}
