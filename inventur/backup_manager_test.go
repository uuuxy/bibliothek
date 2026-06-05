package inventur

import (
	"os"
	"testing"
	"time"
)

func TestNewBackupManager(t *testing.T) {
	// Sichern und Wiederherstellen der Umgebungsvariablen
	originalSmtpHost := os.Getenv("SMTP_HOST")
	defer os.Setenv("SMTP_HOST", originalSmtpHost)

	// Szenario 1: Ohne SMTP_HOST (emailConfig sollte nil sein)
	os.Unsetenv("SMTP_HOST")
	dbURL := "postgres://user:pass@localhost:5432/db"
	bm := NewBackupManager(dbURL)

	if bm == nil {
		t.Fatal("NewBackupManager returned nil")
	}

	if bm.databaseURL != dbURL {
		t.Errorf("Expected databaseURL %s, got %s", dbURL, bm.databaseURL)
	}

	if bm.signalCh == nil {
		t.Error("Expected signalCh to be initialized")
	}

	if cap(bm.signalCh) != 1 {
		t.Errorf("Expected signalCh capacity 1, got %d", cap(bm.signalCh))
	}

	if bm.stopCh == nil {
		t.Error("Expected stopCh to be initialized")
	}

	if bm.backupDir != "backups" {
		t.Errorf("Expected backupDir 'backups', got '%s'", bm.backupDir)
	}

	if bm.debounce != 2*time.Minute {
		t.Errorf("Expected debounce 2m, got %v", bm.debounce)
	}

	if bm.maxBackups != 10 {
		t.Errorf("Expected maxBackups 10, got %d", bm.maxBackups)
	}

	if bm.emailConfig != nil {
		t.Errorf("Expected emailConfig to be nil, got %v", bm.emailConfig)
	}

	// Szenario 2: Mit SMTP_HOST (emailConfig sollte nicht nil sein)
	os.Setenv("SMTP_HOST", "smtp.example.com")
	bmWithEmail := NewBackupManager(dbURL)

	if bmWithEmail.emailConfig == nil {
		t.Error("Expected emailConfig to be initialized when SMTP_HOST is set")
	} else if bmWithEmail.emailConfig.Host != "smtp.example.com" {
		t.Errorf("Expected emailConfig.Host 'smtp.example.com', got '%s'", bmWithEmail.emailConfig.Host)
	}
}
