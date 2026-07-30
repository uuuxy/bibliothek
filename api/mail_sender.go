package api

import (
	"bibliothek/db"
	"bibliothek/mailservice"
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/mail"
	"net/textproto"
	"strings"
	"time"
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

// SendEmail versendet eine Mail. Als überschreibbare Variable (nicht als reine
// Funktion), damit Tests den Versand simulieren können — u. a. um zu prüfen, dass ein
// SMTP-Fehler die Bestell-Transaktion zurückrollt (keine Ghost-Orders).
var SendEmail = sendEmailSMTP

// smtpKonfigLader liefert die Konfiguration für den nächsten Versand.
//
// Als Variable, damit NewServer sie an die Datenbank binden kann: Der Versand ist über
// die injizierbare Funktion SendEmail(MailRequest) erreichbar und hat weder Kontext
// noch Pool zur Hand. Vorgabe bleibt die Umgebung, damit Tests und Werkzeuge ohne
// Datenbank weiter versenden können.
var smtpKonfigLader = func() (mailservice.SMTPKonfig, error) {
	return mailservice.KonfigAusUmgebung(), nil
}

// BindeSMTPKonfigAnDatenbank lässt jeden Versand die in der Oberfläche gespeicherte
// Konfiguration benutzen. Wird beim Start einmal aufgerufen (NewServer), bevor der
// erste Request läuft.
func BindeSMTPKonfigAnDatenbank(pool db.PgxPoolIface) {
	smtpKonfigLader = func() (mailservice.SMTPKonfig, error) {
		// Eigener Kontext mit kurzer Frist: Der Aufrufer hat keinen, und ein hängender
		// Konfigurationszugriff darf einen Massenversand nicht blockieren.
		ctx, abbruch := context.WithTimeout(context.Background(), 5*time.Second)
		defer abbruch()
		return mailservice.LadeSMTPKonfig(ctx, pool)
	}
}

// smtpKonfiguriert meldet, ob ein Versand versucht werden kann. Die Handler fragen
// damit die tatsächlich benutzte Konfiguration ab statt os.Getenv("SMTP_HOST") —
// sonst melden sie "SMTP nicht konfiguriert", während in der Oberfläche ein Server
// steht (und umgekehrt).
func smtpKonfiguriert() bool {
	konfig, err := smtpKonfigLader()
	return err == nil && konfig.IstKonfiguriert()
}

// sendEmailSMTP sends a multipart email (HTML/Text) with attachments using net/smtp.
// Die Zugangsdaten kommen aus der gespeicherten Konfiguration (siehe smtpKonfigLader).
func sendEmailSMTP(req MailRequest) error {
	konfig, err := smtpKonfigLader()
	if err != nil {
		return err
	}
	if !konfig.IstKonfiguriert() {
		return fmt.Errorf("%w: kein SMTP-Server hinterlegt", mailservice.ErrMailNichtKonfiguriert)
	}

	from := konfig.Absender

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

	return mailservice.VersendeUeberSMTP(konfig, from, []string{req.To}, msg)
}

// mailFehlerStatus ordnet einem Versandfehler den HTTP-Status zu. Versagt der
// Zielserver, ist das keine Störung dieser Anwendung — und nur unterhalb von 500
// reicht apierrors die Meldung durch: Als 500 bekäme der Admin für jede
// SMTP-Fehlkonfiguration "Ein interner Datenbankfehler ist aufgetreten" zu lesen.
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
