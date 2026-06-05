package inventur

import (
	"archive/zip"
	"bytes"
	"errors"
	"net/smtp"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewBackupEmailConfigFromEnv(t *testing.T) {
	// Sichern und Wiederherstellen der ursprünglichen Umgebungsvariablen
	oldHost := os.Getenv("SMTP_HOST")
	oldPort := os.Getenv("SMTP_PORT")
	oldUser := os.Getenv("SMTP_USER")
	oldPass := os.Getenv("SMTP_PASSWORD")
	oldTo := os.Getenv("BACKUP_EMAIL_TO")

	defer func() {
		os.Setenv("SMTP_HOST", oldHost)
		os.Setenv("SMTP_PORT", oldPort)
		os.Setenv("SMTP_USER", oldUser)
		os.Setenv("SMTP_PASSWORD", oldPass)
		os.Setenv("BACKUP_EMAIL_TO", oldTo)
	}()

	t.Run("NoHost", func(t *testing.T) {
		os.Setenv("SMTP_HOST", "")
		config := NewBackupEmailConfigFromEnv()
		if config != nil {
			t.Errorf("Erwartete nil-Konfiguration wenn SMTP_HOST leer ist, bekam %v", config)
		}
	})

	t.Run("DefaultPort", func(t *testing.T) {
		os.Setenv("SMTP_HOST", "smtp.example.com")
		os.Setenv("SMTP_PORT", "")
		os.Setenv("SMTP_USER", "user")
		os.Setenv("SMTP_PASSWORD", "pass")
		os.Setenv("BACKUP_EMAIL_TO", "to@example.com")

		config := NewBackupEmailConfigFromEnv()
		if config == nil {
			t.Fatal("Erwartete gültige Konfiguration, bekam nil")
		}
		if config.Port != "587" {
			t.Errorf("Erwarteter Default-Port 587, bekam %s", config.Port)
		}
	})

	t.Run("ValidConfig", func(t *testing.T) {
		os.Setenv("SMTP_HOST", "smtp.example.com")
		os.Setenv("SMTP_PORT", "465")
		os.Setenv("SMTP_USER", "user")
		os.Setenv("SMTP_PASSWORD", "pass")
		os.Setenv("BACKUP_EMAIL_TO", "to@example.com")

		config := NewBackupEmailConfigFromEnv()
		if config == nil {
			t.Fatal("Erwartete gültige Konfiguration, bekam nil")
		}
		if config.Host != "smtp.example.com" || config.Port != "465" ||
			config.User != "user" || config.Password != "pass" || config.To != "to@example.com" {
			t.Errorf("Unerwartete Konfigurationswerte: %+v", config)
		}
	})
}

func TestCreateZip(t *testing.T) {
	// Temporäres Verzeichnis für den Test erstellen
	tempDir := t.TempDir()

	// Eine Dummy-Datei im temporären Verzeichnis erstellen
	filePath := filepath.Join(tempDir, "test.txt")
	err := os.WriteFile(filePath, []byte("Dies ist ein Test"), 0644)
	if err != nil {
		t.Fatalf("Konnte Testdatei nicht erstellen: %v", err)
	}

	// Noch ein Unterverzeichnis mit einer Datei
	subDir := filepath.Join(tempDir, "subdir")
	err = os.Mkdir(subDir, 0755)
	if err != nil {
		t.Fatalf("Konnte Unterverzeichnis nicht erstellen: %v", err)
	}
	subFilePath := filepath.Join(subDir, "subtest.txt")
	err = os.WriteFile(subFilePath, []byte("Unterverzeichnis Test"), 0644)
	if err != nil {
		t.Fatalf("Konnte Datei im Unterverzeichnis nicht erstellen: %v", err)
	}

	// createZip aufrufen
	zipData, err := createZip(tempDir)
	if err != nil {
		t.Fatalf("createZip gab einen Fehler zurück: %v", err)
	}
	if len(zipData) == 0 {
		t.Fatal("createZip gab ein leeres Byte-Slice zurück")
	}

	// Prüfen, ob wir das ZIP entpacken und die Dateien finden können
	reader, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	if err != nil {
		t.Fatalf("Konnte generiertes ZIP nicht lesen: %v", err)
	}

	foundTestTxt := false
	foundSubtestTxt := false

	for _, file := range reader.File {
		if file.Name == "test.txt" {
			foundTestTxt = true
		} else if file.Name == "subdir/subtest.txt" {
			foundSubtestTxt = true
		}
	}

	if !foundTestTxt {
		t.Error("test.txt im ZIP-Archiv nicht gefunden")
	}
	if !foundSubtestTxt {
		t.Error("subdir/subtest.txt im ZIP-Archiv nicht gefunden")
	}
}

func TestBuildEmailWithAttachment(t *testing.T) {
	msg := EmailMessage{
		From:       "from@example.com",
		To:         "to@example.com",
		Subject:    "Test Subject",
		Body:       "Test Body",
		AttachName: "test.zip",
		AttachData: []byte("dummy zip content"),
	}

	emailData, err := buildEmailWithAttachment(msg)
	if err != nil {
		t.Fatalf("buildEmailWithAttachment gab einen Fehler zurück: %v", err)
	}

	emailStr := string(emailData)

	// Überprüfe wichtige Header
	if !strings.Contains(emailStr, "From: from@example.com") {
		t.Error("From Header fehlt oder inkorrekt")
	}
	if !strings.Contains(emailStr, "To: to@example.com") {
		t.Error("To Header fehlt oder inkorrekt")
	}
	if !strings.Contains(emailStr, "Subject: Test Subject") {
		t.Error("Subject Header fehlt oder inkorrekt")
	}
	if !strings.Contains(emailStr, "Content-Type: multipart/mixed; boundary=") {
		t.Error("Content-Type multipart/mixed fehlt oder inkorrekt")
	}

	// Überprüfe Body und Anhang
	if !strings.Contains(emailStr, "Test Body") {
		t.Error("Body-Text fehlt im E-Mail-String")
	}
	if !strings.Contains(emailStr, "filename=\"test.zip\"") {
		t.Error("Anhang-Dateiname fehlt im E-Mail-String")
	}

	// Optional: Könnte man auch detaillierter per MIME-Parsing überprüfen
	// Aber für einen Unit-Test der String-Bastelei ist Contains oft ausreichend.
}

func TestSendBackupEmail(t *testing.T) {
	// Original-Funktion sichern und wiederherstellen
	origSendMail := sendMail
	defer func() { sendMail = origSendMail }()

	var sendMailCalled bool
	var capturedAddr, capturedFrom string
	var capturedTo []string
	var capturedMsg []byte

	// Mock für sendMail
	sendMailMock := func(addr string, a smtp.Auth, from string, to []string, msg []byte) error {
		sendMailCalled = true
		capturedAddr = addr
		capturedFrom = from
		capturedTo = to
		capturedMsg = msg
		return nil
	}

	t.Run("NilConfig", func(t *testing.T) {
		err := SendBackupEmail(nil, "dummy")
		if err != nil {
			t.Errorf("Erwartete keinen Fehler bei nil Config, bekam %v", err)
		}
	})

	t.Run("EmptyTo", func(t *testing.T) {
		config := &BackupEmailConfig{To: ""}
		err := SendBackupEmail(config, "dummy")
		if err != nil {
			t.Errorf("Erwartete keinen Fehler bei leerer To-Adresse, bekam %v", err)
		}
	})

	t.Run("InvalidBackupPath", func(t *testing.T) {
		config := &BackupEmailConfig{To: "to@example.com"}
		err := SendBackupEmail(config, "/path/that/does/not/exist/hopefully")
		if err == nil {
			t.Error("Erwartete Fehler bei ungültigem Pfad, bekam keinen")
		} else if !strings.Contains(err.Error(), "ZIP-Erstellung fehlgeschlagen") {
			t.Errorf("Erwartete 'ZIP-Erstellung fehlgeschlagen', bekam: %v", err)
		}
	})

	t.Run("Success", func(t *testing.T) {
		sendMail = sendMailMock
		sendMailCalled = false

		tempDir := t.TempDir()
		os.WriteFile(filepath.Join(tempDir, "test.txt"), []byte("test"), 0644)

		config := &BackupEmailConfig{
			Host:     "smtp.example.com",
			Port:     "587",
			User:     "user@example.com",
			Password: "password",
			To:       "to@example.com",
		}

		err := SendBackupEmail(config, tempDir)
		if err != nil {
			t.Fatalf("Unerwarteter Fehler: %v", err)
		}

		if !sendMailCalled {
			t.Error("sendMail wurde nicht aufgerufen")
		}

		if capturedAddr != "smtp.example.com:587" {
			t.Errorf("Falscher Server: %s", capturedAddr)
		}
		if capturedFrom != "user@example.com" {
			t.Errorf("Falscher Absender: %s", capturedFrom)
		}
		if len(capturedTo) != 1 || capturedTo[0] != "to@example.com" {
			t.Errorf("Falsche Empfänger: %v", capturedTo)
		}
		if !strings.Contains(string(capturedMsg), "Subject: Schulbuch-Inventar Backup") {
			t.Error("Subject fehlt oder inkorrekt in der E-Mail")
		}
	})

	t.Run("SendMailError", func(t *testing.T) {
		sendMail = func(addr string, a smtp.Auth, from string, to []string, msg []byte) error {
			return errors.New("smtp fehler")
		}

		tempDir := t.TempDir()
		os.WriteFile(filepath.Join(tempDir, "test.txt"), []byte("test"), 0644)

		config := &BackupEmailConfig{
			Host:     "smtp.example.com",
			Port:     "587",
			User:     "user@example.com",
			Password: "password",
			To:       "to@example.com",
		}

		err := SendBackupEmail(config, tempDir)
		if err == nil {
			t.Error("Erwartete Fehler von sendMail, bekam keinen")
		} else if !strings.Contains(err.Error(), "E-Mail-Versand fehlgeschlagen") {
			t.Errorf("Unerwartete Fehlermeldung: %v", err)
		}
	})
}
