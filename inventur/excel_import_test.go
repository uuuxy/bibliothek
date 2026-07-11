package inventur

import (
	"reflect"
	"testing"
)

func TestDetermineColumnIndices(t *testing.T) {
	tests := []struct {
		name          string
		header        []string
		wantColIdx    map[string]int
		wantHasHeader bool
	}{
		{
			name:   "Valid Header Explicit",
			header: []string{"ISBN", "Titel", "Autor", "Fach", "Klasse", "Bestand"},
			wantColIdx: map[string]int{
				"isbn":    0,
				"titel":   1,
				"autor":   2,
				"fach":    3,
				"klasse":  4,
				"bestand": 5,
			},
			wantHasHeader: true,
		},
		{
			name:   "Valid Header Aliases",
			header: []string{"ISBN", "Ausgabe", "Author", "Subject", "Stufe", "Anzahl"},
			wantColIdx: map[string]int{
				"isbn":    0,
				"titel":   1,
				"autor":   2,
				"fach":    3,
				"klasse":  4,
				"bestand": 5,
			},
			wantHasHeader: true,
		},
		{
			name:   "No Header - Fallback (first row is data with 978 ISBN)",
			header: []string{"978-3-16-148410-0", "Ein Buchtitel", "5", "X", "Y"}, // X, Y are extra cols
			wantColIdx: map[string]int{
				"isbn":    0,
				"titel":   1,
				"autor":   -1,
				"fach":    -1,
				"klasse":  -1,
				"bestand": 2,
			},
			wantHasHeader: false,
		},
		{
			name:   "No Header - Fallback (first row is data with 979 ISBN, different order)",
			header: []string{"10", "Ein Buchtitel", "979-1-23-456789-0"},
			wantColIdx: map[string]int{
				"isbn":    2,
				"titel":   1,
				"autor":   -1,
				"fach":    -1,
				"klasse":  -1,
				"bestand": 0,
			},
			wantHasHeader: false,
		},
		{
			name:   "Partial Header Explicit",
			header: []string{"ISBN", "Ignored", "Titel"},
			wantColIdx: map[string]int{
				"isbn":    0,
				"titel":   2,
				"autor":   -1,
				"fach":    -1,
				"klasse":  -1,
				"bestand": -1,
			},
			wantHasHeader: true,
		},
		{
			name:   "Empty Header Array",
			header: []string{},
			wantColIdx: map[string]int{
				"isbn":    -1,
				"titel":   -1,
				"autor":   -1,
				"fach":    -1,
				"klasse":  -1,
				"bestand": -1,
			},
			wantHasHeader: false,
		},
		{
			name:   "No Valid Header Or ISBN",
			header: []string{"Some", "Random", "Values", "Here"},
			wantColIdx: map[string]int{
				"isbn":    -1,
				"titel":   0, // "Some" length > 2
				"autor":   -1,
				"fach":    -1,
				"klasse":  -1,
				"bestand": -1,
			},
			wantHasHeader: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotColIdx, gotHasHeader := determineColumnIndices(tt.header)
			if gotHasHeader != tt.wantHasHeader {
				t.Errorf("determineColumnIndices() gotHasHeader = %v, want %v", gotHasHeader, tt.wantHasHeader)
			}
			if !reflect.DeepEqual(gotColIdx, tt.wantColIdx) {
				t.Errorf("determineColumnIndices() gotColIdx = %v, want %v", gotColIdx, tt.wantColIdx)
			}
		})
	}
}
