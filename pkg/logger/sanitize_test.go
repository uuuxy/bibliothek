package logger

import (
	"testing"
)

func TestSanitizeLog(t *testing.T) {
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
			name:     "no newlines",
			input:    "hello world",
			expected: "hello world",
		},
		{
			name:     "with newline",
			input:    "hello\nworld",
			expected: "helloworld",
		},
		{
			name:     "with carriage return",
			input:    "hello\rworld",
			expected: "helloworld",
		},
		{
			name:     "with both",
			input:    "hello\r\nworld",
			expected: "helloworld",
		},
		{
			name:     "multiple newlines and carriage returns",
			input:    "\r\nhello\n\rworld\n\r",
			expected: "helloworld",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeLog(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeLog() = %q, expected %q", result, tt.expected)
			}
		})
	}
}
