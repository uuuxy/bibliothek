package code39

import (
	"encoding/json"
	"os"
	"testing"
)

// Zwei Umkehrungen des Prüfzeichens, eine Wahrheit.
//
// Online entscheidet der Server, ob ein Scan ein Aufdruck von früher ist
// (internal/service/omnibox_service.go). Ohne Netz muss der Theken-Rechner dasselbe tun
// (frontend/src/lib/code39Pruefzeichen.js) — sonst gälte ein Buch offline als unklar, das
// online gebucht wird, oder umgekehrt. Dieselbe Bauart wie bei den Littera-Etiketten.
//
// Beide Seiten lesen dieselben Fälle. Dieser Test prüft die Go-Seite und dass der Vitest
// die Datei wirklich einliest; der Vitest prüft die JavaScript-Seite.
func TestCode39_GoUndJavaScriptTeilenDiePrueffaelle(t *testing.T) {
	const faelleDatei = "../../frontend/src/lib/code39.faelle.json"
	roh, err := os.ReadFile(faelleDatei)
	if err != nil {
		t.Fatalf("Prüffälle lesen: %v", err)
	}
	var pruefung struct {
		Faelle []struct {
			Scan string  `json:"scan"`
			Kern *string `json:"kern"`
			Fall string  `json:"fall"`
		} `json:"faelle"`
	}
	if err := json.Unmarshal(roh, &pruefung); err != nil {
		t.Fatalf("Prüffälle: %v", err)
	}

	mitZeichen, ohneZeichen := 0, 0
	for _, f := range pruefung.Faelle {
		kern, hat := OhnePruefzeichen(f.Scan)

		if f.Kern == nil {
			ohneZeichen++
			if hat {
				t.Errorf("%s: %q ergab %q — erwartet: kein abtrennbares Prüfzeichen",
					f.Fall, f.Scan, kern)
			}
			if kern != f.Scan {
				t.Errorf("%s: %q kam als %q zurück — ein Scan ohne Prüfzeichen bleibt, wie er ist",
					f.Fall, f.Scan, kern)
			}
			continue
		}

		mitZeichen++
		if !hat {
			t.Errorf("%s: %q erkennt kein Prüfzeichen — erwartet %q", f.Fall, f.Scan, *f.Kern)
			continue
		}
		if kern != *f.Kern {
			t.Errorf("%s: %q ergab %q, erwartet %q", f.Fall, f.Scan, kern, *f.Kern)
		}
	}

	// Sanity-Floor wie bei den Littera-Fällen: Eine Datei, aus der nichts mehr gelesen
	// wird (umbenanntes Feld, geändertes Format), darf nicht still grün durchlaufen.
	if mitZeichen < 5 || ohneZeichen < 4 {
		t.Fatalf("nur %d Fälle MIT und %d OHNE Prüfzeichen gelesen — greift der Scanner "+
			"noch auf code39.faelle.json?", mitZeichen, ohneZeichen)
	}
}
