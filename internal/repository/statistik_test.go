package repository

import (
	"strings"
	"testing"
)

func TestBestandsFilterBedingung(t *testing.T) {
	cases := []struct {
		in, wantFragment, wantName string
	}{
		// Seit Migration 093 ist das Prädikat die Spalte selbst — kein abgeleitetes
		// Textmuster mehr, das Go und SQL getrennt formulieren könnten.
		{"lmf", "AND t.ist_lernmittel", "lmf"},
		{"freihand", "AND NOT t.ist_lernmittel", "freihand"},
		{"", "", "alle"},
		{"kaputt", "", "alle"}, // unbekannte Werte fallen sicher auf Gesamtbestand zurück
	}
	for _, c := range cases {
		frag, name := BestandsFilterBedingung(c.in)
		if frag != c.wantFragment || name != c.wantName {
			t.Errorf("BestandsFilterBedingung(%q) = (%q, %q), erwartet (%q, %q)", c.in, frag, name, c.wantFragment, c.wantName)
		}
	}
}

// Der Filter „Schuljahr“ trägt den Beginn des Schuljahres als festes Datum aus der Zeitzone der
// Schule (schulzeit.SchuljahrBeginn), nicht CURRENT_DATE in der Zeitzone der Sitzung (UTC).
func TestAusleihZeitraum_SchuljahrAusDerSchulzeitzone(t *testing.T) {
	if f := AusleihZeitraumBedingung("schuljahr"); !strings.Contains(f, "'::timestamp AT TIME ZONE 'Europe/Berlin'") || strings.Contains(f, "CURRENT_DATE") {
		t.Errorf("Schuljahr-Filter muss ein festes Datum aus der Schulzeitzone tragen: %q", f)
	}
}
