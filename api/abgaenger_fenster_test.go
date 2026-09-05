package api

import (
	"testing"
	"time"

	"bibliothek/pkg/schulzeit"
)

// Die Saison der Abgängerliste an ihren Rändern — und in der Schulzeitzone, nicht in der
// des Prozesses: Am 30.04. um 23:30 Uhr Berlin ist in UTC noch der 30.04., am 31.07. um
// 23:30 UTC in Berlin schon der 01.08.
func TestAbgaengerFenster_Raender(t *testing.T) {
	zone := schulzeit.Zone()
	faelle := []struct {
		name  string
		wann  time.Time
		offen bool
	}{
		{"30. April, letzter Tag davor", time.Date(2026, time.April, 30, 12, 0, 0, 0, zone), false},
		{"1. Mai, erster Tag", time.Date(2026, time.May, 1, 0, 0, 1, 0, zone), true},
		{"Mitte Juni", time.Date(2026, time.June, 15, 9, 0, 0, 0, zone), true},
		{"31. Juli, letzter Tag", time.Date(2026, time.July, 31, 23, 59, 0, 0, zone), true},
		{"1. August, Schuljahresbeginn", time.Date(2026, time.August, 1, 0, 0, 1, 0, zone), false},
		{"Oktober", time.Date(2026, time.October, 5, 12, 0, 0, 0, zone), false},
		{"31. Juli 23:30 UTC ist in Berlin schon August", time.Date(2026, time.July, 31, 23, 30, 0, 0, time.UTC), false},
	}
	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			fenster := abgaengerFensterFuer(f.wann)
			if fenster.Offen != f.offen {
				t.Errorf("offen = %v, erwartet %v", fenster.Offen, f.offen)
			}
			if fenster.Von != "01.05." || fenster.Bis != "31.07." {
				t.Errorf("Anzeigegrenzen %q–%q, erwartet 01.05.–31.07.", fenster.Von, fenster.Bis)
			}
		})
	}
}

// Ohne gesetzte Uhr läuft der Server nach der Schulzeit — kein Test darf still eine
// Nulluhr (Jahr 1) bekommen, die immer „außerhalb" wäre.
func TestServerJetzt_OhneUhrIstSchulzeit(t *testing.T) {
	s := &Server{}
	if d := time.Since(s.jetzt()); d < 0 || d > time.Minute {
		t.Errorf("Server.jetzt() ohne Uhr weicht um %v von jetzt ab", d)
	}
	fest := time.Date(2026, time.June, 1, 8, 0, 0, 0, time.UTC)
	s.Uhr = func() time.Time { return fest }
	if !s.jetzt().Equal(fest) {
		t.Error("gesetzte Uhr wird nicht benutzt")
	}
}
