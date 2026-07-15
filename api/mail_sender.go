package api

import (
	"bibliothek/pkg/closeutil"
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"mime/multipart"
	"net/mail"
	"net/smtp"
	"net/textproto"
	"os"
	"strings"
)

// MailAttachment represents an email attachment.
type MailAttachment struct {
	Name        string
	ContentType string
	Data        []byte
}

// MailRequest aggregates email recipient, subject, body, and attachments.
type MailRequest struct {
	To          string
	Subject     string
	Body        string
	Attachments []MailAttachment
}

// SendEmail sends a multipart email (HTML/Text) with attachments using net/smtp.
// Environment variables: SMTP_HOST, SMTP_PORT, SMTP_USER, SMTP_PASSWORD, SMTP_FROM
func SendEmail(req MailRequest) error {
	host := os.Getenv("SMTP_HOST")
	port := os.Getenv("SMTP_PORT")
	user := os.Getenv("SMTP_USER")
	pass := os.Getenv("SMTP_PASSWORD")
	from := os.Getenv("SMTP_FROM")

	if host == "" || port == "" {
		return fmt.Errorf("SMTP connection parameters missing from environment (SMTP_HOST/SMTP_PORT)")
	}
	if from == "" {
		from = user
	}

	parsedFrom, err := mail.ParseAddress(from)
	if err != nil {
		return fmt.Errorf("invalid sender email address: %w", err)
	}
	from = parsedFrom.Address

	parsedTo, err := mail.ParseAddress(req.To)
	if err != nil {
		return fmt.Errorf("invalid recipient email address: %w", err)
	}
	req.To = parsedTo.Address

	// Sanitize subject to prevent CRLF injection
	req.Subject = strings.ReplaceAll(req.Subject, "\r", "")
	req.Subject = strings.ReplaceAll(req.Subject, "\n", "")

	msg, err := baueMailNachricht(req, from)
	if err != nil {
		return err
	}

	addr := host + ":" + port
	var auth smtp.Auth
	if user != "" && pass != "" {
		auth = smtp.PlainAuth("", user, pass, host)
	}

	if err := sendMailViaSMTP(addr, host, auth, from, []string{req.To}, msg); err != nil {
		return fmt.Errorf("SMTP send failed: %w", err)
	}

	return nil
}

// baueMailNachricht erstellt die vollständige MIME-Multipart-Nachricht (Header, Textteil,
// Anhänge). req.To und req.Subject müssen bereits sanitiert sein.
func baueMailNachricht(req MailRequest, from string) ([]byte, error) {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	boundary := writer.Boundary()

	// Write SMTP Headers
	fmt.Fprintf(&buf, "From: %s\r\n", from)
	fmt.Fprintf(&buf, "To: %s\r\n", req.To)
	fmt.Fprintf(&buf, "Subject: %s\r\n", req.Subject)
	fmt.Fprintf(&buf, "MIME-Version: 1.0\r\n")
	fmt.Fprintf(&buf, "Content-Type: multipart/mixed; boundary=%s\r\n\r\n", boundary)

	// Body Part
	bodyHeader := make(textproto.MIMEHeader)
	bodyHeader.Set("Content-Type", "text/plain; charset=utf-8")
	part, err := writer.CreatePart(bodyHeader)
	if err != nil {
		return nil, fmt.Errorf("failed to create email body part: %w", err)
	}
	if _, err := part.Write([]byte(req.Body)); err != nil {
		return nil, fmt.Errorf("failed to write email body: %w", err)
	}

	// Attachments
	for _, att := range req.Attachments {
		if err := schreibeAnhang(writer, att); err != nil {
			return nil, err
		}
	}

	// Close writes the closing MIME boundary; a failure leaves the message malformed.
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("failed to finalize multipart message: %w", err)
	}

	return buf.Bytes(), nil
}

// schreibeAnhang hängt eine Datei base64-kodiert als MIME-Part an. Dateiname und
// Content-Type werden gegen Header-Injection abgesichert (CRLF bzw. Anführungszeichen
// entfernt): CreatePart schreibt Header-Werte unvalidiert, ein eingeschleustes CRLF
// würde also zusätzliche MIME-Header erzeugen.
func schreibeAnhang(writer *multipart.Writer, att MailAttachment) error {
	safeName := strings.NewReplacer("\r", "", "\n", "", `"`, "").Replace(att.Name)
	safeContentType := strings.NewReplacer("\r", "", "\n", "").Replace(att.ContentType)
	attHeader := make(textproto.MIMEHeader)
	attHeader.Set("Content-Type", safeContentType)
	attHeader.Set("Content-Transfer-Encoding", "base64")
	attHeader.Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, safeName))
	part, err := writer.CreatePart(attHeader)
	if err != nil {
		return fmt.Errorf("failed to create attachment part for %s: %w", att.Name, err)
	}

	encoder := base64.NewEncoder(base64.StdEncoding, part)
	if _, err := encoder.Write(att.Data); err != nil {
		return fmt.Errorf("failed to write attachment data for %s: %w", att.Name, err)
	}
	// Close flushes the final base64 bytes; a failure here corrupts the attachment.
	if err := encoder.Close(); err != nil {
		return fmt.Errorf("failed to finalize attachment encoding for %s: %w", att.Name, err)
	}
	return nil
}

func sendMailViaSMTP(addr, host string, a smtp.Auth, from string, to []string, msg []byte) error {
	c, err := smtp.Dial(addr)
	if err != nil {
		return err
	}
	defer closeutil.LogClose(c, "smtp client")

	if err := c.Hello("localhost"); err != nil {
		return err
	}
	if err := starttlsWennMoeglich(c, host); err != nil {
		return err
	}
	if err := smtpAuthenticate(c, a); err != nil {
		return err
	}
	return smtpSendData(c, from, to, msg)
}

// starttlsWennMoeglich führt bei Server-Unterstützung ein verifiziertes STARTTLS-Upgrade
// durch. Das Zertifikat wird gegen den konfigurierten Host VERIFIZIERT — ohne Verifikation
// könnte ein MITM beim Upgrade ein beliebiges Zertifikat vorlegen und sowohl die SMTP-AUTH-
// Zugangsdaten als auch den Mailinhalt (Schülernamen, Mahndaten, Elternadressen) mitlesen.
// Escape-Hatch nur für Legacy-Server mit Self-Signed-Zertifikat via Env.
func starttlsWennMoeglich(c *smtp.Client, host string) error {
	ok, _ := c.Extension("STARTTLS")
	if !ok {
		return nil
	}
	config := &tls.Config{
		ServerName:         host,
		MinVersion:         tls.VersionTLS12,
		InsecureSkipVerify: os.Getenv("SMTP_ALLOW_INSECURE_TLS") == "true", // #nosec G402 - bewusst per Env, Default sicher
	}
	return c.StartTLS(config)
}

// smtpAuthenticate authentifiziert sich, sofern eine Auth vorliegt und der Server sie anbietet.
func smtpAuthenticate(c *smtp.Client, a smtp.Auth) error {
	if a == nil {
		return nil
	}
	if ok, _ := c.Extension("AUTH"); ok {
		return c.Auth(a)
	}
	return nil
}

// smtpSendData überträgt den Envelope (MAIL FROM / RCPT TO) und den Nachrichtentext.
func smtpSendData(c *smtp.Client, from string, to []string, msg []byte) error {
	if err := c.Mail(from); err != nil {
		return err
	}
	for _, addr := range to {
		if err := c.Rcpt(addr); err != nil {
			return err
		}
	}
	w, err := c.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write(msg); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	return c.Quit()
}
