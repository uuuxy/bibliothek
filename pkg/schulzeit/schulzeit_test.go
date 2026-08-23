package schulzeit

import (
	"testing"
	"time"
)

// TestJetztIstSchulzeitUnabhaengigVonDerServerzeitzone ist das Gate zum Befund vom
// 23.08.2026: Die gedruckten Dokumente nahmen die Container-Zeit (im Image UTC).
//
// Der Test stellt die Server-Zeitzone bewusst auf Pacific/Midway (UTC-11) — dort ist
// über weite Teile des Tages ein ANDERER Kalendertag als in Berlin. Was hier
// herauskommt, muss trotzdem der Berliner Tag sein, sonst trägt ein Schadensbescheid
// das falsche Datum und seine Zahlungsfrist einen Tag zu wenig.
func TestJetztIstSchulzeitUnabhaengigVonDerServerzeitzone(t *testing.T) {
	for _, tz := range []string{"UTC", "Pacific/Midway", "Pacific/Kiritimati"} {
		t.Setenv("TZ", tz)

		jetzt := Jetzt()
		berlin := time.Now().In(Zone())

		if jetzt.Format("2006-01-02") != berlin.Format("2006-01-02") {
			t.Errorf("TZ=%s: Jetzt() liefert den Tag %s, in der Schule ist %s",
				tz, jetzt.Format("2006-01-02"), berlin.Format("2006-01-02"))
		}
		if name, _ := jetzt.Zone(); name != "CET" && name != "CEST" {
			t.Errorf("TZ=%s: Jetzt() trägt die Zone %q statt CET/CEST", tz, name)
		}
	}
}

// TestTagesEndeFaelltAufDenBerlinerKalendertag hält die Definition fest, die das ganze
// System benutzt (internal/service reicht sie hierher durch).
func TestTagesEndeFaelltAufDenBerlinerKalendertag(t *testing.T) {
	// 23:30 UTC am 15. Juni ist in Berlin bereits der 16. Juni, 01:30 (Sommerzeit).
	spaet := time.Date(2026, 6, 15, 23, 30, 0, 0, time.UTC)

	ende := TagesEnde(spaet)

	if got := ende.Format("2006-01-02 15:04:05"); got != "2026-06-16 23:59:59" {
		t.Errorf("TagesEnde(23:30 UTC am 15.06.) = %s, erwartet 2026-06-16 23:59:59 (Berliner Tag)", got)
	}
	if ende.Location() != Zone() {
		t.Errorf("TagesEnde liefert die Zone %v statt der Schulzeitzone", ende.Location())
	}
}
