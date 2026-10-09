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

	count, err := repository.AnonymisiereBearbeiterAlterAusleihen(ctx, s.db)
	if err != nil {
		log.Printf("Scheduler GDPR Anonymize: Error anonymizing operator IDs: %v", err)
		return
	}
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

	// Bei unlesbaren Einstellungen wird nicht anonymisiert. Die Vorgabe von 90 Tagen wäre
	// kein sicherer Rückfall: Eine kleinere Karenz wählt mehr Zeilen, und hat die Schule 365
	// Tage eingestellt, träfe ein einziger Lesefehler die Abgänger zwischen Tag 91 und 365.
	einst, err := repository.NewSystemSettingsRepository(s.db).GetSettings(ctx)
	if err != nil {
		log.Printf("Scheduler GDPR Anonymize: Einstellungen nicht lesbar, Lauf übersprungen: %v", err)
		return
	}
	karenzTage := repository.AbgaengerKarenzTageOderStandard(einst)
	// Anweisung und Bedingung stehen in repository/: dieselbe Bedingung, die die Selbstprüfung
	// als Rückstand zählt.
	count, err := repository.AnonymisiereSchueler(ctx, s.db, karenzTage)
	if err != nil {
		log.Printf("Scheduler GDPR Anonymize: Error anonymizing old students: %v", err)
		return
	}

	// Scheitert das Löschen der Fotos, läuft die Tilgung der Spuren trotzdem; die nächste
	// Nacht räumt die Fotos nach.
	if delErr := repository.LoescheFotosAnonymisierterSchueler(ctx, s.db); delErr != nil {
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
	ids, err := repository.AnonymisierteSchuelerIDs(ctx, s.db)
	if err != nil {
		log.Printf("Scheduler GDPR Anonymize: anonymisierte Schüler konnten nicht gelesen werden: %v", err)
		return
	}
	if len(ids) == 0 {
		return
	}

	for _, st := range repository.SpurTilgungen() {
		if n, err := st.Tilge(ctx, s.db, ids, "DSGVO-Anonymisierung"); err != nil {
			log.Printf("Scheduler GDPR Anonymize: Spur %s konnte nicht getilgt werden: %v", st.Beschreibung, err)
		} else if n > 0 {
			log.Printf("Scheduler GDPR Anonymize: Spur %s — %d Zeilen bereinigt.", st.Beschreibung, n)
		}
	}
}
