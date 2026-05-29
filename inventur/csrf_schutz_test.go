package inventur

import (
	"net/http"
	"testing"
)

func TestIsMutationMethod(t *testing.T) {
	tests := []struct {
		name     string
		method   string
		expected bool
	}{
		// Mutation methods
		{"POST method", http.MethodPost, true},
		{"PUT method", http.MethodPut, true},
		{"PATCH method", http.MethodPatch, true},
		{"DELETE method", http.MethodDelete, true},

		// Non-mutation methods
		{"GET method", http.MethodGet, false},
		{"HEAD method", http.MethodHead, false},
		{"OPTIONS method", http.MethodOptions, false},
		{"TRACE method", http.MethodTrace, false},
		{"CONNECT method", http.MethodConnect, false},

		// Edge cases
		{"Empty method", "", false},
		{"Invalid method", "INVALID", false},
		{"Lowercase post", "post", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isMutationMethod(tt.method)
			if result != tt.expected {
				t.Errorf("isMutationMethod(%q) = %v; want %v", tt.method, result, tt.expected)
			}
		})
	}
}
