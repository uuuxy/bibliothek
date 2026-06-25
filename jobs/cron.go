package jobs

import (
	"context"
	"fmt"
	"log"
	"time"

	"bibliothek/db"
	"bibliothek/internal/service"
	"bibliothek/pkg/logger"
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

	// Tägliches verschlüsseltes Datenbank-Backup um 02:30 UTC (Zeitraum mit wenig Traffic)
	backup := &BackupJob{}
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

// ── GDPR: Ausleihen-Anonymisierung ───────────────────────────────────────────

// RunGDPRAnonymizeLoans annulliert die Mitarbeiter-Operator-IDs für Ausleihen, die länger als 14 Tage abgeschlossen sind.
// Dies erfüllt die DSGVO-Anforderung der Datensparsamkeit für die Operator-Identität.
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

	// System-Audit-Eintrag schreiben
	if count > 0 {
		if err := s.auditRepo.LogSystemAktion(ctx, "ausleihen", "ANONYMIZE",
			"GDPR 14-Tage-Anonymisierung der Bearbeiter-IDs",
			map[string]any{
				"betroffene_ausleihen": count,
				"schwellwert_tage":     14,
				"ausgefuehrt_am":       time.Now().UTC().Format(time.RFC3339),
			},
		); err != nil {
			log.Printf("audit: ANONYMIZE konnte nicht protokolliert werden: %v", err)
		}
	}
}

// ── GDPR: Abgänger-Löschung (30 Tage nach Schuljahresende) ──────────────────

// RunGDPRDeleteAbgaenger führt eine DSGVO-konforme harte Löschung ehemaliger Schüler durch
// (ist_abgaenger = true), die:
//   - die Schule in einem vergangenen Jahr verlassen haben (abgaenger_jahr < aktuelles Jahr), UND
//   - keine unzurückgegebenen Bücher haben, UND
//   - keine unbezahlten Schadensgebühren haben, UND
//   - mindestens 30 Tage seit Beginn des aktuellen Kalenderjahres vergangen sind
//     (Näherungswert für "30 Tage nach Schuljahresende").
//
// Jede Löschung wird einzeln im audit_log protokolliert (Akteur: SYSTEM).
func (s *Scheduler) RunGDPRDeleteAbgaenger() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// 30-tägige Karenzzeit: nur löschen, wenn es mindestens der 30. Januar des Jahres nach dem Abgang ist
	now := time.Now()
	cutoffYear := now.Year()
	cutoffDate := time.Date(cutoffYear, time.January, 30, 0, 0, 0, 0, time.UTC)
	if now.Before(cutoffDate) {
		// Vor dem 30. Januar: vorheriges Jahr als Stichtag verwenden (Abgänger des letzten Jahres noch in Karenzzeit)
		cutoffYear--
	}

	// Berechtigte Schüler-IDs abrufen
	query := `
		SELECT id, vorname, nachname, klasse, barcode_id, abgaenger_jahr
		FROM schueler
		WHERE ist_abgaenger = true
		  AND deleted_at IS NULL
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
			log.Printf("Scheduler GDPR Delete: failed to delete student ID %s: %v",
				logger.SanitizeLog(student.ID), err)
			failures = append(failures, student.ID)
			continue
		}

		log.Printf("Scheduler GDPR Delete: deleted student ID %s (Klasse %s, Abgang %d)",
			logger.SanitizeLog(student.ID), logger.SanitizeLog(student.Klasse), student.AbgaengerJahr)
		deleted++
	}

	// Batch-Zusammenfassung ins Audit-Log schreiben
	if err := s.auditRepo.LogSystemAktion(ctx, "schueler", "BATCH_DELETE",
		"DSGVO-Abgänger-Batch-Löschung",
		map[string]any{
			"geloescht":      deleted,
			"fehlschlaege":   len(failures),
			"cutoff_jahr":    cutoffYear,
			"ausgefuehrt_am": time.Now().UTC().Format(time.RFC3339),
		},
	); err != nil {
		log.Printf("audit: BATCH_DELETE konnte nicht protokolliert werden: %v", err)
	}

	if len(failures) > 0 {
		log.Printf("Scheduler GDPR Delete: completed with %d failure(s): %v", len(failures), failures)
	} else {
		log.Printf("Scheduler GDPR Delete: successfully deleted %d student(s)", deleted)
	}
}

// ── GDPR: Anonymisierung alter Datensätze (180 Tage nach Soft-Delete / 360 Tage Abgänger) ──

// RunGDPRAnonymizeOldData anonymisiert Schüler, die entweder:
// - seit mehr als 180 Tagen weichgelöscht sind (deleted_at < NOW - 180 Tage)
// - seit mehr als 360 Tagen als Abgänger markiert sind (aktualisiert_am < NOW - 360 Tage UND ist_abgaenger = true)
// Es werden Vorname, Nachname und Klasse geleert oder gehasht und anonymized_at gesetzt.
func (s *Scheduler) RunGDPRAnonymizeOldData() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	query := `
		UPDATE schueler
		SET vorname = left(md5(random()::text), 8),
		    nachname = 'Anonym',
		    klasse = '',
		    barcode_id = NULL,
		    foto_url = NULL,
		    anonymized_at = NOW(),
		    aktualisiert_am = NOW()
		WHERE anonymized_at IS NULL
		  AND (
		      (deleted_at IS NOT NULL AND deleted_at < NOW() - INTERVAL '180 days')
		      OR
		      (ist_abgaenger = true AND abgaenger_jahr <= EXTRACT(YEAR FROM NOW() - INTERVAL '360 days'))
		  )
	`

	tag, err := s.db.Exec(ctx, query)
	if err != nil {
		log.Printf("Scheduler GDPR Anonymize: Error anonymizing old students: %v", err)
		return
	}

	count := tag.RowsAffected()
	if count > 0 {
		log.Printf("Scheduler GDPR Anonymize: successfully anonymized %d old student records.", count)
		if err := s.auditRepo.LogSystemAktion(ctx, "schueler", "ANONYMIZE",
			"DSGVO Anonymisierung alter Datensätze (Soft-Delete > 180T oder Abgänger > 360T)",
			map[string]any{
				"betroffene_schueler": count,
				"ausgefuehrt_am":      time.Now().UTC().Format(time.RFC3339),
			},
		); err != nil {
			log.Printf("audit: ANONYMIZE konnte nicht protokolliert werden: %v", err)
		}
	} else {
		log.Printf("Scheduler GDPR Anonymize: no old students found to anonymize.")
	}
}
