package auth

import (
	"testing"
)

func TestHashToken(t *testing.T) {
	tests := []struct {
		name     string
		token    string
		expected string
	}{
		{
			name:     "Empty string",
			token:    "",
			expected: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		},
		{
			name:     "Simple token",
			token:    "test_token",
			expected: "cc0af97287543b65da2c7e1476426021826cab166f1e063ed012b855ff819656",
		},
		{
			name:     "Realistic JWT",
			token:    "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c",
			expected: "7f75367e7881255134e1375e723d1dea8ad5f6a4fdb79d938df1f1754a830606",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hashToken(tt.token)
			if result != tt.expected {
				t.Errorf("hashToken() = %v, want %v", result, tt.expected)
			}
		})
	}
}
