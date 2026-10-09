package pdfzeichen

import (
	"testing"
	"unicode/utf8"
)

// jeZeichen misst jedes Zeichen mit einem Millimeter, ein großes M mit dreien.
type jeZeichen struct{}

func (jeZeichen) GetStringWidth(text string) float64 {
	breite := 0.0
	for _, r := range text {
		if r == 'M' {
			breite += 3
			continue
		}
		breite++
	}
	return breite
}

// GetCellMargin: ein Millimeter Rand links und rechts vom Text einer Zelle.
func (jeZeichen) GetCellMargin() float64 { return 1 }

func gleich(s string) string { return s }

func TestKuerzeAufBreite(t *testing.T) {
	faelle := []struct {
		name, text string
		breite     float64
		soll       string
	}{
		{"passt", "Demir, Ayla", 11, "Demir, Ayla"},
		{"ein Zeichen zu breit", "Demir, Ayla", 10, "Demir, Ay…"},
		{"gleiche Zeichenzahl, breitere Schrift", "MMMMMMMMMMM", 11, "MMM…"},
		{"kein Leerraum vor dem Auslassungszeichen", "Demir Ayla", 7, "Demir…"},
		{"nichts passt", "Demir", 0, "…"},
	}
	for _, f := range faelle {
		if ist := KuerzeAufBreite(jeZeichen{}, gleich, f.text, f.breite); ist != f.soll {
			t.Errorf("%s: %q auf %.0f mm ergibt %q, erwartet %q", f.name, f.text, f.breite, ist, f.soll)
		}
	}
}

// Gemessen wird der Text in der Form, die gedruckt wird: Ein Zeichen, das der Übersetzer in
// mehrere Bytes wandelt, zählt mit seiner gedruckten Breite.
func TestKuerzeAufBreite_MisstDenUebersetztenText(t *testing.T) {
	doppelt := func(s string) string { return s + s }
	if ist := KuerzeAufBreite(jeZeichen{}, doppelt, "abcdef", 8); ist != "abc…" {
		t.Errorf("mit doppelt so breitem Druckbild: %q, erwartet %q", ist, "abc…")
	}
	if ist := KuerzeAufBreite(jeZeichen{}, gleich, "Öztürk", 6); ist != "Öztürk" || utf8.RuneCountInString(ist) != 6 {
		t.Errorf("Umlaute zählen als ein Zeichen: %q", ist)
	}
}

// Von der Breite der Zelle geht ihr Rand auf beiden Seiten ab: In 13 mm passen elf Zeichen.
func TestKuerzeAufZelle_ZiehtDenRandDerZelleAb(t *testing.T) {
	for _, f := range []struct {
		text   string
		breite float64
		soll   string
	}{
		{"Demir, Ayla", 13, "Demir, Ayla"},
		{"Demir, Ayla", 12, "Demir, Ay…"},
		{"MMMMMMMMMMM", 13, "MMM…"},
	} {
		if ist := KuerzeAufZelle(jeZeichen{}, gleich, f.text, f.breite); ist != f.soll {
			t.Errorf("%q in einer Zelle von %.0f mm ergibt %q, erwartet %q", f.text, f.breite, ist, f.soll)
		}
	}
}
