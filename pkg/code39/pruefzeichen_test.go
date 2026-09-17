package code39

import (
	"fmt"
	"testing"

	"github.com/boombuler/barcode"
	"github.com/boombuler/barcode/code39"
)

// Die Gegenprobe gegen die Bibliothek, die den Aufdruck TATSÄCHLICH erzeugt hat.
//
// Eine selbst nachgebaute Mod-43-Rechnung, die nur gegen sich selbst geprüft wird, ist
// wertlos: Sie wäre genau so richtig oder falsch wie die Annahme, die ihr zugrunde liegt.
//
// Geprüft wird deshalb an den BALKEN, nicht am Text: `Encode(inhalt, true, …)` hängt das
// Prüfzeichen selbst an, `Encode(inhalt+zeichen, false, …)` nimmt es als Teil des Inhalts.
// Stimmt meine Rechnung, sind beide Bilder Pixel für Pixel gleich — und wenn nicht, ist
// es meine Rechnung, die falsch liegt.
//
// Warum nicht über Content(): Die Bibliothek merkt sich dort den EINGABETEXT ohne
// Prüfzeichen (am 17.09.2026 gemessen). Ein Test, der Content() vergleicht, wäre still
// grün geblieben, ohne je ein Prüfzeichen gesehen zu haben.
func TestPruefzeichenStimmtMitDemEchtenDrucker(t *testing.T) {
	inhalte := []string{
		"B-10001", "A-10003", "S-10001", "L-42", "LMF-77", "G-1",
		"B-99999", "123456", "B-100000", "ABC-123",
	}

	for _, inhalt := range inhalte {
		t.Run(inhalt, func(t *testing.T) {
			zeichen, ok := Pruefzeichen(inhalt)
			if !ok {
				t.Fatalf("Pruefzeichen(%q) scheiterte", inhalt)
			}
			gedruckt := inhalt + string(zeichen)

			// Genau der Aufruf von früher (api/barcode_generate.go bis 17.09.2026):
			// includeChecksum=true, fullASCIIMode=true.
			mitAutomatik, err := code39.Encode(inhalt, true, true)
			if err != nil {
				t.Fatalf("Encode(%q, mit Prüfzeichen): %v", inhalt, err)
			}
			// Dasselbe, aber das Prüfzeichen steht schon im Inhalt — die Bibliothek hängt
			// dann keines an.
			mitMeinem, err := code39.Encode(gedruckt, false, true)
			if err != nil {
				t.Fatalf("Encode(%q, ohne Automatik): %v", gedruckt, err)
			}

			gleicheBalken(t, mitAutomatik, mitMeinem, inhalt, gedruckt)

			// Und die Umkehrung führt zurück auf den Aufdruck.
			kern, hat := OhnePruefzeichen(gedruckt)
			if !hat {
				t.Fatalf("OhnePruefzeichen(%q) erkennt kein Prüfzeichen — der Scan eines "+
					"alten Etiketts bliebe damit unauffindbar", gedruckt)
			}
			if kern != inhalt {
				t.Errorf("OhnePruefzeichen(%q) = %q, want %q", gedruckt, kern, inhalt)
			}
		})
	}
}

// gleicheBalken vergleicht zwei Barcodes Pixel für Pixel.
func gleicheBalken(t *testing.T, a, b barcode.Barcode, inhalt, gedruckt string) {
	t.Helper()
	if a.Bounds() != b.Bounds() {
		t.Fatalf("verschieden breit: Encode(%q, mit Automatik) ist %v, Encode(%q, ohne) ist "+
			"%v — mein Prüfzeichen ist nicht das der Bibliothek",
			inhalt, a.Bounds(), gedruckt, b.Bounds())
	}
	r := a.Bounds()
	for x := r.Min.X; x < r.Max.X; x++ {
		for y := r.Min.Y; y < r.Max.Y; y++ {
			if a.At(x, y) != b.At(x, y) {
				t.Fatalf("Balken unterscheiden sich bei (%d,%d): Die Bibliothek hängt an %q "+
					"ein anderes Prüfzeichen an als ich (%q)", x, y, inhalt, gedruckt)
			}
		}
	}
}

// Was NICHT als Prüfzeichen durchgehen darf.
func TestOhnePruefzeichenLaesstFremdesInRuhe(t *testing.T) {
	faelle := []struct {
		scan  string
		grund string
	}{
		{"", "leer"},
		{"B", "ein Zeichen"},
		{"B-", "zwei Zeichen — nach dem Kürzen bliebe eines übrig"},
		{"4006381333931", "eine EAN-13 vom Verlag; ihre Prüfziffer ist eine andere Rechnung"},
		{"B-10001", "unser eigener Aufdruck OHNE Prüfzeichen — Code 128 seit 17.09.2026"},
		{"müller", "Kleinbuchstaben und Umlaut kennt Code 39 nicht"},
	}

	for _, f := range faelle {
		t.Run(f.grund, func(t *testing.T) {
			got, hat := OhnePruefzeichen(f.scan)
			if hat {
				t.Errorf("OhnePruefzeichen(%q) = %q, true — erkannt als Prüfzeichen, obwohl: %s",
					f.scan, got, f.grund)
			}
			if got != f.scan {
				t.Errorf("OhnePruefzeichen(%q) veränderte den Scan zu %q", f.scan, got)
			}
		})
	}
}

// Die beiden Werte, die am 17.09.2026 an echten Ausdrucken gemessen wurden — mit zwei
// unabhängigen Erkennern und am fertigen PDF des Druck-Centers. Sie stehen hier als
// feste Zahlen, damit ein Umbau der Rechnung an der MESSUNG scheitert und nicht erst an
// einer zweiten Rechnung, die denselben Fehler machen könnte.
func TestGemesseneWerteVomEchtenAusdruck(t *testing.T) {
	for gedruckt, erwartet := range map[string]string{
		"B-100016": "B-10001",
		"A-100037": "A-10003",
	} {
		kern, hat := OhnePruefzeichen(gedruckt)
		if !hat || kern != erwartet {
			t.Errorf("OhnePruefzeichen(%q) = %q, %v — want %q, true (am 17.09.2026 an einem "+
				"echten Ausdruck gemessen)", gedruckt, kern, hat, erwartet)
		}
	}
}

// Belegt, dass die Rechnung über den ganzen Zeichensatz trägt und nicht nur für Ziffern.
func TestJedesPruefzeichenDesZeichensatzes(t *testing.T) {
	gesehen := map[byte]bool{}
	for i := 0; i < 200; i++ {
		inhalt := fmt.Sprintf("B-%d", i)
		zeichen, ok := Pruefzeichen(inhalt)
		if !ok {
			t.Fatalf("Pruefzeichen(%q) scheiterte", inhalt)
		}
		gesehen[zeichen] = true
	}
	// Über 200 Nummern müssen deutlich mehr als eine Handvoll verschiedener Prüfzeichen
	// auftreten — sonst rechnet die Funktion nicht, sondern gibt etwas Festes zurück.
	if len(gesehen) < 20 {
		t.Errorf("nur %d verschiedene Prüfzeichen über 200 Nummern — die Rechnung greift "+
			"vermutlich nicht", len(gesehen))
	}
}
