package pdftest

import (
	"strings"
	"testing"
	"unicode/utf8"

	"bibliothek/pkg/pdfzeichen"

	"github.com/jung-kurt/gofpdf"
)

// InSpalte hält einen gedruckten Text gegen seine Quelle: Er steht ganz da oder so weit
// gekürzt wie nötig, mit Auslassungszeichen, und gedruckt ist er nicht breiter als der Platz in
// seiner Zelle (Millimeter, in Arial der genannten Größe).
func InSpalte(t *testing.T, gedruckt, quelle string, groesse, platz float64) {
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
