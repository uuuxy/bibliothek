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

// ── GDPR: Abgänger-Löschung (ab 30. Januar des Folgejahres, nur anonymisierte Zeilen) ──

// RunGDPRDeleteAbgaenger führt eine DSGVO-konforme harte Löschung ehemaliger Schüler durch
// (ist_abgaenger = true), die:
//   - die Schule in einem vergangenen Jahr verlassen haben (abgaenger_jahr < aktuelles Jahr), UND
//   - keine unzurückgegebenen Bücher haben, UND
//   - keine unbezahlten Schadensgebühren haben, UND
//   - mindestens 30 Tage seit Beginn des aktuellen Kalenderjahres vergangen sind
//     (Näherungswert für „30 Tage nach Schuljahresende"); gelöscht wird nur, was die Karenz
//     durchlaufen hat, also schon anonymisiert ist — die Löschung läuft deshalb im Cron NACH
//     der Anonymisierung (RunNaechtlicheDSGVO, 05.09.2026).
//
// Jede Löschung wird einzeln im audit_log protokolliert (Akteur: SYSTEM).
func (s *Scheduler) RunGDPRDeleteAbgaenger() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// 30-tägige Karenzzeit als Stichjahr. Die Rechnung steht in repository/, weil die
	// Selbstprüfung mit demselben Jahr zählen muss — zwei Jahresrechnungen wären zwei
	// Fristen, von denen eine still danebenläge.
	jetzt := time.Now()
	cutoffYear := repository.AbgaengerStichjahr(jetzt)

	// Die Zeilen sind gelesen und geschlossen, die Verbindung ist zurück im Pool, bevor die
	// Löschphase beginnt.
	students, err := repository.LoeschreifeAbgaenger(ctx, s.db, jetzt)
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
