package auskunft

import (
	"encoding/json"
	"os"
	"testing"
)

// Das Blatt schreibt jede Art eines Lesers aus. Die Arten stehen in den Prüffällen, die Server
// und Browser beide lesen; eine Art, die dsgvoLeserart nicht kennt, stünde mit ihrem Wert aus
// der Datenbank auf dem Blatt.
func TestDsgvoLeserart_SchreibtJedeArtAus(t *testing.T) {
	const faelleDatei = "../../frontend/src/lib/leserArt.faelle.json"
	roh, err := os.ReadFile(faelleDatei)
	if err != nil {
		t.Fatalf("Prüffälle lesen: %v", err)
	}
	var pruefung struct {
		Arten []struct {
			Art string `json:"art"`
		} `json:"arten"`
	}
	if err := json.Unmarshal(roh, &pruefung); err != nil {
		t.Fatalf("Prüffälle: %v", err)
	}
	if len(pruefung.Arten) < 7 {
		t.Fatalf("%d Arten — erwartet mindestens 7: liest der Test noch auf leserArt.faelle.json?",
			len(pruefung.Arten))
	}
	for _, a := range pruefung.Arten {
		if dsgvoLeserart(a.Art) == a.Art {
			t.Errorf("%s: Die Auskunft schreibt die Art nicht aus", a.Art)
		}
	}
}
