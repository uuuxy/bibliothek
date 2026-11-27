package ordnung

import (
	"slices"
	"testing"
)

// Umlaute bei ihrem Grundbuchstaben, Zahlen nach ihrer Größe, groß und klein ohne Trennung;
// gleiche Texte in der Reihenfolge, in der sie kamen.
func TestSortiere_WieEinDeutschesRegister(t *testing.T) {
	for _, f := range []struct {
		name           string
		rein, erwartet []string
	}{
		{"Umlaute", []string{"Ökologie", "Zoologie", "Biologie", "Ägypten"}, []string{"Ägypten", "Biologie", "Ökologie", "Zoologie"}},
		{"Zahlen", []string{"Band 10", "Band 2", "Band 0 Einführung"}, []string{"Band 0 Einführung", "Band 2", "Band 10"}},
		{"groß und klein", []string{"cornelsen", "Auer", "Buchner"}, []string{"Auer", "Buchner", "cornelsen"}},
	} {
		t.Run(f.name, func(t *testing.T) {
			liste := slices.Clone(f.rein)
			Sortiere(liste, func(s string) string { return s })
			if !slices.Equal(liste, f.erwartet) {
				t.Errorf("Sortiere(%q) = %q, erwartet %q", f.rein, liste, f.erwartet)
			}
		})
	}

	type lieferant struct{ name, kennung string }
	liste := []lieferant{{"Klett", "2"}, {"Auer", "1"}, {"Klett", "1"}}
	Sortiere(liste, func(l lieferant) string { return l.name })
	if erwartet := []lieferant{{"Auer", "1"}, {"Klett", "2"}, {"Klett", "1"}}; !slices.Equal(liste, erwartet) {
		t.Errorf("gleiche Texte: %v, erwartet %v in der Reihenfolge, in der sie kamen", liste, erwartet)
	}
}
