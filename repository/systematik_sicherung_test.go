package repository

import "testing"

// TestKuerzeRunen sichert die Zeichen-Semantik der Kürzel-Kappung: VARCHAR zählt
// Zeichen, nicht Bytes — ein mittig zerschnittener Umlaut wäre zudem kein gültiges Kürzel.
func TestKuerzeRunen(t *testing.T) {
	faelle := []struct {
		eingabe  string
		max      int
		erwartet string
	}{
		{"Deutsch", 50, "Deutsch"},
		{"Gesellschaftswissenschaften", 10, "Gesellscha"},
		{"ÄÖÜäöüß", 3, "ÄÖÜ"},
		{"", 5, ""},
	}
	for _, f := range faelle {
		if got := kuerzeRunen(f.eingabe, f.max); got != f.erwartet {
			t.Errorf("kuerzeRunen(%q, %d) = %q, erwartet %q", f.eingabe, f.max, got, f.erwartet)
		}
	}
}
