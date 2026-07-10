package mailservice

import (
	"bytes"
	"context"
	"fmt"
	"net/mail"
	"net/smtp"
	"strings"
	"text/template"

	"bibliothek/db"
	"bibliothek/internal/crypto"
)

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

	// SMTP-Konfiguration aus der Datenbank laden
	var smtpHost, smtpPort, smtpUser, sender string
	var smtpPassEncrypted []byte

	err = dbPool.QueryRow(ctx, "SELECT smtp_host, smtp_port, smtp_user, smtp_password_encrypted, sender_email FROM mail_settings_config WHERE id = 1").
		Scan(&smtpHost, &smtpPort, &smtpUser, &smtpPassEncrypted, &sender)

	if err != nil {
		// Fallback, falls die Tabelle leer ist oder noch nicht migriert wurde
		smtpHost = "localhost"
		smtpPort = "1025"
		sender = "noreply@bibliothek-schule.de"
	}

	var smtpPass string
	if len(smtpPassEncrypted) > 0 {
		decrypted, err := crypto.Decrypt(smtpPassEncrypted)
		if err != nil {
			return fmt.Errorf("fehler beim Entschlüsseln des SMTP-Passworts: %w", err)
		}
		smtpPass = string(decrypted)
	}

	if smtpHost == "" {
		smtpHost = "localhost"
	}
	if smtpPort == "" {
		smtpPort = "1025"
	}
	if sender == "" {
		sender = "noreply@bibliothek-schule.de"
	}

	// validate sender
	parsedSender, err := mail.ParseAddress(sender)
	if err != nil {
		return fmt.Errorf("ungültige Absender-E-Mail-Adresse: %w", err)
	}
	sender = parsedSender.Address

	// validate recipient
	parsedTo, err := mail.ParseAddress(to)
	if err != nil {
		return fmt.Errorf("ungültige E-Mail-Adresse: %w", err)
	}
	to = parsedTo.Address

	betreff = strings.ReplaceAll(betreff, "\r", "")
	betreff = strings.ReplaceAll(betreff, "\n", "")

	// Nachricht nach RFC 822 formatieren
	// Für echte HTML-Mails muss der Content-Type auf text/html gesetzt werden
	msg := []byte(fmt.Sprintf("From: %s\r\n"+
		"To: %s\r\n"+
		"Subject: %s\r\n"+
		"Content-Type: text/plain; charset=UTF-8\r\n"+
		"\r\n"+
		"%s\r\n", sender, to, betreff, bodyBuf.String()))

	// SMTP-Verbindung aufbauen und E-Mail versenden
	addr := fmt.Sprintf("%s:%s", smtpHost, smtpPort)

	var auth smtp.Auth
	if smtpUser != "" && smtpPass != "" {
		auth = smtp.PlainAuth("", smtpUser, smtpPass, smtpHost)
	}

	err = smtp.SendMail(addr, auth, sender, []string{to}, msg)
	if err != nil {
		return fmt.Errorf("fehler beim SMTP-Versand (Server unter %s erreichbar?): %w", addr, err)
	}

	return nil
}

// SendTestMail versendet eine einfache Testnachricht, um die SMTP-Konfiguration zu validieren.
func SendTestMail(ctx context.Context, dbPool db.PgxPoolIface, to string) error {
	// SMTP-Konfiguration aus der Datenbank laden
	var smtpHost, smtpPort, smtpUser, sender string
	var smtpPassEncrypted []byte

	err := dbPool.QueryRow(ctx, "SELECT smtp_host, smtp_port, smtp_user, smtp_password_encrypted, sender_email FROM mail_settings_config WHERE id = 1").
		Scan(&smtpHost, &smtpPort, &smtpUser, &smtpPassEncrypted, &sender)

	if err != nil {
		return fmt.Errorf("mail-konfiguration nicht gefunden: %w", err)
	}

	var smtpPass string
	if len(smtpPassEncrypted) > 0 {
		decrypted, err := crypto.Decrypt(smtpPassEncrypted)
		if err != nil {
			return fmt.Errorf("fehler beim Entschlüsseln des SMTP-Passworts: %w", err)
		}
		smtpPass = string(decrypted)
	}

	if smtpHost == "" {
		smtpHost = "localhost"
	}
	if smtpPort == "" {
		smtpPort = "1025"
	}
	if sender == "" {
		sender = "noreply@bibliothek-schule.de"
	}

	parsedSender, err := mail.ParseAddress(sender)
	if err != nil {
		return fmt.Errorf("ungültige Absender-E-Mail-Adresse: %w", err)
	}
	sender = parsedSender.Address

	parsedTo, err := mail.ParseAddress(to)
	if err != nil {
		return fmt.Errorf("ungültige Empfänger-E-Mail-Adresse: %w", err)
	}
	to = parsedTo.Address

	betreff := "Test-E-Mail der Schulbibliothek"
	bodyText := "Dies ist eine automatisch generierte Test-E-Mail zur Überprüfung der SMTP-Konfiguration."

	msg := []byte(fmt.Sprintf("From: %s\r\n"+
		"To: %s\r\n"+
		"Subject: %s\r\n"+
		"Content-Type: text/plain; charset=UTF-8\r\n"+
		"\r\n"+
		"%s\r\n", sender, to, betreff, bodyText))

	addr := fmt.Sprintf("%s:%s", smtpHost, smtpPort)

	var smtpAuth smtp.Auth
	if smtpUser != "" && smtpPass != "" {
		smtpAuth = smtp.PlainAuth("", smtpUser, smtpPass, smtpHost)
	}

	err = smtp.SendMail(addr, smtpAuth, sender, []string{to}, msg)
	if err != nil {
		return fmt.Errorf("fehler beim SMTP-Versand (Server unter %s erreichbar?): %w", addr, err)
	}

	return nil
}
