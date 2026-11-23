package pdf

import (
	"bytes"
	"slices"
	"strings"
	"testing"

	"bibliothek/internal/pkg/pdfzeichen"
)

func signaturBogen1(t *testing.T, zeilen []string) []etikettStueck {
	t.Helper()
	var puffer bytes.Buffer
	if err := ErzeugeSignaturEtiketten(1, [][]string{zeilen}).Output(&puffer); err != nil {
		t.Fatalf("PDF erzeugen: %v", err)
	}
	return etikettStuecke(t, puffer.Bytes())
}

// Das Signaturetikett trägt die Aufschrift vom Buchrücken: Regaladresse über Kürzel, ohne
// Strichcode und ohne Nummer. Kein Text reicht näher als einen Millimeter an die Ober- oder
// Unterkante, und die Zeilen liegen nicht übereinander.
func TestSignaturEtikett_ZeilenAufDemEtikett(t *testing.T) {
	f := etikettFormate[signaturBogen]
	oben, unten := f.RandOben, f.RandOben+f.Hoehe
	for _, zeilen := range [][]string{
		{"Sk", "Bie"},
		{"LMF Deu 7", "Bär"},
		{"PoWi", "Ich / Du"},
		{"Sk"},
		{"LMF Geschichte Oberstufe 13", "Wic"},
	} {
		t.Run(strings.Join(zeilen, "|"), func(t *testing.T) {
			stuecke := signaturBogen1(t, zeilen)
			var texte []string
			for _, s := range stuecke {
				if s.bild {
					t.Fatalf("ein Bild auf dem Signaturetikett: %s", beschreibe(s))
				}
				texte = append(texte, s.text)
			}
			if !slices.Equal(texte, zeilen) {
				t.Fatalf("Zeilen %q, erwartet %q", texte, zeilen)
			}
			for _, s := range stuecke {
				if s.oben < oben+1 || s.unten > unten-1 || s.links < f.RandLinks {
					t.Errorf("%s reicht über die Kante: %.1f–%.1f mm, das Etikett reicht von %.1f bis %.1f",
						beschreibe(s), s.oben, s.unten, oben, unten)
				}
			}
			if len(stuecke) == 2 && stuecke[0].unten > stuecke[1].oben {
				t.Errorf("%s und %s liegen übereinander", beschreibe(stuecke[0]), beschreibe(stuecke[1]))
			}
		})
	}
}

// Eine lange Regaladresse wird kleiner gesetzt, bis sie in die Breite passt, statt auf das
// Nachbaretikett zu laufen; eine kurze steht in der großen Schrift.
func TestSignaturEtikett_SchriftPasstSichDerBreiteAn(t *testing.T) {
	f := etikettFormate[signaturBogen]
	for _, fall := range []struct {
		zeilen      []string
		laengste    string
		verkleinert bool
	}{
		{[]string{"Sk", "Bie"}, "Bie", false},
		{[]string{"LMF Geschichte Oberstufe 13", "Wic"}, "LMF Geschichte Oberstufe 13", true},
	} {
		pdf := pdfzeichen.NewDokument()
		pdf.AddPage()
		zeichneSignaturEtikett(pdf, pdfzeichen.Ersetze, f, fall.zeilen, etikettPosition{f.RandLinks, f.RandOben})
		groesse, _ := pdf.GetFontSize()
		if (groesse < signaturSchriftGross) != fall.verkleinert {
			t.Errorf("%q: Schrift %.1f pt, verkleinert erwartet: %v", fall.zeilen, groesse, fall.verkleinert)
		}
		if breite := pdf.GetStringWidth(pdfzeichen.Ersetze(fall.laengste)); breite > f.Breite-2*signaturRand {
			t.Errorf("%q: %.1f mm breit, das Etikett hat %.1f mm ohne Rand", fall.laengste, breite, f.Breite-2*signaturRand)
		}
	}
}
