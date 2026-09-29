package main

import "testing"

func TestValidateBarcode(t *testing.T) {
	tests := []struct {
		name string
		bc   string
		want bool
	}{
		{"valid standard", "B-12345", true},
		{"valid small", "B-1", true},
		{"valid large", "B-999999999", true},
		{"valid zero padded", "B-00123", true},
		{"invalid empty", "", false},
		{"invalid no number", "B-", false},
		{"invalid letters", "B-123A", false},
		{"invalid lowercase", "b-123", false},
		{"invalid missing B", "12345", false},
		{"invalid dash only", "-", false},
		{"invalid spaces", "B- 123", false},
		{"invalid extra chars start", "AB-123", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := validateBarcode(tt.bc); got != tt.want {
				t.Errorf("validateBarcode(%q) = %v, want %v", tt.bc, got, tt.want)
			}
		})
	}
}
