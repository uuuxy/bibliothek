package mitteltopf

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

// Der Topf hat ein Vokabular und eine Beschriftung, in Go und in der Oberfläche
// (bestellungen/mittel.js: Abschnitte im Warenkorb, Chips in Historie und Detail). Liefen sie
// auseinander, stünde im Warenkorb ein anderer Topf als auf dem Papier, das beim Händler
// landet. Geprüft werden die Werte (sie gehen über den Draht und in die Spalte) und die Wörter
// (sie stehen vor Menschen).

// mittelAusJs ist ein Eintrag des MITTEL-Objekts der Oberfläche.
type mittelAusJs struct{ Label, Traeger string }

// Ein Eintrag steht am Anfang seiner Zeile; eine auskommentierte Zeile zählt nicht.
// land: { label: 'Lernmittelfreiheit', traeger: 'Land' }
var mittelJsEintrag = regexp.MustCompile(`(?m)^\s*(\w+):\s*\{\s*label:\s*'([^']+)',\s*traeger:\s*'([^']+)'`)

func mittelEintraege(quelle string) map[string]mittelAusJs {
	aus := map[string]mittelAusJs{}
	for _, m := range mittelJsEintrag.FindAllStringSubmatch(quelle, -1) {
		aus[m[1]] = mittelAusJs{Label: m[2], Traeger: m[3]}
	}
	return aus
}

// leseMittelJs liefert Beschriftung und Träger je Topf aus dem MITTEL-Objekt der Oberfläche.
func leseMittelJs(t *testing.T) map[string]mittelAusJs {
	t.Helper()
	pfad := filepath.Join("..", "..", "frontend", "src", "lib", "components", "bestellungen", "mittel.js")
	roh, err := os.ReadFile(pfad)
	if err != nil {
		t.Fatalf("%s lesen: %v", pfad, err)
	}
	return mittelEintraege(string(roh))
}

func TestVokabularIstInGoUndOberflaecheDasselbe(t *testing.T) {
	ui := leseMittelJs(t)
	if len(ui) != 2 {
		t.Fatalf("%d Töpfe in der Oberfläche erkannt, erwartet 2 — Parser oder mittel.js prüfen: %v", len(ui), ui)
	}

	for _, wert := range []string{Land, Schultraeger} {
		eintrag, ok := ui[wert]
		if !ok {
			t.Errorf("Der Topf %q fehlt in mittel.js — die Oberfläche kann ihn nicht anbieten.", wert)
			continue
		}
		texte, err := TexteFuer(wert)
		if err != nil {
			t.Errorf("%q hat keine Texte für Anschreiben und Mail: %v", wert, err)
			continue
		}
		if texte.Kurz != eintrag.Label {
			t.Errorf("Topf %q heißt in der Oberfläche %q, auf Anschreiben und in der Mail aber %q — "+
				"die Bibliothekskraft sieht eine andere Bezeichnung als der Händler.", wert, eintrag.Label, texte.Kurz)
		}
		// Der Träger steht im Chip der Oberfläche und über dem Block im Bericht. Mit zwei
		// Schreibweisen hieße die Zahl in der Abrechnung anders als auf dem Bildschirm, gegen den
		// sie geprüft wird.
		if texte.Traeger != eintrag.Traeger {
			t.Errorf("Topf %q trägt in der Oberfläche %q, in den Berichten aber %q.",
				wert, eintrag.Traeger, texte.Traeger)
		}
	}

	// Rückrichtung: Ein Wert der Oberfläche, den Go nicht kennt, endet erst beim Auslösen der
	// Bestellung an der Tür.
	for wert := range ui {
		if !Gueltig(wert) {
			t.Errorf("Die Oberfläche führt den Topf %q, den der Server nicht kennt (Gueltig) — "+
				"eine Bestellung daraus wird an der Tür abgewiesen.", wert)
		}
	}
}

// Gegenprobe am Detektor: Ein Parser, der nichts findet, meldet für immer „alles gut".
func TestVokabularDetektorGreift(t *testing.T) {
	ui := leseMittelJs(t)
	if ui["land"].Label == "" || ui["schultraeger"].Label == "" ||
		ui["land"].Traeger == "" || ui["schultraeger"].Traeger == "" {
		t.Fatalf("Der Parser liest Beschriftung oder Träger nicht mehr: %v", ui)
	}
	if ui["land"] == ui["schultraeger"] {
		t.Error("Beide Töpfe tragen dieselbe Beschriftung — dann unterscheidet die Oberfläche sie nicht")
	}

	// Jede Schreibweise eines Eintrags wird gelesen, eine auskommentierte Zeile nicht.
	gelesen := mittelEintraege("export const MITTEL = {\n" +
		"\tland: { label: 'A', traeger: 'B' },\n" +
		"  kreis:{label:'C',traeger:'D'},\n" +
		"\t// alt: { label: 'E', traeger: 'F' },\n" +
		"\tschultraeger: { label: 'G', traeger: 'H' } // Kommentar dahinter\n" +
		"};\n")
	will := map[string]mittelAusJs{"land": {"A", "B"}, "kreis": {"C", "D"}, "schultraeger": {"G", "H"}}
	if len(gelesen) != len(will) {
		t.Fatalf("gelesen %v, erwartet %v", gelesen, will)
	}
	for wert, eintrag := range will {
		if gelesen[wert] != eintrag {
			t.Errorf("%s: gelesen %v, erwartet %v", wert, gelesen[wert], eintrag)
		}
	}
}
