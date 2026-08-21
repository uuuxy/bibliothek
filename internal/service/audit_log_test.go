package service

import (
	"bytes"
	"errors"
	"log"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLogAuditErr(t *testing.T) {
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
			expected: "audit: Ausleihe konnte nicht protokolliert werden: db connection failed\n",
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

			logAuditErr(tt.action, tt.err)

			assert.Equal(t, tt.expected, buf.String())
		})
	}
}
