package pdftest

import (
	"strings"
	"testing"
	"unicode/utf8"

	"bibliothek/pkg/pdfzeichen"

	"github.com/phpdave11/gofpdf"
)

// InSpalte hält einen gedruckten Text gegen seine Quelle: Er steht ganz da oder so weit
// gekürzt wie nötig, mit Auslassungszeichen, und gedruckt ist er nicht breiter als der Platz in
// seiner Zelle (Millimeter, in Arial der genannten Größe).
func InSpalte(t *testing.T, gedruckt, quelle string, groesse, platz float64) {
	t.Helper()
	inSpalte(t, gedruckt, quelle, "", groesse, platz)
}

// InSpalteFett prüft dasselbe für einen Text in fetter Schrift; fett ist jeder Buchstabe
// breiter, und eine Messung in magerer Schrift ließe einen zu breiten Text durch.
func InSpalteFett(t *testing.T, gedruckt, quelle string, groesse, platz float64) {
	t.Helper()
	inSpalte(t, gedruckt, quelle, "B", groesse, platz)
}

// Breite misst einen Text in Millimetern, so wie gofpdf ihn in der Schrift (etwa „Courier"),
// dem Schnitt („B" für fett, sonst leer) und der Größe druckt.
func Breite(text, schrift, schnitt string, groesse float64) float64 {
	messen := gofpdf.New("P", "mm", "A4", "")
	messen.SetFont(schrift, schnitt, groesse)
	uebersetze := pdfzeichen.Uebersetzer(messen.UnicodeTranslatorFromDescriptor(""))
	return messen.GetStringWidth(uebersetze(text))
}

func inSpalte(t *testing.T, gedruckt, quelle, schnitt string, groesse, platz float64) {
	t.Helper()
	if breite := Breite(gedruckt, "Arial", schnitt, groesse); breite > platz {
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
	if breite := Breite(laenger, "Arial", schnitt, groesse); breite <= platz {
		t.Errorf("gedruckt %q, dabei passte auch %q (%.1f mm von %.1f mm)", gedruckt, laenger, breite, platz)
	}
}
