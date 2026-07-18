package inventur

import (
	"reflect"
	"strings"
	"testing"
)

func TestParseCSVRows(t *testing.T) {
	tests := []struct {
		name      string
		csvData   string
		expect    [][]string
		expectErr bool
	}{
		{
			name:    "Comma separated",
			csvData: "isbn,titel,autor\n9781234567890,Buch1,Autor1\n9780987654321,Buch2,Autor2",
			expect: [][]string{
				{"isbn", "titel", "autor"},
				{"9781234567890", "Buch1", "Autor1"},
				{"9780987654321", "Buch2", "Autor2"},
			},
			expectErr: false,
		},
		{
			name:    "Semicolon separated",
			csvData: "isbn;titel;autor\n9781234567890;Buch1;Autor1\n9780987654321;Buch2;Autor2",
			expect: [][]string{
				{"isbn", "titel", "autor"},
				{"9781234567890", "Buch1", "Autor1"},
				{"9780987654321", "Buch2", "Autor2"},
			},
			expectErr: false,
		},
		{
			name:      "Empty data",
			csvData:   "",
			expect:    nil,
			expectErr: true,
		},
		{
			name:      "Malformed CSV",
			csvData:   "isbn,titel\n\"abc\n", // wrong number of fields
			expect:    nil,
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := strings.NewReader(tt.csvData)
			result, err := parseCSVRows(r)
			if tt.expectErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if !reflect.DeepEqual(result, tt.expect) {
					t.Errorf("expected %v, got %v", tt.expect, result)
				}
			}
		})
	}
}

func TestDetermineColumnIndices(t *testing.T) {
	tests := []struct {
		name            string
		header          []string
		expectedIdx     map[string]int
		expectedHasHead bool
	}{
		{
			name:   "Valid Header",
			header: []string{"ISBN", "Titel", "Autor", "Fach", "Klasse", "Bestand"},
			expectedIdx: map[string]int{
				"isbn":    0,
				"titel":   1,
				"autor":   2,
				"fach":    3,
				"klasse":  4,
				"bestand": 5,
			},
			expectedHasHead: true,
		},
		{
			name:   "Missing some optional headers",
			header: []string{"ISBN", "Title", "Author"},
			expectedIdx: map[string]int{
				"isbn":    0,
				"titel":   1,
				"autor":   2,
				"fach":    -1,
				"klasse":  -1,
				"bestand": -1,
			},
			expectedHasHead: true,
		},
		{
			name:   "No header, guess from data",
			header: []string{"9781234567890", "Der Hobbit", "3"},
			expectedIdx: map[string]int{
				"isbn":    0,
				"titel":   1,
				"autor":   -1,
				"fach":    -1,
				"klasse":  -1,
				"bestand": 2,
			},
			expectedHasHead: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			idx, hasHead := determineColumnIndices(tt.header)
			if hasHead != tt.expectedHasHead {
				t.Errorf("expected hasHeader=%v, got %v", tt.expectedHasHead, hasHead)
			}
			if !reflect.DeepEqual(idx, tt.expectedIdx) {
				t.Errorf("expected %v, got %v", tt.expectedIdx, idx)
			}
		})
	}
}

func TestErrateSpaltenAusInhalt(t *testing.T) {
	tests := []struct {
		name     string
		header   []string
		initIdx  map[string]int
		expected map[string]int
	}{
		{
			name:   "Guess ISBN, Titel, and Bestand",
			header: []string{"978-3-16-148410-0", "Ein sehr langes Buch", "15"},
			initIdx: map[string]int{
				"isbn":    -1,
				"titel":   -1,
				"autor":   -1,
				"fach":    -1,
				"klasse":  -1,
				"bestand": -1,
			},
			expected: map[string]int{
				"isbn":    0,
				"titel":   1,
				"autor":   -1,
				"fach":    -1,
				"klasse":  -1,
				"bestand": 2,
			},
		},
		{
			name:   "Guess ISBN 979 and partial missing",
			header: []string{"9791234567890", "Test"},
			initIdx: map[string]int{
				"isbn":    -1,
				"titel":   -1,
				"autor":   -1,
				"fach":    -1,
				"klasse":  -1,
				"bestand": -1,
			},
			expected: map[string]int{
				"isbn":    0,
				"titel":   1,
				"autor":   -1,
				"fach":    -1,
				"klasse":  -1,
				"bestand": -1,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errateSpaltenAusInhalt(tt.header, tt.initIdx)
			if !reflect.DeepEqual(tt.initIdx, tt.expected) {
				t.Errorf("expected %v, got %v", tt.expected, tt.initIdx)
			}
		})
	}
}
