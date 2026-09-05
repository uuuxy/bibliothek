package lmfplan

import (
	"testing"
	"time"
)

// verteile ist VerteileMit ohne feste Plätze — die Form, die die meisten Fälle brauchen.
func verteile(r Rahmen, n int, istSchultag func(time.Time) bool) []Platz {
	return VerteileMit(r, make([]*Platz, n), istSchultag)
}

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
	p := verteile(r, 12, Schultage(nil))
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
	p := verteile(r, 3, Schultage(frei))
	if !p[0].Datum.Equal(tag("2026-06-17")) || p[0].Stunde != 1 {
		t.Errorf("erster Platz %s/%d, erwartet Mittwoch 17.06. 1. Std.", p[0].Datum.Format("2006-01-02"), p[0].Stunde)
	}
	if !p[2].Datum.Equal(tag("2026-06-18")) || p[2].Stunde != 1 {
		t.Errorf("dritter Platz %s/%d, erwartet Donnerstag 18.06. 1. Std.", p[2].Datum.Format("2006-01-02"), p[2].Stunde)
	}
}

func TestVerteile_StartstundeHinterTagesende(t *testing.T) {
	r := Rahmen{ErsterTag: tag("2026-06-11"), Startstunde: 7, StundenJeTag: 6}
	p := verteile(r, 1, Schultage(nil))
	if !p[0].Datum.Equal(tag("2026-06-12")) || p[0].Stunde != 1 {
		t.Errorf("Startstunde 7 bei 6 Stunden: %s/%d, erwartet Freitag 1. Std.", p[0].Datum.Format("2006-01-02"), p[0].Stunde)
	}
	if got := verteile(r, 0, Schultage(nil)); len(got) != 0 {
		t.Errorf("ohne Zeilen keine Plätze, waren %d", len(got))
	}
}

// Ein Zeitraum ohne Ende („bis 9999") darf den Server nicht in eine Endlosschleife
// schicken — nach einem Jahr Suche gilt der Kalender.
func TestVerteile_EndloseFerienEndenTrotzdem(t *testing.T) {
	frei := []Zeitraum{{Von: tag("2000-01-01"), Bis: tag("9999-12-31")}}
	p := verteile(Rahmen{ErsterTag: tag("2026-06-11"), Startstunde: 1, StundenJeTag: 6}, 2, Schultage(frei))
	if len(p) != 2 {
		t.Fatalf("%d Plätze", len(p))
	}
}

// Fronleichnam (Donnerstag 04.06.2026) ist in Hessen frei — ohne dass jemand ihn
// einträgt. Mittwoch, dann Freitag; und die Ausfall-Liste nennt den Grund.
func TestVerteile_FeiertagFaelltAus(t *testing.T) {
	r := Rahmen{ErsterTag: tag("2026-06-03"), Startstunde: 1, StundenJeTag: 1}
	p := verteile(r, 2, Schultage(nil))
	if !p[0].Datum.Equal(tag("2026-06-03")) || !p[1].Datum.Equal(tag("2026-06-05")) {
		t.Errorf("Plätze %s und %s, erwartet 03.06. und 05.06.", p[0].Datum.Format("2006-01-02"), p[1].Datum.Format("2006-01-02"))
	}
	frei := []Zeitraum{{Von: tag("2026-06-05"), Bis: tag("2026-06-05"), Name: "Brückentag"}}
	a := Ausfaelle(tag("2026-06-01"), tag("2026-06-08"), frei)
	if len(a) != 2 || a[0].Grund != "Fronleichnam" || a[1].Grund != "Brückentag" ||
		!a[0].Datum.Equal(tag("2026-06-04")) || !a[1].Datum.Equal(tag("2026-06-05")) {
		t.Errorf("Ausfälle: %+v", a)
	}
}

// Die Klasse mit dem Ausflug: Zeile 2 bekommt Freitag 12.06. 2. Stunde von Hand. Die
// übrigen fließen weiter — und Zeile 4, die sonst genau auf diesen Platz käme, rückt
// eine Stunde weiter. Der feste Platz darf auch auf einem Tag liegen, der kein
// Schultag ist (Samstag): Wer ihn setzt, weiß es.
func TestVerteileMit_FesterPlatzWirdAusgelassen(t *testing.T) {
	r := Rahmen{ErsterTag: tag("2026-06-11"), Startstunde: 6, StundenJeTag: 6} // Do, letzte Stunde
	fest := []*Platz{nil, {Datum: tag("2026-06-12"), Stunde: 2}, nil, nil, {Datum: tag("2026-06-13"), Stunde: 1}}
	p := VerteileMit(r, fest, Schultage(nil))
	erwartet := []Platz{
		{tag("2026-06-11"), 6}, // Zeile 1: Donnerstag 6.
		{tag("2026-06-12"), 2}, // Zeile 2: fest
		{tag("2026-06-12"), 1}, // Zeile 3: Freitag 1.
		{tag("2026-06-12"), 3}, // Zeile 4: Freitag 3. — die 2. ist belegt
		{tag("2026-06-13"), 1}, // Zeile 5: fest, Samstag
	}
	for i := range erwartet {
		if !p[i].Datum.Equal(erwartet[i].Datum) || p[i].Stunde != erwartet[i].Stunde {
			t.Errorf("Zeile %d: %s/%d, erwartet %s/%d", i+1, p[i].Datum.Format("2006-01-02"), p[i].Stunde,
				erwartet[i].Datum.Format("2006-01-02"), erwartet[i].Stunde)
		}
	}
}
