package inventur

import (
	"testing"
)

func TestExtrahiereZahlenUndBasis(t *testing.T) {
	tests := []struct {
		name         string
		titel        string
		expectedZahl int
		expectedBase string
	}{
		{
			name:         "Normal string with number at the end",
			titel:        "Band 1",
			expectedZahl: 1,
			expectedBase: "Band",
		},
		{
			name:         "String without numbers",
			titel:        "Buch",
			expectedZahl: 0,
			expectedBase: "Buch",
		},
		{
			name:         "String with numbers at the start",
			titel:        "123 Test",
			expectedZahl: 123,
			expectedBase: "Test",
		},
		{
			name:         "String with multiple numbers",
			titel:        "Buch 42 mit 99",
			expectedZahl: 42,
			expectedBase: "Buch  mit",
		},
		{
			name:         "Empty string",
			titel:        "",
			expectedZahl: 0,
			expectedBase: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			zahl, basis := extrahiereZahlenUndBasis(tt.titel)
			if zahl != tt.expectedZahl {
				t.Errorf("extrahiereZahlenUndBasis(%q) got zahl = %d, want %d", tt.titel, zahl, tt.expectedZahl)
			}
			if basis != tt.expectedBase {
				t.Errorf("extrahiereZahlenUndBasis(%q) got basis = %q, want %q", tt.titel, basis, tt.expectedBase)
			}
		})
	}
}
