package repository

import (
	"testing"
)

// Littera kodiert Anführungszeichen je nach Exportweg unterschiedlich
// (PDF-CSV: `""South Africa"""`, XML: `"South Africa"`). Der Matching-Schlüssel
// muss beide Varianten auf denselben Wert abbilden — sonst legt jeder Import
// eine Dublette an. Der gespeicherte Titel bleibt davon unberührt.
func TestNormalisiereTitelKey(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{`"Kein Bock auf Lernen?"`, "Kein Bock auf Lernen?"},
		{`""Kein Bock auf Lernen?"""`, "Kein Bock auf Lernen?"},
		{`LMF-Green Line - Topic zum country of reference ""South Africa`, "LMF-Green Line - Topic zum country of reference South Africa"},
		{`LMF-"Green Line - Topic zum country of reference "South Africa"`, "LMF-Green Line - Topic zum country of reference South Africa"},
		{"  Faust   Teil 1 ", "Faust Teil 1"},
		{"Faust", "Faust"},
		// Additional edge cases
		{"", ""},
		{`""`, ""},
		{`" "`, ""},
		{"   ", ""},
		{`"""`, ""},
		{"\t\n", ""},
		{`"Hello"` + "\t" + `"World"`, "Hello World"},
	}
	for _, tt := range tests {
		if got := NormalisiereTitelKey(tt.in); got != tt.want {
			t.Errorf("NormalisiereTitelKey(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
