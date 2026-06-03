package inventur

import (
	"testing"
)

func TestInferGradeLevelFromTitle(t *testing.T) {
	tests := []struct {
		name     string
		title    string
		expected int
	}{
		{"Single digit at end", "Math 7", 7},
		{"Double digit at end", "Biology 10", 10},
		{"Maximum valid grade", "Something 13", 13},
		{"Minimum valid grade", "Level 1", 1},
		{"Grade out of upper bound", "Grade 14", 0},
		{"Grade out of lower bound", "Level 0", 0},
		{"Digit with suffix", "7th Grade", 7},
		{"Digit inside string", "English 5b", 5},
		{"Just the digit", "12", 12},
		{"No digit", "Something", 0},
		{"Multiple digits concatenated", "123", 0},
		{"Number out of bounds", "Math 101", 0},
		{"Digit at start", "9 Math", 9},
		{"Multiple valid digits (finds first)", "Math 7 and 8", 7},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := inferGradeLevelFromTitle(tt.title)
			if result != tt.expected {
				t.Errorf("inferGradeLevelFromTitle(%q) = %d; want %d", tt.title, result, tt.expected)
			}
		})
	}
}

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
		{"No match", "Sport und Spiel", ""},
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
