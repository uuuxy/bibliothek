package bestellung

import (
	"strings"
	"testing"
	"time"
)

// Die Überschrift des Berichts: Ein mitgegebener Titel geht vor, sonst folgt sie aus Lieferant
// und Jahresansicht. Ist der Bericht auf einen Topf eingeengt, nennt ihn die Überschrift; ein
// Blatt ohne den Zusatz läse sich wie der Gesamtbericht und würde so abgelegt.
func TestBerichtTitelAbleiten(t *testing.T) {
	const lieferant = "00000000-0000-0000-0000-000000000562"
	for _, f := range []struct {
		name          string
		titel         string
		lieferantID   string
		jahresansicht bool
		mittel        string
		want          string
	}{
		{"ohne alles", "", "", false, "", "Bestellbericht"},
		{"ein Lieferant", "", lieferant, false, "", "Lieferantenabrechnung"},
		{"Jahresansicht", "", "", true, "", "Jahresbericht"},
		{"Lieferant geht vor Jahresansicht", "", lieferant, true, "", "Lieferantenabrechnung"},
		{"eigener Titel geht vor", "Abrechnung Herbst", lieferant, true, "", "Abrechnung Herbst"},
		{"Topf Land", "", "", false, "land", "Bestellbericht — Lernmittelfreiheit (Land)"},
		{"Topf Schulträger am Lieferanten", "", lieferant, false, "schultraeger", "Lieferantenabrechnung — Schülerbücherei (Schulträger)"},
		{"ohne Zuordnung", "", "", true, "ohne", "Jahresbericht — ohne Zuordnung"},
		{"Topf am eigenen Titel", "Abrechnung Herbst", "", false, "land", "Abrechnung Herbst — Lernmittelfreiheit (Land)"},
	} {
		if ist := berichtTitelAbleiten(f.titel, f.lieferantID, f.jahresansicht, f.mittel); ist != f.want {
			t.Errorf("%s: Überschrift %q, erwartet %q", f.name, ist, f.want)
		}
	}
}

// Beide Tage des Zeitraums sind Pflicht und stehen als JJJJ-MM-TT da. Die Meldung nennt den
// Parameter, an dem es liegt.
func TestParseBerichtZeitraum(t *testing.T) {
	von, bis, err := parseBerichtZeitraum("2026-09-01", "2026-09-30")
	if err != nil {
		t.Fatalf("gültiger Zeitraum: %v", err)
	}
	if !von.Equal(time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)) || !bis.Equal(time.Date(2026, 9, 30, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("Zeitraum %s bis %s, erwartet 01.09.2026 bis 30.09.2026", von, bis)
	}
	for _, f := range []struct{ von, bis, nennt string }{
		{"", "2026-09-30", "'von' und 'bis'"},
		{"2026-09-01", "", "'von' und 'bis'"},
		{"01.09.2026", "2026-09-30", "'von'"},
		{"2026-09-01", "2026-09-31", "'bis'"},
	} {
		_, _, err := parseBerichtZeitraum(f.von, f.bis)
		if err == nil {
			t.Errorf("von=%q bis=%q: kein Fehler", f.von, f.bis)
			continue
		}
		if !strings.Contains(err.Error(), f.nennt) {
			t.Errorf("von=%q bis=%q: die Meldung nennt %s nicht: %v", f.von, f.bis, f.nennt, err)
		}
	}
}
