package inventur

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIstErlaubteCoverHerkunft(t *testing.T) {
	tests := []struct {
		name     string
		rohURL   string
		expected bool
	}{
		{
			name:     "allowed host OpenLibrary",
			rohURL:   "https://covers.openlibrary.org/b/id/12345-L.jpg",
			expected: true,
		},
		{
			name:     "allowed host Google Books",
			rohURL:   "http://books.google.com/books/content?id=123",
			expected: true,
		},
		{
			name:     "disallowed host",
			rohURL:   "https://evil.com/cover.jpg",
			expected: false,
		},
		{
			name:     "unparseable URL",
			rohURL:   "://invalid",
			expected: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := IstErlaubteCoverHerkunft(tc.rohURL)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestSichereCoverURL(t *testing.T) {
	tests := []struct {
		name        string
		rohURL      string
		expectedURL string
		expectedOk  bool
	}{
		{
			name:        "allowed host OpenLibrary HTTPS",
			rohURL:      "https://covers.openlibrary.org/b/id/12345-L.jpg",
			expectedURL: "https://covers.openlibrary.org/b/id/12345-L.jpg",
			expectedOk:  true,
		},
		{
			name:        "allowed host Google Books HTTP to HTTPS",
			rohURL:      "http://books.google.com/books/content?id=123",
			expectedURL: "https://books.google.com/books/content?id=123",
			expectedOk:  true,
		},
		{
			name:        "disallowed host",
			rohURL:      "https://evil.com/cover.jpg",
			expectedURL: "",
			expectedOk:  false,
		},
		{
			name:        "unparseable URL",
			rohURL:      "://invalid",
			expectedURL: "",
			expectedOk:  false,
		},
		{
			name:        "allowed host with port stripped",
			rohURL:      "https://covers.openlibrary.org:2222/x.jpg",
			expectedURL: "https://covers.openlibrary.org/x.jpg",
			expectedOk:  true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resultURL, ok := SichereCoverURL(tc.rohURL)
			assert.Equal(t, tc.expectedOk, ok)
			assert.Equal(t, tc.expectedURL, resultURL)
		})
	}
}
