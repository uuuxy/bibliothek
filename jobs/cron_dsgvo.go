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

	// PII-Spuren anonymisierter Schüler in den NEBEN-Tabellen tilgen. Bis 21.08.2026
	// leerte dieser Pfad NUR die schueler-Zeile — der Hard-Delete-Pfad
	// (entferneSchuelerPIIUndLoesche) räumte die Audit-Details, die Feld-Anonymisierung
	// aber nicht. Folge: Ein soft-gelöschter Schüler war nach 180 Tagen „anonymisiert",
	// sein Klarname (audit_log.details) und seine LUSD-ID (audit_logs.details) lebten
	// jedoch bis zur Audit-Aufbewahrung (24 Monate) weiter — bis zu 1,5 Jahre
	// Personenbezug nach der Löschung. Selbstheilend über anonymized_at, damit auch
	// Altbestände nachgezogen werden.
	s.bereinigeAnonymisierteSchuelerSpuren(ctx)

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

// bereinigeAnonymisierteSchuelerSpuren tilgt die PII anonymisierter Schüler aus den
// Neben-Tabellen, die RunGDPRAnonymizeOldData selbst nicht anfasst. Jede Anweisung ist
// idempotent und selbstheilend (Kriterium: anonymized_at IS NOT NULL) und wird einzeln
// protokolliert — schlägt eine fehl (etwa am Append-Only-Trigger auf audit_log, den es
// nur auf manchen Altbeständen gibt), bricht das die übrigen NICHT ab.
func (s *Scheduler) bereinigeAnonymisierteSchuelerSpuren(ctx context.Context) {
	anonymisiert := `SELECT id FROM schueler WHERE anonymized_at IS NOT NULL`

	// 1. audit_log (fachliche Datensatz-Historie): DeleteStudent legt Vor-/Nachname,
	//    Klasse und Barcode in details ab. Dieselbe Neutralisierung wie im Hard-Delete-
	//    Pfad (entferneSchuelerPIIUndLoesche). Idempotent über den 'anonymisiert'-Marker.
	if tag, err := s.db.Exec(ctx, `
		UPDATE audit_log
		SET details = jsonb_build_object('anonymisiert', true, 'grund', 'DSGVO-Anonymisierung')
		WHERE tabelle = 'schueler'
		  AND datensatz_id IN (`+anonymisiert+`)
		  AND (details IS NULL OR NOT (details ? 'anonymisiert'))`); err != nil {
		log.Printf("Scheduler GDPR Anonymize: audit_log-PII konnte nicht getilgt werden: %v", err)
	} else if n := tag.RowsAffected(); n > 0 {
		log.Printf("Scheduler GDPR Anonymize: %d audit_log-Einträge anonymisiert.", n)
	}

	// 2. audit_logs (Admin-Eingriffe): LUSD_ID_NACHGETRAGEN speichert die staatliche
	//    Schülerkennung im Klartext. Nur den PII-Schlüssel entfernen; Aktion, Zeit und
	//    schueler_id (jetzt Pseudonym) bleiben für die Rechenschaftspflicht erhalten.
	if tag, err := s.db.Exec(ctx, `
		UPDATE audit_logs
		SET details = details - 'lusd_id'
		WHERE details ? 'lusd_id'
		  AND details->>'schueler_id' IN (SELECT id::text FROM schueler WHERE anonymized_at IS NOT NULL)`); err != nil {
		log.Printf("Scheduler GDPR Anonymize: audit_logs-LUSD-ID konnte nicht getilgt werden: %v", err)
	} else if n := tag.RowsAffected(); n > 0 {
		log.Printf("Scheduler GDPR Anonymize: %d audit_logs-Einträge um die LUSD-ID bereinigt.", n)
	}

	// 3. vormerkungen: Die Freitext-Notiz kann personenbezogen sein, und eine Vormerkung
	//    eines abgegangenen/gelöschten Schülers ist ohnehin funktionslos. Der Hard-Delete-
	//    Pfad räumt sie via CASCADE; hier werden sie gezielt gelöscht.
	if tag, err := s.db.Exec(ctx, `
		DELETE FROM vormerkungen
		WHERE schueler_id IN (`+anonymisiert+`)`); err != nil {
		log.Printf("Scheduler GDPR Anonymize: Vormerkungen anonymisierter Schüler konnten nicht gelöscht werden: %v", err)
	} else if n := tag.RowsAffected(); n > 0 {
		log.Printf("Scheduler GDPR Anonymize: %d Vormerkungen anonymisierter Schüler gelöscht.", n)
	}
}
