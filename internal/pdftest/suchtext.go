package pdftest

import (
	"strings"
	"testing"
)

// Suchtext liefert das PDF als durchsuchbaren Text: die rohen Bytes und dahinter die Texte der
// entpackten Ströme. Kleine Ströme lässt gofpdf offen; sie stehen nur in den rohen Bytes.
func Suchtext(t *testing.T, roh []byte) string {
	t.Helper()
	return string(roh) + "\n" + strings.Join(Texte(t, roh), "\n")
}

// ErsteBytes liefert die ersten n Bytes, von kürzerem Inhalt alle: für eine Meldung, die den
// Anfang einer Antwort zeigt.
func ErsteBytes(b []byte, n int) []byte {
	if len(b) < n {
		return b
	}
	return b[:n]
}
