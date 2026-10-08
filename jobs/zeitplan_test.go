package jobs

import (
	"testing"
	"time"

	"github.com/robfig/cron/v3"
)

// Die Nacht hat eine Reihenfolge: Die Sicherung läuft vor der Audit-Aufbewahrung und enthält
// so die Einträge, die danach gelöscht werden; die Restore-Probe läuft nach beiden und prüft
// die Sicherung derselben Nacht.
func TestZeitplan_SicherungVorAufbewahrungVorProbe(t *testing.T) {
	naechste := func(zeit string, ab time.Time) time.Time {
		t.Helper()
		plan, err := cron.ParseStandard(zeit)
		if err != nil {
			t.Fatalf("%q ist kein gültiger Cron-Ausdruck: %v", zeit, err)
		}
		return plan.Next(ab)
	}

	// Gemessen wird an dem Tag, an dem die Probe das nächste Mal läuft.
	probe := naechste(zeitRestoreProbe, time.Date(2026, time.October, 8, 0, 0, 0, 0, time.UTC))
	tag := time.Date(probe.Year(), probe.Month(), probe.Day(), 0, 0, 0, 0, time.UTC)
	sicherung := naechste(zeitSicherung, tag)
	aufbewahrung := naechste(zeitAuditAufbewahrung, tag)

	const uhr = "Mon 02.01. 15:04"
	if !sicherung.Before(aufbewahrung) {
		t.Errorf("die Audit-Aufbewahrung (%s) läuft nicht nach der Sicherung (%s)",
			aufbewahrung.Format(uhr), sicherung.Format(uhr))
	}
	if !aufbewahrung.Before(probe) {
		t.Errorf("die Restore-Probe (%s) läuft nicht nach der Audit-Aufbewahrung (%s)",
			probe.Format(uhr), aufbewahrung.Format(uhr))
	}
}

// Der Zeitplan rechnet in UTC: In der Ortszeit gibt es 02:30 Uhr in der Nacht der
// Sommerzeit-Umstellung nicht, und die Sicherung fiele aus.
func TestZeitplan_RechnetInUTC(t *testing.T) {
	if ort := NewScheduler(nil, nil).cron.Location(); ort != time.UTC {
		t.Errorf("der Zeitplan rechnet in %s, erwartet UTC", ort)
	}
}
