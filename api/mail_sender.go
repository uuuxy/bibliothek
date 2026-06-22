package api

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"mime/multipart"
	"net/mail"
	"net/smtp"
	"crypto/tls"
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

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	boundary := writer.Boundary()

	parsedTo, err := mail.ParseAddress(req.To)
	if err != nil {
		return fmt.Errorf("invalid recipient email address: %w", err)
	}
	req.To = parsedTo.Address

	// Sanitize subject to prevent CRLF injection
	req.Subject = strings.ReplaceAll(req.Subject, "\r", "")
	req.Subject = strings.ReplaceAll(req.Subject, "\n", "")

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
		return fmt.Errorf("failed to create email body part: %w", err)
	}
	if _, err := part.Write([]byte(req.Body)); err != nil {
		return fmt.Errorf("failed to write email body: %w", err)
	}

	// Attachments
	for _, att := range req.Attachments {
		attHeader := make(textproto.MIMEHeader)
		attHeader.Set("Content-Type", att.ContentType)
		attHeader.Set("Content-Transfer-Encoding", "base64")
		attHeader.Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, att.Name))
		part, err := writer.CreatePart(attHeader)
		if err != nil {
			return fmt.Errorf("failed to create attachment part for %s: %w", att.Name, err)
		}

		encoder := base64.NewEncoder(base64.StdEncoding, part)
		if _, err := encoder.Write(att.Data); err != nil {
			return fmt.Errorf("failed to write attachment data for %s: %w", att.Name, err)
		}
		_ = encoder.Close()
	}

	_ = writer.Close()

	addr := host + ":" + port
	var auth smtp.Auth
	if user != "" && pass != "" {
		auth = smtp.PlainAuth("", user, pass, host)
	}

	if err := sendMailInsecure(addr, auth, from, []string{req.To}, buf.Bytes()); err != nil {
		return fmt.Errorf("SMTP send failed: %w", err)
	}

	return nil
}

func sendMailInsecure(addr string, a smtp.Auth, from string, to []string, msg []byte) error {
	c, err := smtp.Dial(addr)
	if err != nil {
		return err
	}
	defer c.Close()
	if err = c.Hello("localhost"); err != nil {
		return err
	}
	if ok, _ := c.Extension("STARTTLS"); ok {
		config := &tls.Config{InsecureSkipVerify: true}
		if err = c.StartTLS(config); err != nil {
			return err
		}
	}
	if a != nil {
		if ok, _ := c.Extension("AUTH"); ok {
			if err = c.Auth(a); err != nil {
				return err
			}
		}
	}
	if err = c.Mail(from); err != nil {
		return err
	}
	for _, addr := range to {
		if err = c.Rcpt(addr); err != nil {
			return err
		}
	}
	w, err := c.Data()
	if err != nil {
		return err
	}
	_, err = w.Write(msg)
	if err != nil {
		return err
	}
	err = w.Close()
	if err != nil {
		return err
	}
	return c.Quit()
}
