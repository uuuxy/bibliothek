package jobs

import (
	"context"
	"fmt"
	"log"
	"time"

	"bibliothek/db"
	"bibliothek/repository"

	"github.com/robfig/cron/v3"
)

// Scheduler manages background automation tasks.
type Scheduler struct {
	db        db.PgxPoolIface
	auditRepo repository.AuditRepository
	cron      *cron.Cron
}

// NewScheduler builds and returns a new Scheduler instance.
func NewScheduler(db db.PgxPoolIface, auditRepo repository.AuditRepository) *Scheduler {
	return &Scheduler{
		db:        db,
		auditRepo: auditRepo,
		cron:      cron.New(),
	}
}

// Start registers all GDPR, backup, and data-retention cron schedules.
func (s *Scheduler) Start() {
	// Run GDPR anonymization and abgänger-deletion daily at midnight
	if _, err := s.cron.AddFunc("0 0 * * *", func() {
		s.RunGDPRAnonymizeLoans()
		s.RunGDPRDeleteAbgaenger()
	}); err != nil {
		log.Printf("Scheduler: Failed to register GDPR jobs: %v", err)
		return
	}

	// Daily encrypted database backup at 02:30 UTC (low-traffic window)
	backup := &BackupJob{}
	if _, err := s.cron.AddFunc("30 2 * * *", func() {
		log.Println("Scheduler Backup: starting scheduled daily database backup...")
		backup.RunDatabaseBackup()
	}); err != nil {
		log.Printf("Scheduler: Failed to register backup job: %v", err)
	}

	s.cron.Start()
	log.Println("Scheduler: GDPR, backup, and retention jobs successfully started.")
}

// Stop halts the scheduler's cron runner.
func (s *Scheduler) Stop() {
	s.cron.Stop()
}

// ── GDPR: Ausleihen-Anonymisierung ───────────────────────────────────────────

// RunGDPRAnonymizeLoans nullifies staff operator IDs for loans closed for more than 14 days.
// This fulfils the DSGVO Datensparsamkeit requirement for operator identity.
func (s *Scheduler) RunGDPRAnonymizeLoans() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	query := `
		UPDATE ausleihen
		SET bearbeiter_id = NULL,
		    rueckgabe_bearbeiter_id = NULL
		WHERE rueckgabe_am < NOW() - INTERVAL '14 days'
		  AND (bearbeiter_id IS NOT NULL OR rueckgabe_bearbeiter_id IS NOT NULL)
	`
	tag, err := s.db.Exec(ctx, query)
	if err != nil {
		log.Printf("Scheduler GDPR Anonymize: Error anonymizing operator IDs: %v", err)
		return
	}

	count := tag.RowsAffected()
	log.Printf("Scheduler GDPR Anonymize: anonymized %d loans (returned > 14 days ago)", count)

	// Write system audit record
	if count > 0 {
		_ = s.auditRepo.LogSystemAktion(ctx, "ausleihen", "ANONYMIZE",
			"GDPR 14-Tage-Anonymisierung der Bearbeiter-IDs",
			map[string]any{
				"betroffene_ausleihen": count,
				"schwellwert_tage":     14,
				"ausgefuehrt_am":       time.Now().UTC().Format(time.RFC3339),
			},
		)
	}
}

// ── GDPR: Abgänger-Löschung (30 Tage nach Schuljahresende) ──────────────────

// RunGDPRDeleteAbgaenger performs a DSGVO-compliant hard-delete of former students
// (ist_abgaenger = true) who:
//   - left school in a prior year (abgaenger_jahr < current year), AND
//   - have no unreturned books, AND
//   - have no unpaid damage fees, AND
//   - it is at least 30 days past the start of the current calendar year
//     (approximation for "30 Tage nach Schuljahresende").
//
// Each deletion is individually logged in audit_log (SYSTEM actor).
func (s *Scheduler) RunGDPRDeleteAbgaenger() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// 30-day grace period: only delete if it's at least Jan 30 of the year after graduation
	now := time.Now()
	cutoffYear := now.Year()
	cutoffDate := time.Date(cutoffYear, time.January, 30, 0, 0, 0, 0, time.UTC)
	if now.Before(cutoffDate) {
		// Before Jan 30: use previous year as cutoff (last year's graduates still in grace period)
		cutoffYear--
	}

	// Fetch eligible student IDs
	query := `
		SELECT id, vorname, nachname, klasse, barcode_id, abgaenger_jahr
		FROM schueler
		WHERE ist_abgaenger = true
		  AND abgaenger_jahr < $1
		  AND NOT EXISTS (
		      SELECT 1 FROM ausleihen
		      WHERE schueler_id = schueler.id AND rueckgabe_am IS NULL
		  )
		  AND NOT EXISTS (
		      SELECT 1 FROM schadensfaelle
		      WHERE schueler_id = schueler.id AND ist_bezahlt = false
		  )
	`
	rows, err := s.db.Query(ctx, query, cutoffYear)
	if err != nil {
		log.Printf("Scheduler GDPR Delete: Failed to fetch eligible students: %v", err)
		return
	}

	type eligibleStudent struct {
		ID            string
		Vorname       string
		Nachname      string
		Klasse        string
		BarcodeID     string
		AbgaengerJahr int
	}

	var students []eligibleStudent
	for rows.Next() {
		var s eligibleStudent
		if err := rows.Scan(&s.ID, &s.Vorname, &s.Nachname, &s.Klasse, &s.BarcodeID, &s.AbgaengerJahr); err != nil {
			log.Printf("Scheduler GDPR Delete: Scan error: %v", err)
			rows.Close()
			return
		}
		students = append(students, s)
	}
	rows.Close()

	if len(students) == 0 {
		log.Printf("Scheduler GDPR Delete: no eligible students for deletion (cutoff year: %d)", cutoffYear)
		return
	}

	log.Printf("Scheduler GDPR Delete: %d student(s) eligible for DSGVO deletion (Abgangsjahr < %d)",
		len(students), cutoffYear)

	deleted := 0
	var failures []string

	for _, student := range students {
		grund := fmt.Sprintf("DSGVO-Abgänger-Löschung: Abgangsjahr %d, Löschfrist abgelaufen (30 Tage Karenzzeit)",
			student.AbgaengerJahr)

		if err := s.auditRepo.DeleteStudent(ctx, student.ID, "", grund); err != nil {
			log.Printf("Scheduler GDPR Delete: failed to delete student %s %s (ID %s): %v",
				student.Vorname, student.Nachname, student.ID, err)
			failures = append(failures, student.ID)
			continue
		}

		log.Printf("Scheduler GDPR Delete: deleted %s %s (Klasse %s, Abgang %d)",
			student.Vorname, student.Nachname, student.Klasse, student.AbgaengerJahr)
		deleted++
	}

	// Write batch summary to audit log
	_ = s.auditRepo.LogSystemAktion(ctx, "schueler", "BATCH_DELETE",
		"DSGVO-Abgänger-Batch-Löschung",
		map[string]any{
			"geloescht":      deleted,
			"fehlschlaege":   len(failures),
			"cutoff_jahr":    cutoffYear,
			"ausgefuehrt_am": time.Now().UTC().Format(time.RFC3339),
		},
	)

	if len(failures) > 0 {
		log.Printf("Scheduler GDPR Delete: completed with %d failure(s): %v", len(failures), failures)
	} else {
		log.Printf("Scheduler GDPR Delete: successfully deleted %d student(s)", deleted)
	}
}
