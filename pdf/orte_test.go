package pdf

import (
	"regexp"
	"strconv"
	"testing"

	"bibliothek/internal/pdftest"
)

// Text und Bild im Inhaltsstrom einer Seite, in Punkt und von unten gemessen:
// `BT 76.54 584.37 Td (Titel 1)Tj ET` und `q 19.84 0 0 48.19 51.02 562.68 cm /I… Do Q`.
var (
	textOrt = regexp.MustCompile(`BT ([0-9.]+) ([0-9.]+) Td \(((?:\\.|[^()\\])*)\)Tj ET`)
	bildOrt = regexp.MustCompile(`q ([0-9.]+) 0 0 ([0-9.]+) ([0-9.]+) ([0-9.]+) cm /I[0-9a-f]+ Do Q`)
)

// seiteMitOrten ist, was auf einer Seite steht: jeder Text mit seiner Höhe und die Mitten der
// Bilder. Für Prüfungen, ob die Teile einer Tabellenzeile beieinander stehen.
type seiteMitOrten struct {
	texte      map[string]float64
	bildMitten []float64
}

func zahl(t *testing.T, s string) float64 {
	t.Helper()
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		t.Fatalf("Zahl im Inhaltsstrom unlesbar: %q", s)
	}
	return f
}

func seitenMitOrten(t *testing.T, roh []byte) []seiteMitOrten {
	t.Helper()
	var seiten []seiteMitOrten
	for _, strom := range pdftest.InhaltJeSeite(t, roh) {
		seite := seiteMitOrten{texte: map[string]float64{}}
		for _, m := range textOrt.FindAllSubmatch(strom, -1) {
			seite.texte[string(m[3])] = zahl(t, string(m[2]))
		}
		for _, m := range bildOrt.FindAllSubmatch(strom, -1) {
			seite.bildMitten = append(seite.bildMitten, zahl(t, string(m[4]))+zahl(t, string(m[2]))/2)
		}
		seiten = append(seiten, seite)
	}
	return seiten
}

// bilderInDerZeile zählt die Bilder, deren Mitte höchstens eine halbe Zeile neben der Höhe liegt.
func (s seiteMitOrten) bilderInDerZeile(hoehe, halbeZeile float64) int {
	bilder := 0
	for _, mitte := range s.bildMitten {
		if mitte-hoehe <= halbeZeile && hoehe-mitte <= halbeZeile {
			bilder++
		}
	}
	return bilder
}
