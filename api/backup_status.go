package api

// backup_status.go — Backup-Status-Wächter für das Admin-Dashboard.
// Der nächtliche Backup-Job (jobs/backup.go) überspringt sich STILL, wenn
// BACKUP_ENCRYPTION_KEY fehlt — ohne diesen Endpunkt fiele das erst beim
// Restore-Versuch auf. Der Wächter prüft Key-Präsenz und das Alter der
// jüngsten backup_*.sql.gz.enc-Datei im BACKUP_DIR.

import (
	"net/http"
	"os"
	"path/filepath"
	"time"
)

const (
	// backupWarnAge: Der Job läuft täglich 02:30 — ist das jüngste Backup älter
	// als 26h, wurde mindestens ein Lauf verpasst.
	backupWarnAge = 26 * time.Hour
	// backupCriticalAge: Ab 48h ohne Backup ist der Datenverlust-Puffer weg.
	backupCriticalAge = 48 * time.Hour
)

// BackupStatusResponse beschreibt den Zustand der nächtlichen Datenbank-Backups.
type BackupStatusResponse struct {
	LastBackupAt     *time.Time `json:"last_backup_at"` // RFC3339; null = noch nie
	EncryptionKeySet bool       `json:"encryption_key_set"`
	Status           string     `json:"status"` // "ok" | "warning" | "critical"
}

// newestBackupTime liefert den ModTime der jüngsten Backup-Datei oder nil,
// wenn (noch) keine existiert. Ein fehlendes Verzeichnis ist kein Fehler,
// sondern schlicht "kein Backup vorhanden".
func newestBackupTime(dir string) *time.Time {
	matches, err := filepath.Glob(filepath.Join(dir, "backup_*.sql.gz.enc"))
	if err != nil || len(matches) == 0 {
		return nil
	}
	var newest time.Time
	for _, m := range matches {
		info, statErr := os.Stat(m) //nolint:gosec // Pre-existing G703
		if statErr != nil {
			continue
		}
		if info.ModTime().After(newest) {
			newest = info.ModTime()
		}
	}
	if newest.IsZero() {
		return nil
	}
	return &newest
}

// computeBackupStatus wendet die Schwellen an — als reine Funktion testbar.
func computeBackupStatus(keySet bool, last *time.Time, now time.Time) string {
	if !keySet || last == nil {
		return "critical"
	}
	age := now.Sub(*last)
	switch {
	case age > backupCriticalAge:
		return "critical"
	case age > backupWarnAge:
		return "warning"
	default:
		return "ok"
	}
}

// BackupStatusHandler liefert den Backup-Zustand fürs Admin-Badge.
// GET /api/admin/system/backup-status
func (s *Server) BackupStatusHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		keySet := os.Getenv("BACKUP_ENCRYPTION_KEY") != ""

		dir := os.Getenv("BACKUP_DIR")
		if dir == "" {
			dir = "./backups" // identischer Default wie jobs/backup.go
		}
		last := newestBackupTime(dir)

		RespondJSON(w, http.StatusOK, BackupStatusResponse{
			LastBackupAt:     last,
			EncryptionKeySet: keySet,
			Status:           computeBackupStatus(keySet, last, time.Now()),
		})
	}
}
