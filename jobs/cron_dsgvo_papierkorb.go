package jobs

import (
	"context"
	"log"
	"time"

	"bibliothek/pkg/logger"
	"bibliothek/repository"
)

// ── Gelöschte Kollegen endgültig löschen (180 Tage im Papierkorb) ──────────────
//
// Ein gelöschter Kollege blieb bis zum 29.09.2026 ohne Frist im Papierkorb: Die
// Anonymisierung nimmt nur Schüler (PredikatAnonymisierung), und einen Kollegen darf sie nicht
// anonymisieren (chk_leser_nur_schueler_werden_abgaenger). Entschieden am 28.09.2026
// (docs/OFFEN.md 5.19): nach derselben Frist wie beim Schüler endgültig löschen. Sein
// Zugangskonto ist schon beim Löschen gegangen (DeleteStudent, entschieden am 16.09.2026).

// RunPapierkorbKollegenLoeschung löscht Kollegen endgültig, die seit 180 Tagen im Papierkorb
// liegen. Derselbe Weg wie „Endgültig löschen" von Hand: PurgeStudent prüft den Papierkorb und
// offene Vorgänge ein zweites Mal, in der Transaktion der Löschung.
func (s *Scheduler) RunPapierkorbKollegenLoeschung() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Die Kennungen sind vorab gelesen und die Zeilen geschlossen, bevor die Löschungen je
	// eine eigene Transaktion öffnen.
	ids, err := repository.KollegenImPapierkorb(ctx, s.db)
	if err != nil {
		log.Printf("Scheduler Papierkorb: Kollegen konnten nicht gelesen werden: %v", err)
		return
	}
	if len(ids) == 0 {
		return
	}

	geloescht := 0
	var fehlschlaege []string
	for _, id := range ids {
		// Leerer Bearbeiter: Das Protokoll nennt SYSTEM als Akteur statt einer Person.
		if err := s.auditRepo.PurgeStudent(ctx, id, ""); err != nil {
			log.Printf("Scheduler Papierkorb: Leser %s nicht gelöscht: %v", logger.SanitizeLog(id), err)
			fehlschlaege = append(fehlschlaege, id)
			continue
		}
		geloescht++
	}

	if err := s.auditRepo.LogSystemAktion(ctx, "schueler", "BATCH_DELETE",
		"DSGVO: gelöschte Kollegen nach 180 Tagen im Papierkorb endgültig gelöscht",
		map[string]any{
			"geloescht":    geloescht,
			"fehlschlaege": len(fehlschlaege),
			"tage":         repository.StandardAnonymisierungSoftDeleteTage,
		}); err != nil {
		log.Printf("Scheduler Papierkorb: Audit-Eintrag fehlgeschlagen: %v", err)
	}
	log.Printf("Scheduler Papierkorb: %d Kollegen endgültig gelöscht, %d Fehlschläge", geloescht, len(fehlschlaege))
}
