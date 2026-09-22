package inventur

import (
	"testing"
)

func TestInferSubjectFromTitle(t *testing.T) {
	tests := []struct {
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

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := inferSubjectFromTitle(tt.title)
			if result != tt.expected {
				t.Errorf("inferSubjectFromTitle(%q) = %q; want %q", tt.title, result, tt.expected)
			}
		})
	}
}
