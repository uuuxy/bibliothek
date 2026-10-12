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

// Aufdruck hält den Suchtext eines Blatts gegen einen Wortlaut, der dastehen muss, und einen,
// der dort nicht stehen darf. weg nennt in der Meldung, auf welchem Weg das Blatt entstand.
func Aufdruck(t *testing.T, weg, text, want, nicht string) {
	t.Helper()
	if !strings.Contains(text, want) {
		t.Errorf("%s: %q fehlt auf dem Blatt", weg, want)
	}
	if strings.Contains(text, nicht) {
		t.Errorf("%s: %q steht auf dem Blatt und gehört dort nicht hin", weg, nicht)
	}
}

// ErsteBytes liefert die ersten n Bytes, von kürzerem Inhalt alle: für eine Meldung, die den
// Anfang einer Antwort zeigt.
func ErsteBytes(b []byte, n int) []byte {
	if len(b) < n {
		return b
	}
	return b[:n]
}
