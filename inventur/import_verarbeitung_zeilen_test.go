package inventur

import (
	"testing"
)

func TestParseKlassenStufe(t *testing.T) {
	tests := []struct {
		name     string
		gradeStr string
		title    string
		expected int
	}{
		{"Valid grade string inside bounds", "7", "Any Title", 7},
		{"Valid grade string lower bound", "5", "Any Title", 5},
		{"Valid grade string upper bound", "10", "Any Title", 10},
		{"Valid grade string below bounds", "4", "Any Title", 5},
		{"Valid grade string above bounds", "11", "Any Title", 5},
		{"Invalid grade string, title has valid grade", "abc", "Math 8", 8},
		{"Invalid grade string, title has no grade", "abc", "Random Title", 5},
		{"Invalid grade string, title grade below bounds", "abc", "Level 4", 5},
		{"Invalid grade string, title grade above bounds", "abc", "Level 11", 5},
		{"Grade string is zero, title has valid grade", "0", "English 9", 9},
		{"Grade string is zero, title has no grade", "0", "Another Title", 5},
		{"Grade string is empty, title has valid grade", "", "Science 6", 6},
		{"Grade string is empty, title has no grade", "", "Science", 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseKlassenStufe(tt.gradeStr, tt.title)
			if result != tt.expected {
				t.Errorf("parseKlassenStufe(%q, %q) = %d; want %d", tt.gradeStr, tt.title, result, tt.expected)
			}
		})
	}
}
