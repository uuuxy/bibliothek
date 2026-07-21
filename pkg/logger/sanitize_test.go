package logger

import "testing"

func TestSanitizeLog(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Hello World", "Hello World"},
		{"Hello\nWorld", "HelloWorld"},
		{"Hello\rWorld", "HelloWorld"},
		{"Hello\r\nWorld", "HelloWorld"},
		{"\r\n\r\n", ""},
		{"", ""},
	}

	for _, test := range tests {
		actual := SanitizeLog(test.input)
		if actual != test.expected {
			t.Errorf("SanitizeLog(%q) = %q, expected %q", test.input, actual, test.expected)
		}
	}
}
