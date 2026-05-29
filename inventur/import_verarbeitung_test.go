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
		{"Year before grade", "2023 Edition Grade 8", 8},
		{"Year after grade", "Grade 8 (2023)", 8},
		{"Grade with slash", "Grade 5/6", 5},
		{"Grade with dash", "Klasse 7-8", 7},
		{"Grade list with comma", "Grade 1,2,3", 1},
		{"Ordinal format", "10. Klasse", 10},
		{"Leading zero ignored", "Grade 08", 0},
		{"Volume or Band", "Band 5", 5},
		{"Language level A1", "English A1", 1},
		{"Language level B2", "French B2", 2},
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
