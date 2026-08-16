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
	"strconv"
	"time"
)

const (
	auditAufbewahrungSchluessel = "audit_aufbewahrung_monate"
	auditAufbewahrungVorgabe    = 24
	auditAufbewahrungMinimum    = 6
)

// RunAuditAufbewahrung löscht Audit-Einträge jenseits der Aufbewahrungsfrist.
func (s *Scheduler) RunAuditAufbewahrung() {
	ctx, abbrechen := context.WithTimeout(context.Background(), 2*time.Minute)
	defer abbrechen()

	monate := s.ladeAufbewahrungsMonate(ctx)

	geloeschtDatensatz, err := s.loescheAeltereAls(ctx,
		`DELETE FROM audit_log WHERE timestamp < now() - make_interval(months => $1)`, monate)
	if err != nil {
		log.Printf("Audit-Aufbewahrung: audit_log: %v", err)
		return
	}
	geloeschtAdmin, err := s.loescheAeltereAls(ctx,
		`DELETE FROM audit_logs WHERE zeitstempel < now() - make_interval(months => $1)`, monate)
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

// ladeAufbewahrungsMonate liest die Frist aus den Einstellungen (Vorgabe 24,
// Untergrenze 6 — Begründung im Dateikopf).
func (s *Scheduler) ladeAufbewahrungsMonate(ctx context.Context) int {
	var wert string
	err := s.db.QueryRow(ctx,
		`SELECT wert FROM system_einstellungen WHERE schluessel = $1`,
		auditAufbewahrungSchluessel).Scan(&wert)
	if err != nil {
		return auditAufbewahrungVorgabe
	}
	monate, err := strconv.Atoi(wert)
	if err != nil || monate < auditAufbewahrungMinimum {
		return auditAufbewahrungVorgabe
	}
	return monate
}

func (s *Scheduler) loescheAeltereAls(ctx context.Context, query string, monate int) (int64, error) {
	tag, err := s.db.Exec(ctx, query, monate)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}
