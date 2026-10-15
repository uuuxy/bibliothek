package system

// Der Wächter der Backups. Der nächtliche Backup-Job (jobs/backup.go) überspringt sich still,
// wenn BACKUP_ENCRYPTION_KEY fehlt; ohne diese Tür fiele das erst beim Versuch einer
// Wiederherstellung auf. Der Wächter prüft den Schlüssel und das Alter der jüngsten
// backup_*.sql.gz.enc-Datei im BACKUP_DIR.

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

// juengstesBackup liefert die Änderungszeit der jüngsten Backup-Datei oder nil,
// wenn (noch) keine existiert. Ein fehlendes Verzeichnis ist kein Fehler,
// sondern schlicht "kein Backup vorhanden".
func juengstesBackup(verzeichnis string) *time.Time {
	dateien, err := jobs.LeseBackupDateien(verzeichnis)
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

// BackupStand liefert den Zustand der Backups für den Fuß der Navigation und die Warnung
// über dem Inhalt.
// GET /api/admin/system/backup-status
func (t *Tueren) BackupStand() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		schluessel := os.Getenv("BACKUP_ENCRYPTION_KEY")
		gesetzt := schluessel != ""
		schwach := jobs.IstSchluesselSchwach(schluessel)

		verzeichnis := os.Getenv("BACKUP_DIR")
		if verzeichnis == "" {
			verzeichnis = "./backups" // dieselbe Vorgabe wie jobs/backup.go
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
