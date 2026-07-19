package isbnutil

import "testing"

func TestClean(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"with hyphens", "978-3-16-148410-0", "9783161484100"},
		{"with spaces", "978 3 16 148410 0", "9783161484100"},
		{"with both", "978-3 16-148410 0", "9783161484100"},
		{"already clean", "9783161484100", "9783161484100"},
		{"empty string", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Clean(tt.input); got != tt.expected {
				t.Errorf("Clean() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func BenchmarkClean(b *testing.B) {
	s := "978-3-16-148410-0"
	for i := 0; i < b.N; i++ {
		Clean(s)
	}
}

func BenchmarkClean_NoAlloc(b *testing.B) {
	s := "9783161484100"
	for i := 0; i < b.N; i++ {
		Clean(s)
	}
}
