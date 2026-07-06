package inventur

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/smtp"
	"net/textproto"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// EmailMessage kapselt die Daten einer zu versendenden E-Mail.
type EmailMessage struct {
	From       string
	To         string
	Subject    string
	Body       string
	AttachName string
	AttachData []byte
}

// BackupEmailConfig enthält die SMTP-Konfiguration für Backup-E-Mails.
type BackupEmailConfig struct {
	Host     string // z.B. "smtp.gmail.com" oder "mail.schule.de"
	Port     string // z.B. "587"
	User     string // z.B. "backup@schule.de"
	Password string // App-Passwort bei Gmail
	To       string // Empfänger-Adresse
}

// NewBackupEmailConfigFromEnv liest die SMTP-Konfiguration aus Umgebungsvariablen.
// Gibt nil zurück, wenn die Konfiguration nicht gesetzt ist (E-Mail wird dann übersprungen).
func NewBackupEmailConfigFromEnv() *BackupEmailConfig {
	host := os.Getenv("SMTP_HOST")
	if host == "" {
		return nil
	}

	port := os.Getenv("SMTP_PORT")
	if port == "" {
		port = "587"
	}

	return &BackupEmailConfig{
		Host:     host,
		Port:     port,
		User:     os.Getenv("SMTP_USER"),
		Password: os.Getenv("SMTP_PASSWORD"),
		To:       os.Getenv("BACKUP_EMAIL_TO"),
	}
}

// SendBackupEmail komprimiert den Backup-Ordner als ZIP und verschickt ihn per E-Mail.
func SendBackupEmail(config *BackupEmailConfig, backupPath string) error {
	if config == nil || config.To == "" {
		return nil // E-Mail nicht konfiguriert, still überspringen
	}

	backupName := filepath.Base(backupPath)

	// 1. Backup-Ordner als ZIP komprimieren
	zipData, err := createZip(backupPath)
	if err != nil {
		return fmt.Errorf("ZIP-Erstellung fehlgeschlagen: %w", err)
	}

	log.Printf("Backup-E-Mail: ZIP erstellt (%d KB)", len(zipData)/1024)

	// 2. E-Mail mit Anhang erstellen
	subject := fmt.Sprintf("Schulbuch-Inventar Backup %s", backupName)
	body := fmt.Sprintf(
		"Automatisches Backup vom %s\n\nDieses Backup wurde automatisch erstellt.\nGröße: %d KB\n\nDiese E-Mail wurde automatisch generiert.",
		time.Now().Format("02.01.2006 15:04"),
		len(zipData)/1024,
	)

	emailMsg := EmailMessage{
		From:       config.User,
		To:         config.To,
		Subject:    subject,
		Body:       body,
		AttachName: fmt.Sprintf("backup_%s.zip", backupName),
		AttachData: zipData,
	}

	msg, err := buildEmailWithAttachment(emailMsg)
	if err != nil {
		return fmt.Errorf("E-Mail-Erstellung fehlgeschlagen: %w", err)
	}

	// 3. Über SMTP versenden
	auth := smtp.PlainAuth("", config.User, config.Password, config.Host)
	addr := config.Host + ":" + config.Port

	if err := smtp.SendMail(addr, auth, config.User, []string{config.To}, msg); err != nil {
		return fmt.Errorf("E-Mail-Versand fehlgeschlagen: %w", err)
	}

	log.Printf("Backup-E-Mail erfolgreich an %s gesendet", config.To)
	return nil
}

// createZip komprimiert einen Ordner rekursiv in ein ZIP-Archiv im Speicher.
func createZip(srcDir string) ([]byte, error) {
	var buf bytes.Buffer
	zipWriter := zip.NewWriter(&buf)

	root, err := os.OpenRoot(srcDir)
	if err != nil {
		return nil, err
	}
	defer func() { _ = root.Close() }()

	err = filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Relativen Pfad berechnen
		relPath, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}

		if info.IsDir() {
			// Ordner-Einträge brauchen einen trailing slash
			if relPath != "." {
				_, err := zipWriter.Create(relPath + "/")
				return err
			}
			return nil
		}

		// Datei zum ZIP hinzufügen
		writer, err := zipWriter.Create(relPath)
		if err != nil {
			return err
		}

		// Safe file open within root
		file, err := root.Open(relPath)
		if err != nil {
			return err
		}
		defer func() { _ = file.Close() }()

		_, err = io.Copy(writer, file)
		return err
	})

	if err != nil {
		return nil, err
	}

	if err := zipWriter.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// buildEmailWithAttachment erstellt eine MIME-Multipart-E-Mail mit Anhang.
func buildEmailWithAttachment(msg EmailMessage) ([]byte, error) {
	var buf bytes.Buffer

	writer := multipart.NewWriter(&buf)
	boundary := writer.Boundary()

	// E-Mail-Header
	headers := []string{
		fmt.Sprintf("From: %s", msg.From),
		fmt.Sprintf("To: %s", msg.To),
		fmt.Sprintf("Subject: %s", msg.Subject),
		"MIME-Version: 1.0",
		fmt.Sprintf("Content-Type: multipart/mixed; boundary=\"%s\"", boundary),
		"",
		"",
	}
	buf.Reset()
	buf.WriteString(strings.Join(headers, "\r\n"))

	// Neu: multipart writer mit gleichem boundary
	writer2 := multipart.NewWriter(&buf)
	_ = writer2.SetBoundary(boundary)

	// Text-Teil
	textHeader := make(textproto.MIMEHeader)
	textHeader.Set("Content-Type", "text/plain; charset=utf-8")
	textPart, err := writer2.CreatePart(textHeader)
	if err != nil {
		return nil, err
	}
	_, _ = textPart.Write([]byte(msg.Body))

	// ZIP-Anhang
	attachHeader := make(textproto.MIMEHeader)
	attachHeader.Set("Content-Type", "application/zip")
	attachHeader.Set("Content-Transfer-Encoding", "base64")
	attachHeader.Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", msg.AttachName))
	attachPart, err := writer2.CreatePart(attachHeader)
	if err != nil {
		return nil, err
	}

	// Base64-kodiert schreiben (76 Zeichen pro Zeile)
	encoded := base64.StdEncoding.EncodeToString(msg.AttachData)
	for i := 0; i < len(encoded); i += 76 {
		end := i + 76
		if end > len(encoded) {
			end = len(encoded)
		}
		_, _ = attachPart.Write([]byte(encoded[i:end] + "\r\n"))
	}

	_ = writer2.Close()

	return buf.Bytes(), nil
}
