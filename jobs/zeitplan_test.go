package jobs

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/robfig/cron/v3"
)

// Die Nacht hat eine Reihenfolge: Die Sicherung läuft vor der Aufbewahrung des Protokolls und
// enthält so die Einträge, die danach gelöscht werden; die Restore-Probe läuft nach beiden und
// prüft die Sicherung derselben Nacht.
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
	aufbewahrung := naechste(zeitProtokollAufbewahrung, tag)

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

// Der Zeitplan rechnet in UTC: In der Ortszeit gibt es 02:30 Uhr in der Nacht der Umstellung
// auf Sommerzeit nicht, und die Sicherung fiele aus.
func TestZeitplan_RechnetInUTC(t *testing.T) {
	if ort := NewZeitplan(nil, nil).cron.Location(); ort != time.UTC {
		t.Errorf("der Zeitplan rechnet in %s, erwartet UTC", ort)
	}
}

// Ein Panic in einem Lauf des Zeitplans beendet nur diesen Lauf. Die Bibliothek fängt es nicht;
// ohne die Hülle käme es hier durch und beendete im Betrieb den Server.
func TestZeitplan_PanicBeendetNurDenLauf(t *testing.T) {
	var protokoll bytes.Buffer
	vorher := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&protokoll, nil)))
	t.Cleanup(func() { slog.SetDefault(vorher) })

	z := NewZeitplan(nil, nil)
	if _, err := z.cron.AddFunc("@every 1h", func() { panic("simulierter Absturz") }); err != nil {
		t.Fatalf("AddFunc: %v", err)
	}
	func() {
		defer func() {
			if p := recover(); p != nil {
				t.Errorf("das Panic des Laufs kam durch: %v", p)
			}
		}()
		z.cron.Entries()[0].WrappedJob.Run()
	}()
	if !strings.Contains(protokoll.String(), "level=ERROR") || !strings.Contains(protokoll.String(), "goroutine=zeitplan") {
		t.Errorf("kein Fehler im Log zum Panic des Laufs:\n%s", protokoll.String())
	}
}
