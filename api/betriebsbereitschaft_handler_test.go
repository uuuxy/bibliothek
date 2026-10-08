package api

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
