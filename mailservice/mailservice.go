package mailservice

import (
	"bytes"
	"context"
	"fmt"
	"net/smtp"
	"os"
	"strings"
	"text/template"

	"bibliothek/db"
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

	// SMTP-Konfiguration aus ENV-Variablen
	smtpHost := os.Getenv("SMTP_HOST")
	if smtpHost == "" {
		smtpHost = "localhost"
	}
	smtpPort := os.Getenv("SMTP_PORT")
	if smtpPort == "" {
		smtpPort = "1025" // Fallback für lokales MailHog/Mailpit
	}
	smtpUser := os.Getenv("SMTP_USER")
	smtpPass := os.Getenv("SMTP_PASS")
	sender := os.Getenv("SMTP_SENDER")
	if sender == "" {
		sender = "noreply@bibliothek-schule.de"
	}

	to = strings.ReplaceAll(to, "\r", "")
	to = strings.ReplaceAll(to, "\n", "")

	// Nachricht nach RFC 822 formatieren
	// Für echte HTML-Mails muss der Content-Type auf text/html gesetzt werden
	msg := []byte(fmt.Sprintf("To: %s\r\n"+
		"Subject: %s\r\n"+
		"Content-Type: text/plain; charset=UTF-8\r\n"+
		"\r\n"+
		"%s\r\n", to, betreff, bodyBuf.String()))

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
