package jobs

import (
	"context"
	"log"
	"time"

	"bibliothek/pkg/logger"
	"bibliothek/repository"
)

// Die endgültige Löschung von Abgängern — der einzige Weg, auf dem dieses System
// Schülerdaten unwiederbringlich entfernt. Steht deshalb für sich und nicht zwischen
// den beiden Anonymisierungsläufen, die nur Felder überschreiben.

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

	// 30-tägige Karenzzeit als Stichjahr. Die Rechnung steht in repository/, weil die
	// Selbstprüfung mit demselben Jahr zählen muss — zwei Jahresrechnungen wären zwei
	// Fristen, von denen eine still danebenläge.
	cutoffYear := repository.AbgaengerStichjahr(time.Now())

	// Berechtigte Abgänger laden. Der Helfer schließt die Rows per defer — die
	// Connection ist damit zurück im Pool, bevor die eigentliche Löschphase beginnt.
	students, err := s.fetchDeletionEligibleStudents(ctx, cutoffYear)
	if err != nil {
		log.Printf("Scheduler GDPR Delete: Failed to fetch eligible students: %v", err)
		return
	}

	if len(students) == 0 {
		log.Printf("Scheduler GDPR Delete: no eligible students for deletion (cutoff year: %d)", cutoffYear)
		return
	}

	log.Printf("Scheduler GDPR Delete: %d student(s) eligible for DSGVO deletion (Abgangsjahr < %d)",
		len(students), cutoffYear)

	deleted := 0
	var failures []string

	for _, student := range students {
		// PurgeAbgaenger statt DeleteStudent: DeleteStudent ist ein Soft-Delete
		// (Papierkorb) — die PII bliebe erhalten. PurgeAbgaenger entfernt sie wirklich
		// (Ausleihhistorie anonymisiert, Datensatz gelöscht). Der Löschgrund steht im
		// Audit-Log über den festen Kontext der Methode.
		if err := s.auditRepo.PurgeAbgaenger(ctx, student.ID, ""); err != nil {
			log.Printf("Scheduler GDPR Delete: failed to purge student ID %s: %v",
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

// deletionEligibleStudent ist ein für die DSGVO-Löschung berechtigter Abgänger.
type deletionEligibleStudent struct {
	ID            string
	Vorname       string
	Nachname      string
	Klasse        string
	BarcodeID     string
	AbgaengerJahr int
}

// fetchDeletionEligibleStudents lädt alle löschberechtigten Abgänger (Abgangsjahr <
// cutoffYear, ohne offene Ausleihen und ohne unbezahlte Schadensgebühren). Die Rows
// werden per defer geschlossen — robust gegen künftige Early-Returns und die
// Connection kehrt vor der Löschphase in den Pool zurück.
func (s *Scheduler) fetchDeletionEligibleStudents(ctx context.Context, cutoffYear int) ([]deletionEligibleStudent, error) {
	query := `
		SELECT id, vorname, nachname, klasse, barcode_id, abgaenger_jahr
		FROM schueler
		WHERE ` + repository.PredikatAbgaengerLoeschung()
	rows, err := s.db.Query(ctx, query, cutoffYear)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var students []deletionEligibleStudent
	for rows.Next() {
		var st deletionEligibleStudent
		if err := rows.Scan(&st.ID, &st.Vorname, &st.Nachname, &st.Klasse, &st.BarcodeID, &st.AbgaengerJahr); err != nil {
			return nil, err
		}
		students = append(students, st)
	}
	return students, rows.Err()
}
