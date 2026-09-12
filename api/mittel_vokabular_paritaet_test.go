package api

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"bibliothek/repository"
)

// Der Topf hat EIN Vokabular und EINE Beschriftung — in Go und im Frontend.
//
// Drei Orte nennen ihn: das Go-Vokabular (repository/mittel.go, geht in die Datenbank und
// in den CHECK), die Texte für Anschreiben und Mail (api/mittel_vermerk.go) und die
// Oberfläche (frontend/.../bestellungen/mittel.js, Warenkorb-Abschnitte, Chips in Historie
// und Detail). Dasselbe Muster wie die Platzhalter-Parität der Mail-Vorlagen: Ohne Gate
// driftet einer der drei ab, und dann steht im Warenkorb ein anderer Topf als auf dem
// Papier, das beim Händler landet — die Bibliothekskraft ordnet der Bestellung dann eine
// Bezeichnung zu, die das Anschreiben nicht trägt.
//
// Geprüft wird beides: die WERTE (sie gehen über den Draht und in die Spalte) und die
// BESCHRIFTUNG (sie steht vor Menschen).

// mittelAusJs ist ein Eintrag des MITTEL-Objekts der Oberfläche.
type mittelAusJs struct{ Label, Traeger string }

// leseMittelJs liefert Beschriftung und Träger je Topf aus dem MITTEL-Objekt der
// Oberfläche. Der Träger steht seit dem 12.09.2026 auch in Go (mittelText.Traeger): Die
// Berichte überschreiben ihre Blöcke mit „Lernmittelfreiheit (Land)".
func leseMittelJs(t *testing.T) map[string]mittelAusJs {
	t.Helper()
	pfad := filepath.Join("..", "frontend", "src", "lib", "components", "bestellungen", "mittel.js")
	roh, err := os.ReadFile(pfad)
	if err != nil {
		t.Fatalf("%s lesen: %v", pfad, err)
	}
	// land: { label: 'Lernmittelfreiheit', traeger: 'Land' }
	muster := regexp.MustCompile(`(\w+):\s*\{\s*label:\s*'([^']+)',\s*traeger:\s*'([^']+)'`)
	out := map[string]mittelAusJs{}
	for _, m := range muster.FindAllStringSubmatch(ohneKommentare(string(roh)), -1) {
		out[m[1]] = mittelAusJs{Label: m[2], Traeger: m[3]}
	}
	return out
}

func TestMittelVokabularIstInGoUndFrontendDasselbe(t *testing.T) {
	ui := leseMittelJs(t)
	if len(ui) != 2 {
		t.Fatalf("%d Töpfe im Frontend erkannt, erwartet 2 — Parser oder mittel.js prüfen: %v", len(ui), ui)
	}

	for _, wert := range []string{repository.MittelLand, repository.MittelSchultraeger} {
		eintrag, ok := ui[wert]
		if !ok {
			t.Errorf("Der Topf %q fehlt im Frontend-Vokabular (mittel.js) — die Oberfläche kann ihn nicht anbieten.", wert)
			continue
		}
		texte, err := mittelTexteFuer(wert)
		if err != nil {
			t.Errorf("%q hat keine Texte für Anschreiben und Mail: %v", wert, err)
			continue
		}
		if texte.Kurz != eintrag.Label {
			t.Errorf("Topf %q heißt im Frontend %q, auf Anschreiben und in der Mail aber %q — "+
				"die Bibliothekskraft sieht eine andere Bezeichnung als der Händler.", wert, eintrag.Label, texte.Kurz)
		}
		// Der Träger steht in beiden: im Chip der Oberfläche und über dem Block im
		// Bericht. Zwei Schreibweisen („Schulträger" / „Träger") ließen die Zahl in der
		// Abrechnung anders heißen als auf dem Bildschirm, gegen den sie geprüft wird.
		if texte.Traeger != eintrag.Traeger {
			t.Errorf("Topf %q trägt im Frontend %q, in den Berichten aber %q.",
				wert, eintrag.Traeger, texte.Traeger)
		}
	}

	// Rückrichtung: ein Wert im Frontend, den Go nicht kennt, endet an der Tür in einem
	// 400 — laut, aber erst beim Auslösen der Bestellung.
	for wert := range ui {
		if !repository.MittelGueltig(wert) {
			t.Errorf("Das Frontend führt den Topf %q, den der Server nicht kennt (repository.MittelGueltig) — "+
				"eine Bestellung daraus wird an der Tür abgewiesen.", wert)
		}
	}
}

// Gegenprobe am Detektor: Ein Parser, der nichts findet, meldet ewig „alles gut".
func TestMittelVokabularDetektorGreift(t *testing.T) {
	ui := leseMittelJs(t)
	if ui["land"].Label == "" || ui["schultraeger"].Label == "" ||
		ui["land"].Traeger == "" || ui["schultraeger"].Traeger == "" {
		t.Fatalf("Der Parser liest Beschriftung oder Träger nicht mehr: %v", ui)
	}
	if ui["land"] == ui["schultraeger"] {
		t.Error("Beide Töpfe tragen dieselbe Beschriftung — dann unterscheidet die Oberfläche sie nicht")
	}
}
