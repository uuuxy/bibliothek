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
	"log"
	"net/smtp"
	"os"
	"strings"

	"bibliothek/db"
	"bibliothek/internal/crypto"
	"bibliothek/repository"
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
	if k.Passwort == "" {
		// docker-compose setzt beide Namen.
		k.Passwort = os.Getenv("SMTP_PASS")
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
		log.Printf("mail: gespeicherte SMTP-Konfiguration nicht lesbar, Umgebung wird benutzt: %v", err)
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
	return k, nil
}
