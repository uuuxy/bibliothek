package service

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

// Zwei Rückrechnungen eines Littera-Etiketts, eine Wahrheit.
//
// Online rechnet der Server ein 13-stelliges Littera-Etikett auf die Exemplarnummer zurück
// (dekodiereLitteraEtikett). Ohne Netz muss der Theken-Rechner dasselbe tun
// (frontend/src/lib/litteraEtikett.js) und die Nummer dann in der Barcode-Liste nachschlagen:
// Die Liste führt nur die Nummern, und auf dem Server tragen 30.658 von 34.777 Exemplaren die
// nackte Nummer (Rasterdurchgang 15.09.2026, OFFEN.md 5.15). Rechneten die beiden Seiten
// verschieden, gälte ein Buch offline als unklar, das online gebucht wird — oder umgekehrt.
//
// Beide Seiten lesen dieselben Fälle. Dieser Test prüft die Go-Seite und dass der Vitest die
// Datei wirklich einliest; der Vitest prüft die JavaScript-Seite.
func TestLitteraEtikett_GoUndJavaScriptTeilenDiePrueffaelle(t *testing.T) {
	const faelleDatei = "../../frontend/src/lib/litteraEtikett.faelle.json"
	roh, err := os.ReadFile(faelleDatei)
	if err != nil {
		t.Fatalf("Prüffälle lesen: %v", err)
	}
	var pruefung struct {
		Faelle []struct {
			Scan   string  `json:"scan"`
			Nummer *string `json:"nummer"`
			Fall   string  `json:"fall"`
		} `json:"faelle"`
	}
	if err := json.Unmarshal(roh, &pruefung); err != nil {
		t.Fatalf("Prüffälle: %v", err)
	}
	gueltig, ungueltig := 0, 0
	for _, f := range pruefung.Faelle {
		nummer, ok := dekodiereLitteraEtikett(f.Scan)
		if f.Nummer == nil {
			ungueltig++
			if ok {
				t.Errorf("%s: %q ergibt %q, erwartet kein Etikett", f.Fall, f.Scan, nummer)
			}
			continue
		}
		gueltig++
		if !ok || nummer != *f.Nummer {
			t.Errorf("%s: %q ergibt %q (%v), erwartet %q", f.Fall, f.Scan, nummer, ok, *f.Nummer)
		}
	}
	if gueltig < 5 || ungueltig < 5 {
		t.Fatalf("%d gültige und %d ungültige Fälle — zu wenige, um beide Seiten zu binden", gueltig, ungueltig)
	}

	vitest, err := os.ReadFile("../../frontend/src/lib/litteraEtikett.test.js")
	if err != nil {
		t.Fatalf("Vitest lesen: %v", err)
	}
	if !strings.Contains(string(vitest), "./litteraEtikett.faelle.json") {
		t.Error("litteraEtikett.test.js liest die gemeinsamen Prüffälle nicht mehr ein — dann prüft nur noch die Go-Seite")
	}
}
