package repository

import (
	"strings"
	"testing"
	"time"
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
			t.Errorf("BestandsFilterBedingung(%q) = (%q, %q), want (%q, %q)", c.in, frag, name, c.wantFragment, c.wantName)
		}
	}
}

// Das Schuljahr beginnt am 1. August — Juli zählt zum vorigen (Rasterdurchgang 02.09.2026:
// vorher rechnete CURRENT_DATE in der DB-Sitzungszone UTC).
func TestSchuljahresBeginn(t *testing.T) {
	berlin, err := time.LoadLocation("Europe/Berlin")
	if err != nil {
		t.Fatal(err)
	}
	faelle := map[time.Time]string{
		time.Date(2026, 8, 1, 0, 30, 0, 0, berlin):  "2026-08-01",
		time.Date(2026, 7, 31, 23, 0, 0, 0, berlin): "2025-08-01",
		time.Date(2027, 1, 1, 0, 30, 0, 0, berlin):  "2026-08-01",
	}
	for zeit, soll := range faelle {
		if got := schuljahresBeginn(zeit).Format("2006-01-02"); got != soll {
			t.Errorf("%s → %s, erwartet %s", zeit, got, soll)
		}
	}
	if f := AusleihZeitraumBedingung("schuljahr"); !strings.Contains(f, "'::date") || strings.Contains(f, "CURRENT_DATE") {
		t.Errorf("Schuljahr-Filter muss ein festes Datum aus der Schulzeitzone tragen: %q", f)
	}
}
