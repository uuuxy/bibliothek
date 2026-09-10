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

// leseMittelJs liefert die Wert→Beschriftung-Paare aus dem MITTEL-Objekt der Oberfläche.
func leseMittelJs(t *testing.T) map[string]string {
	t.Helper()
	pfad := filepath.Join("..", "frontend", "src", "lib", "components", "bestellungen", "mittel.js")
	roh, err := os.ReadFile(pfad)
	if err != nil {
		t.Fatalf("%s lesen: %v", pfad, err)
	}
	// land: { label: 'Lernmittelfreiheit', traeger: 'Land' }
	muster := regexp.MustCompile(`(\w+):\s*\{\s*label:\s*'([^']+)'`)
	out := map[string]string{}
	for _, m := range muster.FindAllStringSubmatch(ohneKommentare(string(roh)), -1) {
		out[m[1]] = m[2]
	}
	return out
}

func TestMittelVokabularIstInGoUndFrontendDasselbe(t *testing.T) {
	ui := leseMittelJs(t)
	if len(ui) != 2 {
		t.Fatalf("%d Töpfe im Frontend erkannt, erwartet 2 — Parser oder mittel.js prüfen: %v", len(ui), ui)
	}

	for _, wert := range []string{repository.MittelLand, repository.MittelSchultraeger} {
		beschriftung, ok := ui[wert]
		if !ok {
			t.Errorf("Der Topf %q fehlt im Frontend-Vokabular (mittel.js) — die Oberfläche kann ihn nicht anbieten.", wert)
			continue
		}
		texte, err := mittelTexteFuer(wert)
		if err != nil {
			t.Errorf("%q hat keine Texte für Anschreiben und Mail: %v", wert, err)
			continue
		}
		if texte.Kurz != beschriftung {
			t.Errorf("Topf %q heißt im Frontend %q, auf Anschreiben und in der Mail aber %q — "+
				"die Bibliothekskraft sieht eine andere Bezeichnung als der Händler.", wert, beschriftung, texte.Kurz)
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
	if ui["land"] == "" || ui["schultraeger"] == "" {
		t.Fatalf("Der Parser liest die Beschriftungen nicht mehr: %v", ui)
	}
	if ui["land"] == ui["schultraeger"] {
		t.Error("Beide Töpfe tragen dieselbe Beschriftung — dann unterscheidet die Oberfläche sie nicht")
	}
}
