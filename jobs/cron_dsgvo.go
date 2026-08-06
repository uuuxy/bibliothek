package jobs

import (
	"context"
	"log"
	"time"
)

// Die DSGVO-Löschroutinen des Schedulers. Getrennt von cron.go, weil dort nur noch steht,
// WANN etwas läuft — hier steht, WAS gelöscht und anonymisiert wird. Das sind die
// Fristen aus dem Fachkonzept und die einzige Stelle, an der das System von sich aus
// Personendaten entfernt; sie gehört nicht zwischen Registrierungszeilen.

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
		      (ist_abgaenger = true AND aktualisiert_am < NOW() - INTERVAL '360 days')
		  )
		  AND NOT EXISTS (SELECT 1 FROM ausleihen WHERE schueler_id = schueler.id AND rueckgabe_am IS NULL)
		  AND NOT EXISTS (SELECT 1 FROM schadensfaelle WHERE schueler_id = schueler.id AND ist_bezahlt = false)
	`

	tag, err := s.db.Exec(ctx, query)
	if err != nil {
		log.Printf("Scheduler GDPR Anonymize: Error anonymizing old students: %v", err)
		return
	}

	// Verschlüsselte Passfotos anonymisierter Schüler entfernen. Selbstheilend: räumt auch
	// Altbestände, deren Anonymisierung vor der Foto-Löschung lief. (Das Foto lebt in
	// schueler_fotos, nicht in schueler.foto_url — Letzteres wird oben bereits geleert.)
	if _, delErr := s.db.Exec(ctx,
		"DELETE FROM schueler_fotos WHERE schueler_id IN (SELECT id FROM schueler WHERE anonymized_at IS NOT NULL)",
	); delErr != nil {
		log.Printf("Scheduler GDPR Anonymize: Fotos anonymisierter Schüler konnten nicht gelöscht werden: %v", delErr)
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
