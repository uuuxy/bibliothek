package repository

import (
	"testing"
	"time"
)

// Der LUSD-Namensschlüssel ist die Normalform suchnorm (Migration 054), nicht lower+trim
// (Befund-Register, Entscheidung 1 vom 05.09.2026): „Anna Müller" von Hand angelegt und
// „Anna Mueller" im Export sind derselbe Mensch — bis dahin wurden daraus Neuanlage plus
// „nicht im Export". Sicher ist das, weil der Import einen doppelt belegten Schlüssel als
// mehrdeutig markiert und dann niemanden anfasst: Ein zu weiter Schlüssel fällt auf das
// alte Verhalten zurück, nie in eine falsche Zuordnung.
func TestLusdNamensSchluessel_IstDieSuchnormDerNamen(t *testing.T) {
	gleich := [][2][2]string{
		{{"Anna", "Müller"}, {"Anna", "Mueller"}},
		{{"Anna", "Müller"}, {"ANNA", "MÜLLER"}},
		{{"Ayşe", "Öztürk"}, {"Ayse", "Oeztuerk"}},
		{{"Anna Lena", "Müller"}, {"Anna  Lena ", " Müller"}}, // Mehrfach-Leerzeichen, Ränder
		{{"Đorđe", "Šimić"}, {"Dorde", "Simic"}},
	}
	for _, p := range gleich {
		a, b := LusdNamensSchluessel(p[0][0], p[0][1]), LusdNamensSchluessel(p[1][0], p[1][1])
		if a != b {
			t.Errorf("%v und %v müssen denselben Namensschlüssel haben: %q ≠ %q", p[0], p[1], a, b)
		}
	}

	// Verschiedene Menschen bleiben verschieden — auch Vor- und Nachname vertauscht.
	verschieden := [][2][2]string{
		{{"Anna", "Müller"}, {"Anne", "Müller"}},
		{{"Anna", "Müller"}, {"Müller", "Anna"}},
		{{"Anna-Lena", "Müller"}, {"Anna", "Müller"}},
	}
	for _, p := range verschieden {
		if LusdNamensSchluessel(p[0][0], p[0][1]) == LusdNamensSchluessel(p[1][0], p[1][1]) {
			t.Errorf("%v und %v dürfen nicht denselben Namensschlüssel haben", p[0], p[1])
		}
	}
}

// Dieselbe Normalform für den Name+Geburtsdatum-Schlüssel — zwei Stufen, EINE Regel.
func TestLusdSchluessel_NormalisiertWieDerNamensschluessel(t *testing.T) {
	geb := time.Date(2012, 3, 4, 0, 0, 0, 0, time.UTC)
	if a, b := LusdSchluessel("Anna", "Müller", &geb), LusdSchluessel("anna", "MUELLER", &geb); a != b {
		t.Errorf("Müller/MUELLER mit gleichem Geburtsdatum: %q ≠ %q", a, b)
	}
	andere := geb.AddDate(0, 0, 1)
	if LusdSchluessel("Anna", "Müller", &geb) == LusdSchluessel("Anna", "Müller", &andere) {
		t.Error("verschiedene Geburtsdaten müssen verschiedene Schlüssel geben")
	}
	if LusdSchluessel("Anna", "Müller", nil) != "" {
		t.Error("ohne Geburtsdatum gibt es keinen Name+Datum-Schlüssel")
	}
}
