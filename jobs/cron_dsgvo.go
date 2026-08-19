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

	// ALLE direkt identifizierenden Spalten leeren, nicht nur den Namen: Bis 19.08.2026
	// referenzierte diese Query eine nicht existente Spalte foto_url und scheiterte
	// deshalb ZUR LAUFZEIT beim Planen — die Anonymisierung lief NIE, der Fehler wurde nur
	// geloggt. Zudem hätte barcode_id = NULL an der NOT-NULL-Constraint gescheitert, und
	// Adresse, Eltern-E-Mail, Geburtsdatum und LUSD-ID blieben in Klarschrift stehen (eine
	// Namens-Anonymisierung ohne diese ist wirkungslos: Adresse+Geburtsdatum
	// reidentifizieren). barcode_id wird auf einen anonymen, eindeutigen Wert gesetzt
	// (NOT NULL); block_reason auf einen festen Text statt NULL — chk_schueler_block_reason
	// verlangt bei gesperrten Schülern einen nicht-leeren Grund, und ein alter Freitext-
	// Grund könnte selbst personenbezogen sein. Deckt dieselben identifizierenden Felder ab
	// wie anonymisiereAbgaenger (LUSD-Pfad).
	query := `
		UPDATE schueler
		SET vorname = left(md5(random()::text), 8),
		    nachname = 'Anonym',
		    klasse = '',
		    barcode_id = 'ANON-' || id::text,
		    geburtsdatum = NULL,
		    lusd_id = NULL,
		    strasse = NULL,
		    hausnummer = NULL,
		    plz = NULL,
		    ort = NULL,
		    eltern_email = NULL,
		    block_reason = 'Anonymisiert (DSGVO)',
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
	// Altbestände, deren Anonymisierung vor der Foto-Löschung lief. Das Foto lebt in
	// schueler_fotos (BYTEA), es gibt keine schueler.foto_url-Spalte.
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
