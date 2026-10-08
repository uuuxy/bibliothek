package api

// backup_status.go — Backup-Status-Wächter für das Admin-Dashboard.
// Der nächtliche Backup-Job (jobs/backup.go) überspringt sich STILL, wenn
// BACKUP_ENCRYPTION_KEY fehlt — ohne diesen Endpunkt fiele das erst beim
// Restore-Versuch auf. Der Wächter prüft Key-Präsenz und das Alter der
// jüngsten backup_*.sql.gz.enc-Datei im BACKUP_DIR.

import (
	"net/http"
	"os"
	"time"

	"bibliothek/jobs"
)

// BackupStatusResponse beschreibt den Zustand der nächtlichen Datenbank-Backups.
type BackupStatusResponse struct {
	LastBackupAt      *time.Time `json:"last_backup_at"` // RFC3339; null = noch nie
	EncryptionKeySet  bool       `json:"encryption_key_set"`
	EncryptionKeyWeak bool       `json:"encryption_key_weak"` // gesetzt, aber zu kurz (unter MinBackupSchluesselLaenge)
	Status            string     `json:"status"`              // "ok" | "warning" | "critical"
}

// newestBackupTime liefert den ModTime der jüngsten Backup-Datei oder nil,
// wenn (noch) keine existiert. Ein fehlendes Verzeichnis ist kein Fehler,
// sondern schlicht "kein Backup vorhanden".
func newestBackupTime(dir string) *time.Time {
	dateien, err := jobs.BackupDateien(dir)
	if err != nil || len(dateien) == 0 {
		return nil
	}
	var newest time.Time
	for _, d := range dateien {
		if d.GeaendertAm.After(newest) {
			newest = d.GeaendertAm
		}
	}
	if newest.IsZero() {
		return nil
	}
	return &newest
}

// BackupStatusHandler liefert den Backup-Zustand fürs Admin-Badge.
// GET /api/admin/system/backup-status
func (s *Server) BackupStatusHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		encKey := os.Getenv("BACKUP_ENCRYPTION_KEY")
		keySet := encKey != ""
		keyWeak := jobs.SchluesselIstSchwach(encKey)

		dir := os.Getenv("BACKUP_DIR")
		if dir == "" {
			dir = "./backups" // identischer Default wie jobs/backup.go
		}
		last := newestBackupTime(dir)

		RespondJSON(w, http.StatusOK, BackupStatusResponse{
			LastBackupAt:      last,
			EncryptionKeySet:  keySet,
			EncryptionKeyWeak: keyWeak,
			Status:            jobs.BackupStatus(keySet, keyWeak, last, time.Now()),
		})
	}
}
