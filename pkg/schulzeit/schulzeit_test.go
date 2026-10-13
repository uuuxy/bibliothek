package schulzeit

import (
	"testing"
	"time"
)

// Die Zeitzone des Servers steht auf Pacific/Midway (UTC-11) und Pacific/Kiritimati (UTC+14):
// Dort ist über weite Teile des Tages ein anderer Kalendertag als in Berlin. Heraus kommt
// trotzdem der Berliner Tag; sonst trüge ein Schadensbescheid das falsche Datum und seine
// Zahlungsfrist einen Tag zu wenig.
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

// Das Ende des Tages ist für das ganze Programm hier festgelegt; internal/service ruft diese
// Funktion.
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

// Ein Datum wird als Mitternacht in der Zone der Schule gelesen, im Sommer wie im Winter; was
// kein Datum der Form JJJJ-MM-TT ist, ergibt einen Fehler.
func TestKalendertag(t *testing.T) {
	for datum, utc := range map[string]string{
		"2027-06-29": "2027-06-28T22:00:00Z", // Sommerzeit: zwei Stunden vor UTC
		"2027-01-15": "2027-01-14T23:00:00Z", // Winterzeit: eine Stunde vor UTC
	} {
		tag, err := Kalendertag(datum)
		if err != nil {
			t.Fatalf("%s: %v", datum, err)
		}
		if ist := tag.UTC().Format(time.RFC3339); ist != utc {
			t.Errorf("Kalendertag(%q) ist in UTC %s, erwartet %s", datum, ist, utc)
		}
		if tag.Format(time.DateOnly) != datum || tag.Location() != Zone() {
			t.Errorf("Kalendertag(%q) = %s in %s, erwartet denselben Tag in der Zone der Schule", datum, tag, tag.Location())
		}
	}
	for _, kein := range []string{"", "29.06.2027", "2027-6-29", "2027-06-29T10:00:00Z", "2027-02-30"} {
		if tag, err := Kalendertag(kein); err == nil {
			t.Errorf("Kalendertag(%q) = %s ohne Fehler", kein, tag)
		}
	}
}
