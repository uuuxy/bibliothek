package service

import (
	"bytes"
	"errors"
	"log"
	"testing"
)

func TestMeldeProtokollFehler(t *testing.T) {
	tests := []struct {
		name     string
		action   string
		err      error
		expected string
	}{
		{
			name:     "No error, no log",
			action:   "Rückgabe",
			err:      nil,
			expected: "",
		},
		{
			name:     "Error logged correctly",
			action:   "Ausleihe",
			err:      errors.New("db connection failed"),
			expected: "ERROR audit: Eintrag konnte nicht protokolliert werden aktion=Ausleihe fehler=\"db connection failed\"\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			originalWriter := log.Writer()
			originalFlags := log.Flags()

			log.SetOutput(&buf)
			log.SetFlags(0)
			defer func() {
				log.SetOutput(originalWriter)
				log.SetFlags(originalFlags)
			}()

			meldeProtokollFehler(tt.action, tt.err)

			if got := buf.String(); got != tt.expected {
				t.Errorf("buf.String() = %q, erwartet %q", got, tt.expected)
			}
		})
	}
}
