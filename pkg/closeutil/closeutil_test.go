package closeutil

import (
	"bytes"
	"errors"
	"log"
	"strings"
	"testing"
)

type mockCloser struct {
	err error
}

func (m *mockCloser) Close() error {
	return m.err
}

func TestLogClose(t *testing.T) {
	originalOutput := log.Writer()
	defer log.SetOutput(originalOutput)

	var buf bytes.Buffer
	log.SetOutput(&buf)

	t.Run("Kein Fehler beim Schliessen", func(t *testing.T) {
		buf.Reset()
		m := &mockCloser{err: nil}

		LogClose(m, "test-success")

		if buf.Len() > 0 {
			t.Errorf("Es wurde keine Log-Ausgabe erwartet, aber erhalten: %q", buf.String())
		}
	})

	t.Run("Fehler beim Schliessen", func(t *testing.T) {
		buf.Reset()
		testErr := errors.New("simulierter Fehler")
		m := &mockCloser{err: testErr}

		LogClose(m, "test-failure")

		output := buf.String()
		expectedSnippet := "test-failure: close failed: simulierter Fehler"
		if !strings.Contains(output, expectedSnippet) {
			t.Errorf("Log-Ausgabe sollte %q enthalten, stattdessen erhalten: %q", expectedSnippet, output)
		}
	})
}
