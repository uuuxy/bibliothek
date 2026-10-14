package bereitschaft

import (
	"slices"
	"strings"
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

// Die Prüfung der Klassen unterscheidet „nicht erhoben" von „alles verbunden" und nennt jede
// gerissene Verbindung, statt sie still zu schlucken.
func TestKlassenDrift(t *testing.T) {
	t.Run("nicht erhoben ist eine Warnung, kein OK", func(t *testing.T) {
		l := lageEingerichtet()
		l.KlassenOhneLehrkraft = nil
		b := pruefeKlassenDrift(l)
		if b.Stufe != StufeWarnung || !strings.Contains(b.Befund, "nicht geprüft") {
			t.Errorf("nil muss als 'nicht geprüft' warnen, got %+v", b)
		}
	})
	t.Run("Klasse ohne Lehrkraft wird benannt", func(t *testing.T) {
		l := lageEingerichtet()
		l.KlassenOhneLehrkraft = []string{"06a"}
		b := pruefeKlassenDrift(l)
		if b.Stufe != StufeWarnung || !strings.Contains(b.Befund, "06a") {
			t.Errorf("06a muss in der Warnung stehen, got %+v", b)
		}
		if b.Abhilfe == "" || b.Folge == "" {
			t.Error("Warnung ohne Folge/Abhilfe landet auf einem Zettel statt in der Zuordnung")
		}
	})
	t.Run("verwaiste Buecherliste wird benannt", func(t *testing.T) {
		l := lageEingerichtet()
		l.VerwaisteBuecherliste = []string{"7x"}
		if b := pruefeKlassenDrift(l); b.Stufe != StufeWarnung || !strings.Contains(b.Befund, "7x") {
			t.Errorf("7x muss in der Warnung stehen, got %+v", b)
		}
	})
}
