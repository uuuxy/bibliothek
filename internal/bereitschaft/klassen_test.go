package bereitschaft

import (
	"slices"
	"testing"
)

// Eine Klassenleitung erwartet die Prüfung bis Jahrgang 10, in jeder Schreibweise der Schule.
// „70R1" steht für einen Tippfehler: keine lesbare Stufe, aber Schüler ohne Lehrkraft.
func TestMitKlassenleitung(t *testing.T) {
	klassen := []string{"05F1", "10G6", "10H1", "ET1", "E2", "Q3", "11", "12T5", "13T3", "ABG", "AUS", "70R1", ""}
	soll := []string{"05F1", "10G6", "10H1", "70R1"}
	if ist := mitKlassenleitung(klassen); !slices.Equal(ist, soll) {
		t.Errorf("mit Klassenleitung: %v, erwartet %v", ist, soll)
	}
}

// Die drei Listen der Lage, je mit eigenen Werten: Klassen bis Jahrgang 10 ohne Zuordnung,
// Zuordnungen ohne Schüler, Bücherlisten ohne Schüler. Die Oberstufe braucht keine Zuordnung.
func TestTrageKlassenEin(t *testing.T) {
	var l Lage
	l.TrageKlassenEin(
		[]string{"05F1", "07G2", "12T1"}, // Klassen mit aktiven Schülern
		[]string{"05F1", "09H9"},         // Zuordnungen der Klassenleitungen
		[]string{"07G2", "06F4", "08R8"}, // Bücherlisten
	)
	if !slices.Equal(l.KlassenOhneLehrkraft, []string{"07G2"}) {
		t.Errorf("Klassen ohne Zuordnung: %v, erwartet 07G2 (12T1 hat als Oberstufe keine Klassenleitung)", l.KlassenOhneLehrkraft)
	}
	if !slices.Equal(l.VerwaisteZuordnungen, []string{"09H9"}) {
		t.Errorf("Zuordnungen ohne Schüler: %v, erwartet 09H9", l.VerwaisteZuordnungen)
	}
	if !slices.Equal(l.VerwaisteBuecherliste, []string{"06F4", "08R8"}) {
		t.Errorf("Bücherlisten ohne Schüler: %v, erwartet 06F4 und 08R8", l.VerwaisteBuecherliste)
	}

	// Passt alles zusammen, sind die Listen leer und nicht nil: nil hieße „nicht erhoben", und
	// die Prüfung meldete eine Warnung statt „in Ordnung".
	var leer Lage
	leer.TrageKlassenEin([]string{"05F1"}, []string{"05F1"}, nil)
	if leer.KlassenOhneLehrkraft == nil || leer.VerwaisteZuordnungen == nil || leer.VerwaisteBuecherliste == nil {
		t.Errorf("ohne Abweichung trägt die Lage nil: %+v", leer)
	}
	if b := pruefeKlassenDrift(leer); b.Stufe != StufeOK {
		t.Errorf("ohne Abweichung: Stufe %s, erwartet %s (%s)", b.Stufe, StufeOK, b.Befund)
	}
	// Ohne den Aufruf bleibt es bei „nicht erhoben".
	if b := pruefeKlassenDrift(Lage{}); b.Stufe != StufeWarnung {
		t.Errorf("nicht erhoben: Stufe %s, erwartet %s", b.Stufe, StufeWarnung)
	}
}

// Die schärfste Stufe gewinnt, gleich an welcher Stelle sie steht; ohne Befund ist alles in
// Ordnung.
func TestGesamt(t *testing.T) {
	befunde := func(stufen ...string) []Befund {
		var b []Befund
		for _, s := range stufen {
			b = append(b, Befund{Stufe: s})
		}
		return b
	}
	for _, f := range []struct {
		stufen []string
		want   string
	}{
		{nil, StufeOK},
		{[]string{StufeOK, StufeOK}, StufeOK},
		{[]string{StufeOK, StufeWarnung, StufeOK}, StufeWarnung},
		{[]string{StufeWarnung, StufeKritisch, StufeOK}, StufeKritisch},
		{[]string{StufeKritisch, StufeWarnung}, StufeKritisch},
		{[]string{StufeOK, StufeKritisch}, StufeKritisch},
	} {
		if ist := Gesamt(befunde(f.stufen...)); ist != f.want {
			t.Errorf("Gesamt über %v: %s, erwartet %s", f.stufen, ist, f.want)
		}
	}
}
