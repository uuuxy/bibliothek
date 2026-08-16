package jobs

import (
	"context"
	"log"
	"os"
	"time"

	"bibliothek/db"
	"bibliothek/internal/service"
	"bibliothek/repository"

	"github.com/robfig/cron/v3"
)

// Scheduler verwaltet Hintergrund-Automatisierungsaufgaben.
type Scheduler struct {
	db        db.PgxPoolIface
	auditRepo repository.AuditRepository
	cron      *cron.Cron
}

// NewScheduler erstellt und gibt eine neue Scheduler-Instanz zurück.
func NewScheduler(db db.PgxPoolIface, auditRepo repository.AuditRepository) *Scheduler {
	return &Scheduler{
		db:        db,
		auditRepo: auditRepo,
		cron:      cron.New(),
	}
}

// Start registriert alle Cronjobs für DSGVO, Backup und Vorhaltefristen.
func (s *Scheduler) Start() {
	// Tägliche DSGVO-Anonymisierung und Abgänger-Löschung um Mitternacht
	if _, err := s.cron.AddFunc("0 0 * * *", func() {
		s.RunGDPRAnonymizeLoans()
		s.RunGDPRDeleteAbgaenger()
		s.RunGDPRAnonymizeOldData()
	}); err != nil {
		log.Printf("Scheduler: Failed to register GDPR jobs: %v", err)
		return
	}

	// Nächtliche Audit-Aufbewahrung um 03:00 UTC — nach dem Backup (02:30), damit die
	// Sicherung des Tages die Einträge noch VOR ihrer Löschung enthält.
	if _, err := s.cron.AddFunc("0 3 * * *", s.RunAuditAufbewahrung); err != nil {
		log.Printf("Scheduler: Failed to register audit retention job: %v", err)
	}

	// Tägliches verschlüsseltes Datenbank-Backup um 02:30 UTC (Zeitraum mit wenig Traffic)
	backup := &BackupJob{}

	// Beim Start sagen, nicht erst um 02:30 in einem Logeintrag, den niemand liest.
	//
	// Ohne BACKUP_ENCRYPTION_KEY überspringt sich der Job (jobs/backup.go) — still. Das
	// ist der gefährlichste Zustand, den dieses System kennt: Es läuft alles normal, und
	// erst wenn jemand ein Backup BRAUCHT, stellt sich heraus, dass es nie eines gab.
	// Der Admin-Wächter (api/backup_status.go) meldet den Fall zwar als "critical",
	// dafür muss aber jemand ins Dashboard sehen. Diese Zeile steht im Startprotokoll.
	if os.Getenv("BACKUP_ENCRYPTION_KEY") == "" {
		log.Printf("ACHTUNG: BACKUP_ENCRYPTION_KEY ist nicht gesetzt — es werden KEINE " +
			"Datenbank-Backups erstellt. Der nächtliche Job überspringt sich. Variable in " +
			"der .env setzen und den Stack neu starten.")
	} else if SchluesselIstSchwach(os.Getenv("BACKUP_ENCRYPTION_KEY")) {
		log.Printf("ACHTUNG: BACKUP_ENCRYPTION_KEY ist kürzer als %d Zeichen. Die Ableitung "+
			"per SHA-256 ist schnell — eine kurze Passphrase ist an einer entwendeten "+
			"Backup-Datei offline angreifbar.", MinBackupSchluesselLaenge)
	}

	if _, err := s.cron.AddFunc("30 2 * * *", func() {
		log.Println("Scheduler Backup: starting scheduled daily database backup...")
		backup.RunDatabaseBackup()
	}); err != nil {
		log.Printf("Scheduler: Failed to register backup job: %v", err)
	}

	// Stündliche Bereinigung abgelaufener Idempotenz-Schlüssel, damit die Tabelle nicht
	// unbegrenzt wächst. 24h Retention reicht weit über die Lebensdauer eines Scan-Retrys hinaus.
	if _, err := s.cron.AddFunc("17 * * * *", func() {
		s.RunIdempotencyCleanup()
	}); err != nil {
		log.Printf("Scheduler: Failed to register idempotency cleanup job: %v", err)
	}

	// Alle 6 Stunden fehlende/fehlgeschlagene Buchcover nachladen (neu importierte Titel
	// und transiente FAILED-Fälle). Der Re-Entrancy-Guard verhindert Überlappung mit dem
	// Start-Lauf. Lokales WebP wird dabei erzeugt.
	if _, err := s.cron.AddFunc("0 */6 * * *", func() {
		service.NewCoverService(s.db).SyncMissingCoversAsync()
	}); err != nil {
		log.Printf("Scheduler: Failed to register cover resync job: %v", err)
	}

	s.cron.Start()
	log.Println("Scheduler: GDPR, backup, retention, and idempotency cleanup jobs successfully started.")
}

// Stop hält den Cron-Runner des Schedulers an.
func (s *Scheduler) Stop() {
	s.cron.Stop()
}

// RunIdempotencyCleanup entfernt abgelaufene Idempotenz-Schlüssel (> 24h), damit die Tabelle
// nicht unbegrenzt wächst. Die Lebensdauer eines Scanner-Retrys liegt im Sekunden-/Minutenbereich,
// daher ist 24h Retention großzügig.
func (s *Scheduler) RunIdempotencyCleanup() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tag, err := s.db.Exec(ctx, "DELETE FROM idempotency_keys WHERE created_at < NOW() - INTERVAL '24 hours'")
	if err != nil {
		log.Printf("Scheduler Idempotency Cleanup: Fehler beim Löschen abgelaufener Schlüssel: %v", err)
		return
	}
	if n := tag.RowsAffected(); n > 0 {
		log.Printf("Scheduler Idempotency Cleanup: %d abgelaufene Idempotenz-Schlüssel entfernt.", n)
	}
}
