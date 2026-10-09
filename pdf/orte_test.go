package pdf

import (
	"regexp"
	"strconv"
	"strings"
	"testing"
	"unicode/utf8"

	"bibliothek/internal/pdftest"
	"bibliothek/pkg/pdfzeichen"

	"github.com/jung-kurt/gofpdf"
)

// Text und Bild im Inhaltsstrom einer Seite, in Punkt und von unten gemessen:
// `BT 76.54 584.37 Td (Titel 1)Tj ET` und `q 19.84 0 0 48.19 51.02 562.68 cm /I… Do Q`.
var (
	textOrt = regexp.MustCompile(`BT ([0-9.]+) ([0-9.]+) Td \(((?:\\.|[^()\\])*)\)Tj ET`)
	bildOrt = regexp.MustCompile(`q ([0-9.]+) 0 0 ([0-9.]+) ([0-9.]+) ([0-9.]+) cm /I[0-9a-f]+ Do Q`)
)

// seiteMitOrten ist, was auf einer Seite steht: jeder Text mit seiner Höhe und seinem linken
// Rand, dazu die Mitten der Bilder. Für Prüfungen, ob die Teile einer Tabellenzeile beieinander
// stehen.
type seiteMitOrten struct {
	texte      map[string]float64
	links      map[string]float64
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
		seite := seiteMitOrten{texte: map[string]float64{}, links: map[string]float64{}}
		for _, m := range textOrt.FindAllSubmatch(strom, -1) {
			seite.texte[string(m[3])] = zahl(t, string(m[2]))
			seite.links[string(m[3])] = zahl(t, string(m[1]))
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

// pruefeInSpalte hält einen gedruckten Text gegen seine Quelle: Er steht ganz da oder so weit
// gekürzt wie nötig, mit Auslassungszeichen, und gedruckt ist er nicht breiter als der Platz in
// seiner Zelle (Millimeter, in Arial der genannten Größe).
func pruefeInSpalte(t *testing.T, gedruckt, quelle string, groesse, platz float64) {
	t.Helper()
	messen := gofpdf.New("P", "mm", "A4", "")
	messen.SetFont("Arial", "", groesse)
	tr := pdfzeichen.Uebersetzer(messen.UnicodeTranslatorFromDescriptor(""))
	if breite := messen.GetStringWidth(tr(gedruckt)); breite > platz {
		t.Errorf("%q ist gedruckt %.1f mm breit, die Zelle lässt %.1f mm", gedruckt, breite, platz)
	}
	if gedruckt == quelle {
		return
	}
	anfang, gekuerzt := strings.CutSuffix(gedruckt, "…")
	if !gekuerzt || !strings.HasPrefix(quelle, anfang) {
		t.Errorf("gedruckt %q ist weder %q noch ein Anfang davon mit Auslassungszeichen", gedruckt, quelle)
		return
	}
	// Gekürzt wird nicht mehr als nötig: Mit dem nächsten Zeichen der Quelle passte es nicht.
	rest := quelle[len(anfang):]
	naechstes := strings.IndexFunc(rest, func(r rune) bool { return r != ' ' })
	if naechstes < 0 {
		t.Errorf("gedruckt %q ist gekürzt, obwohl von %q nur Leerzeichen fehlen", gedruckt, quelle)
		return
	}
	_, laenge := utf8.DecodeRuneInString(rest[naechstes:])
	laenger := anfang + rest[:naechstes+laenge] + "…"
	if breite := messen.GetStringWidth(tr(laenger)); breite <= platz {
		t.Errorf("gedruckt %q, dabei passte auch %q (%.1f mm von %.1f mm)", gedruckt, laenger, breite, platz)
	}
}
