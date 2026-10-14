package system

// backup_status.go — Backup-Status-Wächter für das Admin-Dashboard.
// Der nächtliche Backup-Job (jobs/backup.go) überspringt sich STILL, wenn
// BACKUP_ENCRYPTION_KEY fehlt — ohne diesen Endpunkt fiele das erst beim
// Restore-Versuch auf. Der Wächter prüft Key-Präsenz und das Alter der
// jüngsten backup_*.sql.gz.enc-Datei im BACKUP_DIR.

import (
	"net/http"
	"os"
	"time"

	"bibliothek/api/tuer"
	"bibliothek/jobs"
)

// backupStandAntwort beschreibt den Zustand der nächtlichen Datenbank-Backups.
type backupStandAntwort struct {
	LetztesBackup     *time.Time `json:"last_backup_at"` // RFC3339; null = noch nie
	SchluesselGesetzt bool       `json:"encryption_key_set"`
	SchluesselSchwach bool       `json:"encryption_key_weak"` // gesetzt, aber zu kurz (unter MinBackupSchluesselLaenge)
	Status            string     `json:"status"`              // "ok" | "warning" | "critical"
}

// juengstesBackup liefert den ModTime der jüngsten Backup-Datei oder nil,
// wenn (noch) keine existiert. Ein fehlendes Verzeichnis ist kein Fehler,
// sondern schlicht "kein Backup vorhanden".
func juengstesBackup(verzeichnis string) *time.Time {
	dateien, err := jobs.BackupDateien(verzeichnis)
	if err != nil || len(dateien) == 0 {
		return nil
	}
	var juengstes time.Time
	for _, d := range dateien {
		if d.GeaendertAm.After(juengstes) {
			juengstes = d.GeaendertAm
		}
	}
	if juengstes.IsZero() {
		return nil
	}
	return &juengstes
}

// BackupStand liefert den Backup-Zustand fürs Admin-Badge.
// GET /api/admin/system/backup-status
func (t *Tueren) BackupStand() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		schluessel := os.Getenv("BACKUP_ENCRYPTION_KEY")
		gesetzt := schluessel != ""
		schwach := jobs.SchluesselIstSchwach(schluessel)

		verzeichnis := os.Getenv("BACKUP_DIR")
		if verzeichnis == "" {
			verzeichnis = "./backups" // identischer Default wie jobs/backup.go
		}
		letztes := juengstesBackup(verzeichnis)

		tuer.AntworteJSON(w, http.StatusOK, backupStandAntwort{
			LetztesBackup:     letztes,
			SchluesselGesetzt: gesetzt,
			SchluesselSchwach: schwach,
			Status:            jobs.BackupStatus(gesetzt, schwach, letztes, time.Now()),
		})
	}
}
