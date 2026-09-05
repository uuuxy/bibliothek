package lmfplan

import (
	"testing"
	"time"
)

func tag(s string) time.Time {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		panic(err)
	}
	return t
}

// Der echte Plan 2026: Donnerstag 11.06. ab 3. Stunde, 6 Stunden je Tag. Zeile 5 muss
// am Freitag in der 1. Stunde liegen, Zeile 11 am Montag — das Wochenende fällt aus.
func TestVerteile_LaeuftUeberTageUndWochenende(t *testing.T) {
	r := Rahmen{ErsterTag: tag("2026-06-11"), Startstunde: 3, StundenJeTag: 6}
	p := Verteile(r, 12, Schultage(nil))
	erwartet := []Platz{
		{tag("2026-06-11"), 3}, {tag("2026-06-11"), 4}, {tag("2026-06-11"), 5}, {tag("2026-06-11"), 6},
		{tag("2026-06-12"), 1}, {tag("2026-06-12"), 2}, {tag("2026-06-12"), 3}, {tag("2026-06-12"), 4},
		{tag("2026-06-12"), 5}, {tag("2026-06-12"), 6},
		{tag("2026-06-15"), 1}, {tag("2026-06-15"), 2},
	}
	if len(p) != len(erwartet) {
		t.Fatalf("%d Plätze, erwartet %d", len(p), len(erwartet))
	}
	for i := range erwartet {
		if !p[i].Datum.Equal(erwartet[i].Datum) || p[i].Stunde != erwartet[i].Stunde {
			t.Errorf("Zeile %d: %s/%d, erwartet %s/%d", i+1, p[i].Datum.Format("2006-01-02"), p[i].Stunde,
				erwartet[i].Datum.Format("2006-01-02"), erwartet[i].Stunde)
		}
	}
}

func TestVerteile_FerienUndWochenendeAlsErsterTag(t *testing.T) {
	frei := []Zeitraum{{Von: tag("2026-06-15"), Bis: tag("2026-06-16")}}       // Mo/Di frei
	r := Rahmen{ErsterTag: tag("2026-06-13"), Startstunde: 1, StundenJeTag: 2} // Samstag
	p := Verteile(r, 3, Schultage(frei))
	if !p[0].Datum.Equal(tag("2026-06-17")) || p[0].Stunde != 1 {
		t.Errorf("erster Platz %s/%d, erwartet Mittwoch 17.06. 1. Std.", p[0].Datum.Format("2006-01-02"), p[0].Stunde)
	}
	if !p[2].Datum.Equal(tag("2026-06-18")) || p[2].Stunde != 1 {
		t.Errorf("dritter Platz %s/%d, erwartet Donnerstag 18.06. 1. Std.", p[2].Datum.Format("2006-01-02"), p[2].Stunde)
	}
}

func TestVerteile_StartstundeHinterTagesende(t *testing.T) {
	r := Rahmen{ErsterTag: tag("2026-06-11"), Startstunde: 7, StundenJeTag: 6}
	p := Verteile(r, 1, Schultage(nil))
	if !p[0].Datum.Equal(tag("2026-06-12")) || p[0].Stunde != 1 {
		t.Errorf("Startstunde 7 bei 6 Stunden: %s/%d, erwartet Freitag 1. Std.", p[0].Datum.Format("2006-01-02"), p[0].Stunde)
	}
	if got := Verteile(r, 0, Schultage(nil)); len(got) != 0 {
		t.Errorf("ohne Zeilen keine Plätze, waren %d", len(got))
	}
}

// Ein Zeitraum ohne Ende („bis 9999") darf den Server nicht in eine Endlosschleife
// schicken — nach einem Jahr Suche gilt der Kalender.
func TestVerteile_EndloseFerienEndenTrotzdem(t *testing.T) {
	frei := []Zeitraum{{Von: tag("2000-01-01"), Bis: tag("9999-12-31")}}
	p := Verteile(Rahmen{ErsterTag: tag("2026-06-11"), Startstunde: 1, StundenJeTag: 6}, 2, Schultage(frei))
	if len(p) != 2 {
		t.Fatalf("%d Plätze", len(p))
	}
}
