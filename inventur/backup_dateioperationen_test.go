package inventur

import (
	"testing"
)

func TestEscapePgPass(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "no escaping needed",
			input:    "localhost",
			expected: "localhost",
		},
		{
			name:     "only backslashes",
			input:    "\\\\",
			expected: "\\\\\\\\",
		},
		{
			name:     "only colons",
			input:    "::",
			expected: "\\:\\:",
		},
		{
			name:     "string with backslash",
			input:    "domain\\user",
			expected: "domain\\\\user",
		},
		{
			name:     "string with colon",
			input:    "password:with:colons",
			expected: "password\\:with\\:colons",
		},
		{
			name:     "string with both",
			input:    "my\\password:123",
			expected: "my\\\\password\\:123",
		},
		{
			name:     "trailing slash",
			input:    "trailing\\",
			expected: "trailing\\\\",
		},
		{
			name:     "trailing colon",
			input:    "trailing:",
			expected: "trailing\\:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := escapePgPass(tt.input)
			if result != tt.expected {
				t.Errorf("escapePgPass(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}
