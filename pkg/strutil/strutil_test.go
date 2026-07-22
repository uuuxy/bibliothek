package strutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCleanISBN(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"with hyphens", "978-3-16-148410-0", "9783161484100"},
		{"with spaces", "978 3 16 148410 0", "9783161484100"},
		{"mixed hyphens and spaces", " 978-3 16-148410-0 ", "9783161484100"},
		{"already clean", "9783161484100", "9783161484100"},
		{"empty string", "", ""},
		{"only hyphens and spaces", " - -   - ", ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := CleanISBN(tc.input)
			assert.Equal(t, tc.expected, got)
		})
	}
}
