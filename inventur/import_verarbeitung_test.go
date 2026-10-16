package inventur

import (
	"testing"
)

func TestErrateFachAusTitel(t *testing.T) {
	faelle := []struct {
		name     string
		title    string
		expected string
	}{
		{"Exact match", "Deutsch", "Deutsch"},
		{"Case insensitive", "maThe", "Mathematik"},
		{"Whitespace trimming", "  Physik  ", "Physik"},
		{"Substring match", "Chemiebuch für Anfänger", "Chemie"},
		{"Another substring match", "Einführung in die Biologie", "Biologie"},
		{"Umlaut handling", "Französisch 1", "Französisch"},
		{"Alternative spelling", "Franzoesisch 2", "Französisch"},
		{"Multiple words keyword", "Natur und Technik 5", "Naturwissenschaften"},
		{"No match", "Roman und Erzählung", ""},
		{"Multiple matches (returns first rule matched)", "Deutsch und Englisch", "Deutsch"}, // Deutsch is before Englisch in rules
		{"Empty string", "", ""},
	}

	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			ergebnis := errateFachAusTitel(f.title)
			if ergebnis != f.expected {
				t.Errorf("errateFachAusTitel(%q) = %q; erwartet %q", f.title, ergebnis, f.expected)
			}
		})
	}
}
