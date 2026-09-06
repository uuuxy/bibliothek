package repository

import (
	"reflect"
	"testing"
)

// Die Einstellung der Eingangsjahrgänge wird von Hand getippt („5, 7"); der Leser muss
// Trennzeichen, führende Nullen und Unsinn wegstecken — und darf NIE leer werden, sonst
// bekäme nach den Ferien niemand Bücher und vor den Ferien gäbe niemand „nur zurück".
func TestEingangsjahrgaengeAus(t *testing.T) {
	faelle := []struct {
		in   string
		soll []int
	}{
		{"5, 7", []int{5, 7}},
		{"7;5", []int{5, 7}},
		{"05 07 7", []int{5, 7}},
		{"5, 7, 11", []int{5, 7, 11}},
		{"", []int{5, 7}},
		{"abc, 0, 14", []int{5, 7}},
		{"9", []int{9}},
	}
	for _, f := range faelle {
		if ist := EingangsjahrgaengeAus(f.in); !reflect.DeepEqual(ist, f.soll) {
			t.Errorf("EingangsjahrgaengeAus(%q) = %v, erwartet %v", f.in, ist, f.soll)
		}
	}
}

// Rasterdurchgang 06.09.2026 (Frage 5, stille Fehler): „8/9" mit Schrägstrich — so wie
// Klassen geschrieben werden — wurde gespeichert, angezeigt und beim Lesen still auf die
// Vorgabe „5, 7" zurückgeworfen. Der Ausgabe-Plan schlug weiter 5er und 7er vor, und
// „nur Rückgabe" traf die falschen Klassen. Jetzt lehnt der Schreibpfad ab.
func TestNormalisiereEingangsjahrgaenge(t *testing.T) {
	gut := map[string]string{
		"5, 7":    "5, 7",
		"7;5":     "5, 7",
		"05 07":   "5, 7",
		"5,7,5":   "5, 7",
		"11":      "11",
		"":        "",
		"   ":     "",
		"7, 5, 9": "5, 7, 9",
	}
	for ein, will := range gut {
		got, err := NormalisiereEingangsjahrgaenge(ein)
		if err != nil || got != will {
			t.Errorf("%q → %q, %v (erwartet %q)", ein, got, err, will)
		}
	}
	for _, schlecht := range []string{"8/9", "5-7", "abc", "0", "14", "5, x"} {
		if _, err := NormalisiereEingangsjahrgaenge(schlecht); err == nil {
			t.Errorf("%q muss abgelehnt werden — sonst rechnet der Plan mit der Vorgabe weiter", schlecht)
		}
	}
}
