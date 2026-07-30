package mailservice

// smtp_konfig.go — Eine Quelle für die Frage "womit wird verschickt?".
//
// Vorher gab es zwei Antworten: Der Testversand las die in der Oberfläche
// gespeicherte Konfiguration aus der Datenbank, jeder echte Versand (Mahnungen
// einzeln und im Massenlauf, Abgänger-Kontoauszüge, Bestellungen) las die
// Umgebungsvariablen des Containers. Damit bestätigte der Test-Knopf eine
// Konfiguration, die kein Mahnlauf jemals benutzt hat — und wer im Admin-Bereich den
// SMTP-Server umstellte, änderte am Versand nichts. Beides fiel erst auf, wenn eine
// Mahnung nicht ankam.
//
// Ab hier gilt: Die gespeicherte Konfiguration ist die Wahrheit. Die Umgebung ist nur
// noch Rückfall, solange dort nichts steht — beim ersten Start übernimmt
// db.InitMailKonfig sie in die Datenbank, damit im Formular steht, was gilt.

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
)

// ErrMailNichtKonfiguriert meldet, dass kein Mailserver hinterlegt ist. Das ist kein
// Fehler des Systems, sondern eine offene Einstellung — der Aufrufer antwortet damit
// 503 statt 500 und die Meldung erreicht den Admin (apierrors neutralisiert nur 500).
var ErrMailNichtKonfiguriert = errors.New("SMTP nicht konfiguriert")

// SMTPKonfig ist die effektive Versandkonfiguration.
type SMTPKonfig struct {
	Host     string
	Port     string
	Benutzer string
	Passwort string
	Absender string
}

// Adresse liefert das host:port für net/smtp.
func (k SMTPKonfig) Adresse() string {
	return k.Host + ":" + k.Port
}

// Auth liefert die SMTP-Authentifizierung — oder nil, wenn keine Zugangsdaten
// hinterlegt sind (Relay im Schulnetz, das nach IP zulässt).
func (k SMTPKonfig) Auth() smtp.Auth {
	if k.Benutzer == "" || k.Passwort == "" {
		return nil
	}
	return smtp.PlainAuth("", k.Benutzer, k.Passwort, k.Host)
}

// IstKonfiguriert meldet, ob mit dieser Konfiguration ein Versand überhaupt versucht
// werden kann. Die Platzhalter-Erkennung stammt aus der Bestell-Abwicklung: Eine
// unausgefüllte Beispielkonfiguration soll zu "übersprungen" führen und nicht zu
// einem Verbindungsversuch gegen einen Host namens "Ihr SMTP-Host".
func (k SMTPKonfig) IstKonfiguriert() bool {
	if strings.TrimSpace(k.Host) == "" || strings.TrimSpace(k.Port) == "" {
		return false
	}
	if k.Host == "Ihr SMTP-Host" {
		return false
	}
	return !strings.Contains(k.Passwort, "Passwort") && k.Passwort != "secret"
}

// KonfigAusUmgebung liest die Konfiguration aus den Umgebungsvariablen. Sie ist der
// Rückfall für eine Datenbank ohne gespeicherte Konfiguration — und die Quelle, aus
// der db.InitMailKonfig beim ersten Start übernimmt.
func KonfigAusUmgebung() SMTPKonfig {
	k := SMTPKonfig{
		Host:     strings.TrimSpace(os.Getenv("SMTP_HOST")),
		Port:     strings.TrimSpace(os.Getenv("SMTP_PORT")),
		Benutzer: os.Getenv("SMTP_USER"),
		Passwort: os.Getenv("SMTP_PASSWORD"),
		Absender: strings.TrimSpace(os.Getenv("SMTP_FROM")),
	}
	if k.Passwort == "" {
		// docker-compose setzt beide Namen; historisch gewachsen.
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

const smtpKonfigSQL = `SELECT smtp_host, smtp_port, smtp_user, smtp_password_encrypted, sender_email
	FROM mail_settings_config WHERE id = 1`

// LadeSMTPKonfig liefert die Konfiguration, mit der tatsächlich versendet wird.
// Jeder Versender im System benutzt diese Funktion — nur so kann der Test-Knopf
// etwas über den Mahnlauf aussagen.
func LadeSMTPKonfig(ctx context.Context, dbPool db.PgxPoolIface) (SMTPKonfig, error) {
	var k SMTPKonfig
	var passwortVerschluesselt []byte

	err := dbPool.QueryRow(ctx, smtpKonfigSQL).
		Scan(&k.Host, &k.Port, &k.Benutzer, &passwortVerschluesselt, &k.Absender)
	if err != nil {
		// Keine Zeile (frische Datenbank, noch nicht gelaufene Migration): Die Umgebung
		// trägt weiter, damit ein Mahnlauf nicht an einer Konfigurationslücke scheitert.
		// Protokolliert, weil es sonst niemand merkt.
		log.Printf("mail: gespeicherte SMTP-Konfiguration nicht lesbar, Umgebung wird benutzt: %v", err)
		return KonfigAusUmgebung(), nil
	}

	if len(passwortVerschluesselt) > 0 {
		entschluesselt, err := crypto.Decrypt(passwortVerschluesselt)
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
		k.Absender = defaultFromAddress
	}
	return k, nil
}
