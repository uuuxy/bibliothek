package repository

import "testing"

// Seit die Frist der Schülerbücherei ein Tag ist (29.09.2026), nennen Selbstprüfung und
// Datenschutz-Auskunft die Eins; „1 Tage nach Rückgabe" stünde sonst in jeder Auskunft.
func TestTageMitZahl(t *testing.T) {
	for tage, want := range map[int]string{1: "1 Tag", 2: "2 Tage", 730: "730 Tage"} {
		if got := TageMitZahl(tage); got != want {
			t.Errorf("TageMitZahl(%d) = %q, erwartet %q", tage, got, want)
		}
	}
	if got := tageText(1); got != "1 Tag" {
		t.Errorf("tageText(1) = %q, erwartet \"1 Tag\"", got)
	}
}
