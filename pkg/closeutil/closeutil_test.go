package closeutil

import (
	"bytes"
	"errors"
	"log"
	"strings"
	"testing"
)

type attrappe struct {
	err error
}

func (a *attrappe) Close() error {
	return a.err
}

func TestSchliesse(t *testing.T) {
	vorher := log.Writer()
	defer log.SetOutput(vorher)

	var ausgabe bytes.Buffer
	log.SetOutput(&ausgabe)

	t.Run("ohne Fehler bleibt das Log leer", func(t *testing.T) {
		ausgabe.Reset()

		Schliesse(&attrappe{err: nil}, "test-success")

		if ausgabe.Len() > 0 {
			t.Errorf("Log = %q, erwartet keine Zeile", ausgabe.String())
		}
	})

	t.Run("ein Fehler steht mit seiner Stelle im Log", func(t *testing.T) {
		ausgabe.Reset()

		Schliesse(&attrappe{err: errors.New("simulierter Fehler")}, "test-failure")

		want := "test-failure: close failed: simulierter Fehler"
		if got := ausgabe.String(); !strings.Contains(got, want) {
			t.Errorf("Log = %q, erwartet darin %q", got, want)
		}
	})
}
