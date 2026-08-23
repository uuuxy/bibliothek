package jobs

// cron_audit_retention.go — auch Protokolle brauchen ein „wie lange".
//
// audit_log (Datensatz-Audit) und audit_logs (Admin-Aktionen, inkl. IP-Adressen)
// wuchsen bis zum 16.08.2026 unbegrenzt. Das ist doppelt falsch: Speicher — und
// DSGVO Art. 5 (Speicherbegrenzung), denn IP-Adressen und Bearbeiter-Bezüge sind
// personenbezogen. Der Job löscht nächtlich, was älter ist als die eingestellte
// Frist, und protokolliert die Löschung selbst als eine Zeile (Meta-Audit).
//
// Die Frist kommt aus system_einstellungen (audit_aufbewahrung_monate), Vorgabe
// 24 Monate. Untergrenze 6: Eine versehentliche 0 in der Einstellung darf nicht
// das komplette Protokoll wegräumen — Revisionsfähigkeit (wer hat die Gebühr
// storniert?) ist der Zweck der Tabellen.

import (
	"context"
	"log"
	"time"

	"bibliothek/repository"
)

// RunAuditAufbewahrung löscht Audit-Einträge jenseits der Aufbewahrungsfrist.
func (s *Scheduler) RunAuditAufbewahrung() {
	ctx, abbrechen := context.WithTimeout(context.Background(), 2*time.Minute)
	defer abbrechen()

	// Frist und Bedingung kommen aus repository/ — dieselbe Zeile, dieselbe Untergrenze,
	// die auch der Wächter der Selbstprüfung liest.
	monate := repository.NewBetriebszustandRepository(s.db).AuditAufbewahrungMonate(ctx)

	geloeschtDatensatz, err := s.loescheAeltereAls(ctx, "audit_log",
		repository.PredikatAuditLog(monate, repository.KulanzJob))
	if err != nil {
		log.Printf("Audit-Aufbewahrung: audit_log: %v", err)
		return
	}
	geloeschtAdmin, err := s.loescheAeltereAls(ctx, "audit_logs",
		repository.PredikatAuditLogs(monate, repository.KulanzJob))
	if err != nil {
		log.Printf("Audit-Aufbewahrung: audit_logs: %v", err)
		return
	}

	if geloeschtDatensatz == 0 && geloeschtAdmin == 0 {
		return // nichts zu tun, nichts zu protokollieren
	}

	// Meta-Audit: Die Löschung hinterlässt genau EINE Zeile mit den Zahlen — sonst
	// sähe eine spätere Prüfung nur ein Protokoll mit unerklärlicher Vorderkante.
	if err := s.auditRepo.LogSystemAktion(ctx, "audit_log", "RETENTION",
		"Aufbewahrungsfrist angewendet", map[string]any{
			"frist_monate":        monate,
			"geloescht_audit_log": geloeschtDatensatz,
			"geloescht_admin_log": geloeschtAdmin,
		}); err != nil {
		log.Printf("Audit-Aufbewahrung: Meta-Eintrag: %v", err)
	}
	log.Printf("Audit-Aufbewahrung: %d + %d Einträge älter als %d Monate entfernt",
		geloeschtDatensatz, geloeschtAdmin, monate)
}

func (s *Scheduler) loescheAeltereAls(ctx context.Context, tabelle string, b repository.Loeschbedingung) (int64, error) {
	tag, err := s.db.Exec(ctx, `DELETE FROM `+tabelle+` WHERE `+b.Where, b.Args...)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}
