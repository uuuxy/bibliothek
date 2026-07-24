package mailservice

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"net/mail"
	"net/smtp"
	"strings"
	"text/template"

	"bibliothek/db"
	"bibliothek/internal/crypto"
)

type smtpConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Sender   string
}

func loadSMTPConfig(ctx context.Context, dbPool db.PgxPoolIface, allowFallback bool) (*smtpConfig, error) {
	var host, port, user, sender string
	var passEncrypted []byte

	err := dbPool.QueryRow(ctx, "SELECT smtp_host, smtp_port, smtp_user, smtp_password_encrypted, sender_email FROM mail_settings_config WHERE id = 1").
		Scan(&host, &port, &user, &passEncrypted, &sender)

	if err != nil {
		if allowFallback {
			// Fallback, falls die Tabelle leer ist oder noch nicht migriert wurde
			host = "localhost"
			port = "1025"
			sender = defaultFromAddress
		} else {
			return nil, fmt.Errorf("mail-konfiguration nicht gefunden: %w", err)
		}
	}

	var pass string
	if len(passEncrypted) > 0 {
		decrypted, err := crypto.Decrypt(passEncrypted)
		if err != nil {
			return nil, fmt.Errorf("fehler beim Entschlüsseln des SMTP-Passworts: %w", err)
		}
		pass = string(decrypted)
	}

	if host == "" {
		host = "localhost"
	}
	if port == "" {
		port = "1025"
	}
	if sender == "" {
		sender = defaultFromAddress
	}

	parsedSender, err := mail.ParseAddress(sender)
	if err != nil {
		return nil, fmt.Errorf("ungültige Absender-E-Mail-Adresse: %w", err)
	}
	sender = parsedSender.Address

	return &smtpConfig{
		Host:     host,
		Port:     port,
		User:     user,
		Password: pass,
		Sender:   sender,
	}, nil
}

func (c *smtpConfig) sendMail(to, betreff, bodyText string) error {
	parsedTo, err := mail.ParseAddress(to)
	if err != nil {
		return fmt.Errorf("ungültige Empfänger-E-Mail-Adresse: %w", err)
	}
	to = parsedTo.Address

	// Ensure no newlines in headers
	betreff = strings.ReplaceAll(betreff, "\r", "")
	betreff = strings.ReplaceAll(betreff, "\n", "")
	safeSender := strings.ReplaceAll(strings.ReplaceAll(c.Sender, "\r", ""), "\n", "")
	safeTo := strings.ReplaceAll(strings.ReplaceAll(to, "\r", ""), "\n", "")

	// Fix CodeQL alert: Email content may contain untrusted input
	// By encoding the body text in Base64, we prevent any chance of CRLF injection into the SMTP session.
	encodedBody := base64.StdEncoding.EncodeToString([]byte(bodyText))

	var b bytes.Buffer
	b.WriteString(fmt.Sprintf("From: %s\r\n", safeSender))
	b.WriteString(fmt.Sprintf("To: %s\r\n", safeTo))
	b.WriteString(fmt.Sprintf("Subject: %s\r\n", betreff))
	b.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	b.WriteString("Content-Transfer-Encoding: base64\r\n")
	b.WriteString("\r\n")

	// Write encoded body chunked into lines (optional, but good practice for base64 in emails)
	// For short emails, appending the whole encoded string is also okay. Let's just append it.
	b.WriteString(encodedBody)
	b.WriteString("\r\n")

	msg := b.Bytes()

	addr := fmt.Sprintf("%s:%s", c.Host, c.Port)

	var auth smtp.Auth
	if c.User != "" && c.Password != "" {
		auth = smtp.PlainAuth("", c.User, c.Password, c.Host)
	}

	err = smtp.SendMail(addr, auth, c.Sender, []string{to}, msg)
	if err != nil {
		return fmt.Errorf("fehler beim SMTP-Versand (Server unter %s erreichbar?): %w", addr, err)
	}

	return nil
}

// SendTemplateMail lädt eine Vorlage aus der Datenbank, ersetzt Platzhalter (z.B. {{.Name}}) und versendet die E-Mail.
func SendTemplateMail(ctx context.Context, dbPool db.PgxPoolIface, to string, templateType string, data map[string]interface{}) error {
	var betreff, textBody string

	// Vorlage aus der DB laden
	err := dbPool.QueryRow(ctx, "SELECT betreff, text_body FROM mail_vorlagen WHERE typ = $1", templateType).Scan(&betreff, &textBody)
	if err != nil {
		return fmt.Errorf("vorlage '%s' nicht gefunden oder Fehler beim Laden: %w", templateType, err)
	}

	// Template parsen
	tmpl, err := template.New("mail_body").Parse(textBody)
	if err != nil {
		return fmt.Errorf("fehler beim parsen des Vorlagentextes: %w", err)
	}

	// Daten in das Template einsetzen
	var bodyBuf bytes.Buffer
	if err := tmpl.Execute(&bodyBuf, data); err != nil {
		return fmt.Errorf("fehler beim anwenden der Daten auf Vorlage: %w", err)
	}

	cfg, err := loadSMTPConfig(ctx, dbPool, true)
	if err != nil {
		return err
	}

	return cfg.sendMail(to, betreff, bodyBuf.String())
}

// SendTestMail versendet eine einfache Testnachricht, um die SMTP-Konfiguration zu validieren.
func SendTestMail(ctx context.Context, dbPool db.PgxPoolIface, to string) error {
	cfg, err := loadSMTPConfig(ctx, dbPool, false)
	if err != nil {
		return err
	}

	betreff := "Test-E-Mail der Schulbibliothek"
	bodyText := "Dies ist eine automatisch generierte Test-E-Mail zur Überprüfung der SMTP-Konfiguration."

	return cfg.sendMail(to, betreff, bodyText)
}
