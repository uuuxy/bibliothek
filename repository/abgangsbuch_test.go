package repository

import (
	"testing"
)

func TestAbgangsgrundText(t *testing.T) {
	tests := []struct {
		name  string
		grund string
		want  string
	}{
		{
			name:  "VERLUST",
			grund: "VERLUST",
			want:  "Verlust",
		},
		{
			name:  "BESCHAEDIGUNG",
			grund: "BESCHAEDIGUNG",
			want:  "Beschädigung",
		},
		{
			name:  "AUSSORTIERT",
			grund: "AUSSORTIERT",
			want:  "Aussortiert",
		},
		{
			name:  "BESTANDSKORREKTUR",
			grund: "BESTANDSKORREKTUR",
			want:  "Bestandskorrektur",
		},
		{
			name:  "empty string (Altdaten)",
			grund: "",
			want:  "ohne Angabe",
		},
		{
			name:  "unknown ground defaults to itself",
			grund: "UNBEKANNT",
			want:  "UNBEKANNT",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := AbgangsgrundText(tt.grund); got != tt.want {
				t.Errorf("AbgangsgrundText() = %q, want %q", got, tt.want)
			}
		})
	}
}
