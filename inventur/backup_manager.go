package inventur

import (
	"log"
	"os"
	"time"
)

// BackupManager überwacht Datenänderungen und erstellt automatisch Backups.
// Nach einer Änderung wird eine Beruhigungsphase (Debounce) abgewartet,
// bevor das Backup tatsächlich ausgeführt wird. Weitere Änderungen in der
// Zwischenzeit setzen den Timer zurück.
type BackupManager struct {
	signalCh    chan struct{}
	stopCh      chan struct{}
	databaseURL string
	backupDir   string
	debounce    time.Duration
	maxBackups  int
	emailConfig *BackupEmailConfig
}

// NewBackupManager erstellt einen neuen BackupManager.
func NewBackupManager(databaseURL string) *BackupManager {
	return &BackupManager{
		signalCh:    make(chan struct{}, 1),
		stopCh:      make(chan struct{}),
		databaseURL: databaseURL,
		backupDir:   "backups",
		debounce:    2 * time.Minute,
		maxBackups:  10,
		emailConfig: NewBackupEmailConfigFromEnv(),
	}
}

// NotifyChange signalisiert dem BackupManager, dass Daten geändert wurden.
// Der Aufruf ist nicht-blockierend – wenn bereits ein Signal wartet, wird
// das neue Signal verworfen (macht nichts, der Timer wird trotzdem zurückgesetzt).
func (bm *BackupManager) NotifyChange() {
	select {
	case bm.signalCh <- struct{}{}:
	default:
	}
}

// Stop beendet den BackupManager sauber (Graceful Shutdown).
func (bm *BackupManager) Stop() {
	close(bm.stopCh)
}

// Start startet die Hintergrund-Goroutine des BackupManagers.
// Diese läuft bis Stop() aufgerufen wird und sollte als `go bm.Start()` gestartet werden.
func (bm *BackupManager) Start() {
	log.Println("BackupManager gestartet – Backups werden bei Datenänderungen erstellt")

	for {
		select {
		case <-bm.stopCh:
			log.Println("BackupManager gestoppt")
			return
		case <-bm.signalCh:
			log.Println("Backup: Datenänderung erkannt, warte auf Beruhigungsphase...")
		}

		// Debounce-Schleife: Timer zurücksetzen bei weiteren Änderungen
		timer := time.NewTimer(bm.debounce)
	drainLoop:
		for {
			select {
			case <-bm.stopCh:
				timer.Stop()
				log.Println("BackupManager gestoppt (während Debounce)")
				return
			case <-bm.signalCh:
				if !timer.Stop() {
					<-timer.C
				}
				timer.Reset(bm.debounce)
				log.Println("Backup: Weitere Änderung erkannt, Timer zurückgesetzt")
			case <-timer.C:
				break drainLoop
			}
		}

		bm.runBackup()
	}
}

// runBackup führt das eigentliche Backup aus: pg_dump + kopiere uploads.
func (bm *BackupManager) runBackup() {
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	backupPath := bm.backupDir + "/" + timestamp

	if err := os.MkdirAll(backupPath, 0755); err != nil {
		log.Printf("Backup FEHLER: Verzeichnis konnte nicht erstellt werden: %v", err)
		return
	}

	log.Printf("Backup: Starte Backup nach %s ...", backupPath)

	if err := bm.dumpDatabase(backupPath); err != nil {
		log.Printf("Backup FEHLER: Datenbank-Dump fehlgeschlagen: %v", err)
		os.RemoveAll(backupPath)
		return
	}

	if err := bm.copyUploads(backupPath); err != nil {
		log.Printf("Backup WARNUNG: Uploads konnten nicht kopiert werden: %v", err)
	}

	log.Printf("Backup ERFOLGREICH: %s", backupPath)

	if bm.emailConfig != nil {
		if err := SendBackupEmail(bm.emailConfig, backupPath); err != nil {
			log.Printf("Backup WARNUNG: E-Mail-Versand fehlgeschlagen: %v", err)
		}
	}

	bm.rotateBackups()
}
