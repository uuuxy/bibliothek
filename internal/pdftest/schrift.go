package pdftest

import (
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// schriftOderText findet im Inhaltsstrom die Wahl der Schrift (`/F3 9.00 Tf`) und die gedruckten
// Textstücke wie tjText, in der Reihenfolge des Stroms.
var schriftOderText = regexp.MustCompile(`/\w+\s+([0-9.]+)\s+Tf|\(((?:\\.|[^()\\])*)\)\s*Tj`)

// Schriftgroesse liefert die Größe in Punkt, in der der Text gedruckt ist: die der letzten
// Schriftwahl davor im Strom. Für einen Erzeuger, der die Schrift verkleinert, wenn ein Text
// nicht in seine Zelle passt; seine Breite misst dann Breite in dieser Größe.
func Schriftgroesse(t *testing.T, roh []byte, text string) float64 {
	t.Helper()
	inhalt, _ := lies(t, roh)
	var aktuell, groesse float64
	treffer := 0
	for _, m := range schriftOderText.FindAllSubmatch(inhalt, -1) {
		if m[1] != nil {
			wert, err := strconv.ParseFloat(string(m[1]), 64)
			if err != nil {
				t.Fatalf("Schriftgröße %q nicht lesbar: %v", m[1], err)
			}
			aktuell = wert
			continue
		}
		if strings.TrimSpace(nachUTF8(m[2])) == text {
			groesse = aktuell
			treffer++
		}
	}
	if treffer != 1 {
		t.Fatalf("%q ist %d-mal gedruckt, erwartet einmal", text, treffer)
	}
	if groesse == 0 {
		t.Fatalf("vor %q wählt der Strom keine Schrift", text)
	}
	return groesse
}
