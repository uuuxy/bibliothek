package jobs

import (
	"context"
	"log"
	"time"

	"bibliothek/repository"
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

// ── GDPR: Anonymisierung alter Datensätze (180 Tage nach Soft-Delete / Karenzzeit nach Abgang) ──

// RunGDPRAnonymizeOldData anonymisiert Schüler, die entweder:
//   - seit mehr als 180 Tagen weichgelöscht sind (deleted_at < NOW - 180 Tage)
//   - länger als die Karenzzeit (Einstellung abgaenger_karenz_tage, Vorgabe 90) Abgänger
//     sind (abgaenger_seit, Migration 094) und keine offenen Vorgänge mehr haben.
//
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
	einst, err := repository.NewSystemSettingsRepository(s.db).GetSettings(ctx)
	if err != nil {
		log.Printf("Scheduler GDPR Anonymize: Einstellungen nicht lesbar, Vorgabe-Karenz gilt: %v", err)
	}
	karenzTage := repository.AbgaengerKarenzTageOderStandard(einst)
	bedingung := repository.PredikatAnonymisierung(karenzTage, repository.KulanzJob)
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
		WHERE ` + bedingung.Where

	// Die Bedingung kommt aus repository/loeschfristen.go — DIESELBE, die die
	// Selbstprüfung als count(*) stellt, samt ihrer Zahlen. Vorher stand sie hier und
	// dort getrennt: Der Wächter hätte weiter „alles gut" gemeldet, wenn diese Abfrage
	// sich ändert.
	tag, err := s.db.Exec(ctx, query, bedingung.Args...)
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
			"DSGVO Anonymisierung alter Datensätze (Soft-Delete > 180T oder Abgänger > Karenzzeit)",
			map[string]any{
				"betroffene_schueler": count,
				"karenz_tage":         karenzTage,
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
// Neben-Tabellen, die RunGDPRAnonymizeOldData selbst nicht anfasst. Die Statements
// kommen aus repository.SpurTilgungen — DERSELBEN Liste, die auch Purge und
// LUSD-Abgang fahren. Bis 31.08.2026 stand hier eine eigene Abschrift mit drei der
// vier Statements; es fehlte genau die Lesehistorie, und der Klarname
// (details->>'entleiher') überlebte die Anonymisierung
// (jobs/dsgvo_spuren_paarung_pg_test.go, am alten Stand rot gesehen).
//
// Selbstheilend: Kriterium ist anonymized_at IS NOT NULL, jede Nacht über den ganzen
// Bestand (idempotent). Jede Anweisung wird einzeln protokolliert — schlägt eine fehl
// (etwa am Append-Only-Trigger auf audit_log, den es nur auf manchen Altbeständen
// gibt), bricht das die übrigen NICHT ab; das ist der Unterschied zur Tx des Purge.
func (s *Scheduler) bereinigeAnonymisierteSchuelerSpuren(ctx context.Context) {
	rows, err := s.db.Query(ctx, `SELECT id::text FROM schueler WHERE anonymized_at IS NOT NULL`)
	if err != nil {
		log.Printf("Scheduler GDPR Anonymize: anonymisierte Schüler konnten nicht gelesen werden: %v", err)
		return
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			log.Printf("Scheduler GDPR Anonymize: Schüler-ID lesen: %v", err)
			return
		}
		ids = append(ids, id)
	}
	if rows.Err() != nil {
		log.Printf("Scheduler GDPR Anonymize: anonymisierte Schüler konnten nicht gelesen werden: %v", rows.Err())
		return
	}
	if len(ids) == 0 {
		return
	}

	for _, st := range repository.SpurTilgungen() {
		if n, err := st.Exec(ctx, s.db, ids, "DSGVO-Anonymisierung"); err != nil {
			log.Printf("Scheduler GDPR Anonymize: Spur %s konnte nicht getilgt werden: %v", st.Beschreibung, err)
		} else if n > 0 {
			log.Printf("Scheduler GDPR Anonymize: Spur %s — %d Zeilen bereinigt.", st.Beschreibung, n)
		}
	}
}
