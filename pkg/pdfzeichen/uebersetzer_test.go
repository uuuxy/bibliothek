package pdfzeichen

import (
	"testing"

	"golang.org/x/text/encoding/charmap"
)

// cp1252Nachbau verhält sich wie der Übersetzer von gofpdf: Was Windows-1252 kennt,
// bleibt (hier über die echte Tabelle, nicht über eine Auswahl von Hand — í, â, ø
// gehören dazu), alles andere wird zum Punkt. Der echte Übersetzer hängt an einem
// Dokument; das Verhalten, um das es geht, ist der Punkt.
func cp1252Nachbau(text string) string {
	bekannt := map[rune]bool{}
	for b := 0x80; b <= 0xFF; b++ {
		if r := charmap.Windows1252.DecodeByte(byte(b)); r != '�' {
			bekannt[r] = true
		}
	}
	out := make([]rune, 0, len(text))
	for _, r := range text {
		switch {
		case r < 0x80, bekannt[r]:
			out = append(out, r)
		default:
			out = append(out, '.')
		}
	}
	return string(out)
}

func TestUebersetzer_ErsetztWasCp1252NichtKennt(t *testing.T) {
	tr := Uebersetzer(cp1252Nachbau)
	faelle := map[string]string{
		"Ayşe Öztürk":          "Ayse Öztürk",
		"Łukasz Wiśniewski":    "Lukasz Wisniewski",
		"Ağaoğlu, İlknur":      "Agaoglu, Ilknur",
		"Nagy Őrs":             "Nagy Ors",
		"Ștefan Țăran":         "Stefan Taran",
		"Đorđe Šimić":          "Dorde Simic",
		"Ľubomír Ďuriš":        "Lubomír Duris",
		"Ūla Ėglė Vaišvilaitė": "Ula Egle Vaisvilaite",
		"Müller, François":     "Müller, François", // alles in cp1252 — bleibt, wie es ist
		"Deutschbuch 5 (LMF)":  "Deutschbuch 5 (LMF)",
	}
	for eingabe, erwartet := range faelle {
		if got := tr(eingabe); got != erwartet {
			t.Errorf("%q → %q, erwartet %q", eingabe, got, erwartet)
		}
	}
}

// Was die Liste nicht kennt, bleibt beim Verhalten des Übersetzers — der Punkt. Das ist
// die Grenze der Ersetzung, nicht ihr Fehler: Für kyrillische oder griechische Namen
// gibt es keine Ersetzung, die ein Etikett noch lesbar macht.
func TestUebersetzer_LaesstUnbekanntesBeimPunkt(t *testing.T) {
	tr := Uebersetzer(cp1252Nachbau)
	if got := tr("Дмитрий"); got != "......." {
		t.Errorf("kyrillisch: %q, erwartet sieben Punkte", got)
	}
}
