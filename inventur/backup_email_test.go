package inventur

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewBackupEmailConfigFromEnv(t *testing.T) {
	// Test 1: Kein Host
	t.Setenv("SMTP_HOST", "")
	config := NewBackupEmailConfigFromEnv()
	if config != nil {
		t.Errorf("Erwartet nil wenn SMTP_HOST leer ist, aber %v erhalten", config)
	}

	// Test 2: Vollständige Konfiguration (mit Default-Port)
	t.Setenv("SMTP_HOST", "smtp.example.com")
	t.Setenv("SMTP_PORT", "") // Sollte zu "587" werden
	t.Setenv("SMTP_USER", "user")
	t.Setenv("SMTP_PASSWORD", "pass")
	t.Setenv("BACKUP_EMAIL_TO", "to@example.com")

	config = NewBackupEmailConfigFromEnv()
	if config == nil {
		t.Fatal("Erwartet Config, aber nil erhalten")
	}
	if config.Host != "smtp.example.com" {
		t.Errorf("Erwartet Host smtp.example.com, aber %s erhalten", config.Host)
	}
	if config.Port != "587" {
		t.Errorf("Erwartet Port 587, aber %s erhalten", config.Port)
	}
	if config.User != "user" {
		t.Errorf("Erwartet User user, aber %s erhalten", config.User)
	}
	if config.Password != "pass" {
		t.Errorf("Erwartet Password pass, aber %s erhalten", config.Password)
	}
	if config.To != "to@example.com" {
		t.Errorf("Erwartet To to@example.com, aber %s erhalten", config.To)
	}

	// Test 3: Benutzerdefinierter Port
	t.Setenv("SMTP_PORT", "465")
	config = NewBackupEmailConfigFromEnv()
	if config.Port != "465" {
		t.Errorf("Erwartet Port 465, aber %s erhalten", config.Port)
	}
}

func TestSendBackupEmail_NotConfigured(t *testing.T) {
	// Test 1: Nil-Konfiguration
	err := SendBackupEmail(nil, "dummy/path")
	if err != nil {
		t.Errorf("Erwartet keinen Fehler, wenn config nil ist, aber %v erhalten", err)
	}

	// Test 2: Keine Empfängeradresse
	config := &BackupEmailConfig{
		Host: "smtp.example.com",
		To:   "",
	}
	err = SendBackupEmail(config, "dummy/path")
	if err != nil {
		t.Errorf("Erwartet keinen Fehler, wenn config.To leer ist, aber %v erhalten", err)
	}
}

func TestCreateZip(t *testing.T) {
	// Temp-Verzeichnis erstellen
	tempDir := t.TempDir()

	// Dateien und Ordner erstellen
	subDir := filepath.Join(tempDir, "subdir")
	err := os.Mkdir(subDir, 0755)
	if err != nil {
		t.Fatalf("Konnte Unterordner nicht erstellen: %v", err)
	}

	file1Path := filepath.Join(tempDir, "file1.txt")
	err = os.WriteFile(file1Path, []byte("Inhalt 1"), 0644)
	if err != nil {
		t.Fatalf("Konnte Datei nicht erstellen: %v", err)
	}

	file2Path := filepath.Join(subDir, "file2.txt")
	err = os.WriteFile(file2Path, []byte("Inhalt 2"), 0644)
	if err != nil {
		t.Fatalf("Konnte Datei nicht erstellen: %v", err)
	}

	// createZip ausführen
	zipData, err := createZip(tempDir)
	if err != nil {
		t.Fatalf("createZip fehlgeschlagen: %v", err)
	}

	// ZIP-Inhalt verifizieren
	zipReader, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	if err != nil {
		t.Fatalf("Konnte ZIP-Reader nicht erstellen: %v", err)
	}

	var foundFile1, foundFile2, foundSubDir bool
	for _, file := range zipReader.File {
		if file.Name == "file1.txt" {
			foundFile1 = true
		}
		if file.Name == "subdir/" {
			foundSubDir = true
		}
		if file.Name == "subdir/file2.txt" {
			foundFile2 = true
		}
	}

	if !foundFile1 {
		t.Error("file1.txt fehlt im ZIP")
	}
	if !foundSubDir {
		t.Error("subdir/ fehlt im ZIP")
	}
	if !foundFile2 {
		t.Error("subdir/file2.txt fehlt im ZIP")
	}
}

func TestBuildEmailWithAttachment(t *testing.T) {
	msg := EmailMessage{
		From:       "sender@example.com",
		To:         "receiver@example.com",
		Subject:    "Test Subject",
		Body:       "Test Body",
		AttachName: "test.txt",
		AttachData: []byte("Test Attachment Data"),
	}

	emailData, err := buildEmailWithAttachment(msg)
	if err != nil {
		t.Fatalf("buildEmailWithAttachment fehlgeschlagen: %v", err)
	}

	emailStr := string(emailData)

	// Prüfen, ob Header vorhanden sind
	if !strings.Contains(emailStr, "From: sender@example.com") {
		t.Error("From-Header fehlt oder ist falsch")
	}
	if !strings.Contains(emailStr, "To: receiver@example.com") {
		t.Error("To-Header fehlt oder ist falsch")
	}
	if !strings.Contains(emailStr, "Subject: Test Subject") {
		t.Error("Subject-Header fehlt oder ist falsch")
	}

	// Prüfen, ob Body vorhanden ist
	if !strings.Contains(emailStr, "Test Body") {
		t.Error("Email-Body fehlt")
	}

	// Prüfen, ob Attachment-Name vorhanden ist
	if !strings.Contains(emailStr, "filename=\"test.txt\"") {
		t.Error("Attachment-Dateiname fehlt")
	}
}
