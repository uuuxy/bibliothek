package inventur

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestLeseCSVZeilen(t *testing.T) {
	faelle := []struct {
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

	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			r := strings.NewReader(f.csvData)
			ergebnis, err := leseCSVZeilen(r)
			if f.expectErr {
				if err == nil {
					t.Errorf("leseCSVZeilen(…) = nil, erwartet einen Fehler")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if !cmp.Equal(ergebnis, f.expect) {
					t.Errorf("leseCSVZeilen(r) = %v, erwartet %v", ergebnis, f.expect)
				}
			}
		})
	}
}

func TestDetermineColumnIndices(t *testing.T) {
	faelle := []struct {
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

	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			idx, mitKopf := bestimmeSpalten(f.header)
			if mitKopf != f.expectedHasHead {
				t.Errorf("bestimmeSpalten(f.header) = %v, erwartet hasHeader=%v", mitKopf, f.expectedHasHead)
			}
			if !cmp.Equal(idx, f.expectedIdx) {
				t.Errorf("bestimmeSpalten(f.header) = %v, erwartet %v", idx, f.expectedIdx)
			}
		})
	}
}

func TestErrateSpaltenAusInhalt(t *testing.T) {
	faelle := []struct {
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

	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			errateSpaltenAusInhalt(f.header, f.initIdx)
			if !cmp.Equal(f.initIdx, f.expected) {
				t.Errorf("f.initIdx = %v, erwartet %v", f.initIdx, f.expected)
			}
		})
	}
}
