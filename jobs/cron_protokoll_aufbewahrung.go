package jobs

// Auch das Protokoll braucht ein „wie lange".
//
// audit_log (Änderungen an Datensätzen) und audit_logs (Handlungen der Verwaltung, mit
// IP-Adressen) wüchsen sonst ohne Grenze. Das ist doppelt falsch: Speicher, und DSGVO Art. 5
// (Speicherbegrenzung), denn IP-Adressen und Bezüge auf Bearbeiter sind personenbezogen. Der Lauf
// löscht nachts, was älter ist als die eingestellte Frist, und protokolliert die Löschung selbst
// in einer Zeile.
//
// Die Frist kommt aus system_einstellungen (audit_aufbewahrung_monate), Vorgabe 24 Monate.
// Untergrenze 6: Eine versehentliche 0 in der Einstellung darf nicht das ganze Protokoll
// wegräumen, denn der Zweck der Tabellen ist, nachzuvollziehen, wer etwa eine Forderung storniert
// hat.

import (
	"context"
	"log"
	"time"

	"bibliothek/repository"
)

// fuehreProtokollAufbewahrungAus löscht die Einträge des Protokolls jenseits der Frist.
func (z *Zeitplan) fuehreProtokollAufbewahrungAus(ctx context.Context) {
	ctx, abbrechen := context.WithTimeout(ctx, 2*time.Minute)
	defer abbrechen()

	// Frist und Bedingung kommen aus repository/: derselbe Weg und dieselbe Untergrenze, die
	// auch der Wächter der Selbstprüfung nimmt, über die Einstellungen, weil die Frist in der
	// Oberfläche steht.
	einst, err := repository.NewEinstellungsAbfragen(z.pool).Lade(ctx)
	if err != nil {
		log.Printf("Audit-Aufbewahrung: Einstellungen nicht lesbar, Lauf übersprungen: %v", err)
		return
	}
	monate := repository.ProtokollAufbewahrungMonateOderVorgabe(einst.ProtokollAufbewahrungMonate)

	geloeschtDatensatz, err := repository.LoescheAltesProtokoll(ctx, z.pool, monate)
	if err != nil {
		log.Printf("Audit-Aufbewahrung: audit_log: %v", err)
		return
	}
	geloeschtAdmin, err := repository.LoescheAltesVerwaltungsProtokoll(ctx, z.pool, monate)
	if err != nil {
		log.Printf("Audit-Aufbewahrung: audit_logs: %v", err)
		return
	}

	if geloeschtDatensatz == 0 && geloeschtAdmin == 0 {
		return // nichts zu tun, nichts zu protokollieren
	}

	// Die Löschung hinterlässt genau eine Zeile mit den Zahlen, sonst sähe eine spätere Prüfung
	// nur ein Protokoll, das ohne Erklärung erst an einem bestimmten Tag beginnt.
	if err := z.protokollAbfragen.ProtokolliereSystemAktion(ctx, "audit_log", "RETENTION",
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
