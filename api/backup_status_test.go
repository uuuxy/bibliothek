package api

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func doBackupStatus(t *testing.T) *httptest.ResponseRecorder {
	t.Helper()
	s := &Server{}
	req := httptest.NewRequest(http.MethodGet, "/api/admin/system/backup-status", nil)
	rec := httptest.NewRecorder()
	s.BackupStatusHandler()(rec, req)
	return rec
}

// writeBackupFile legt eine Backup-Datei mit definiertem Alter an.
func writeBackupFile(t *testing.T, dir string, age time.Duration) {
	t.Helper()
	f := filepath.Join(dir, "backup_2026-07-09.sql.gz.enc")
	if err := os.WriteFile(f, []byte("x"), 0600); err != nil {
		t.Fatalf("write: %v", err)
	}
	mtime := time.Now().Add(-age)
	if err := os.Chtimes(f, mtime, mtime); err != nil {
		t.Fatalf("chtimes: %v", err)
	}
}

func TestBackupStatus_MissingKeyIsCritical(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("BACKUP_DIR", dir)
	t.Setenv("BACKUP_ENCRYPTION_KEY", "")
	writeBackupFile(t, dir, time.Hour) // frisches Backup ändert nichts: ohne Key läuft der Job still leer

	body := doBackupStatus(t).Body.String()
	if !strings.Contains(body, `"status":"critical"`) || !strings.Contains(body, `"encryption_key_set":false`) {
		t.Errorf("fehlender Key muss critical sein: %s", body)
	}
}

func TestBackupStatus_EmptyDirIsCriticalNotCrash(t *testing.T) {
	t.Setenv("BACKUP_DIR", t.TempDir()) // leer
	t.Setenv("BACKUP_ENCRYPTION_KEY", "test-key")

	rec := doBackupStatus(t)
	if rec.Code != http.StatusOK {
		t.Fatalf("leeres Verzeichnis darf nicht crashen: %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `"status":"critical"`) || !strings.Contains(body, `"last_backup_at":null`) {
		t.Errorf("kein Backup = critical mit null-Timestamp: %s", body)
	}
}

func TestBackupStatus_MissingDirIsCriticalNotCrash(t *testing.T) {
	t.Setenv("BACKUP_DIR", filepath.Join(t.TempDir(), "gibt-es-nicht"))
	t.Setenv("BACKUP_ENCRYPTION_KEY", "test-key")

	rec := doBackupStatus(t)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"status":"critical"`) {
		t.Errorf("fehlendes Verzeichnis = sauberer critical: %d %s", rec.Code, rec.Body.String())
	}
}

func TestBackupStatus_Thresholds(t *testing.T) {
	cases := []struct {
		age  time.Duration
		want string
	}{
		{2 * time.Hour, `"status":"ok"`},
		{30 * time.Hour, `"status":"warning"`},  // ein verpasster Tageslauf
		{50 * time.Hour, `"status":"critical"`}, // Puffer aufgebraucht
	}
	for _, c := range cases {
		dir := t.TempDir()
		t.Setenv("BACKUP_DIR", dir)
		t.Setenv("BACKUP_ENCRYPTION_KEY", "test-key")
		writeBackupFile(t, dir, c.age)

		body := doBackupStatus(t).Body.String()
		if !strings.Contains(body, c.want) {
			t.Errorf("Alter %v: erwartet %s, Body: %s", c.age, c.want, body)
		}
		if strings.Contains(body, `"last_backup_at":null`) {
			t.Errorf("Alter %v: Timestamp fehlt: %s", c.age, body)
		}
	}
}
