package docs

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"testing"

	"bibliothek/pkg/ersatzwert"
)

// Gate gegen eine Staffel, die im Konzept anders steht als im Code. Die Prozentzahlen stehen
// am Ende in einem Bescheid an Erziehungsberechtigte, und wer das Konzept liest, um eine
// Änderung zu bauen, baut nach dem Dokument; ein Test am Code allein sieht den Widerspruch
// nicht. Die Staffel steht in mittel_konzept.md an zwei Stellen in derselben
// maschinenlesbaren Form (`Verleihjahr→Prozent`), und das Gate hält jedes Paar gegen
// ersatzwert.Berechne.
//
// Maßstab ist die Arbeitshilfe zum Erlass vom 17.12.2014: „Nach 5 Jahren und für jedes
// weitere Jahr werden insgesamt 10 % des Neupreises berechnet", also ab dem sechsten
// Verleihjahr.
//
// Reparatur bei Rot: Zuerst die Arbeitshilfe lesen, nicht die jeweils andere Stelle
// abschreiben. Stimmt der Code, gehört das Dokument nachgezogen; stimmt das Dokument,
// ist es ein Fehler in pkg/ersatzwert und betrifft echtes Geld.
func TestStaffelImKonzeptStimmtMitDerRechnung(t *testing.T) {
	konzept, err := os.ReadFile("mittel_konzept.md")
	if err != nil {
		t.Fatalf("mittel_konzept.md lesen: %v", err)
	}

	// `1→100 %`, `6+→10 %` — das Pluszeichen markiert „und jedes weitere Jahr".
	muster := regexp.MustCompile(`(\d+)(\+?)→(\d+)\s*%`)
	treffer := muster.FindAllStringSubmatch(string(konzept), -1)

	// Sanity-Floor nach dem Muster von TestInvariantenFundstellenExistieren: Greift der
	// Scanner durch eine geänderte Schreibweise plötzlich nicht mehr, ist das Gate
	// faktisch abgeschaltet. Zwei Stellen mit je sechs Paaren = zwölf.
	if len(treffer) < 12 {
		t.Fatalf("nur %d Staffel-Angaben in mittel_konzept.md erkannt (erwartet ≥12) — "+
			"steht die Staffel dort noch in der Form `Verleihjahr→Prozent`?", len(treffer))
	}

	for _, paar := range treffer {
		jahr, err := strconv.Atoi(paar[1])
		if err != nil {
			t.Fatalf("Verleihjahr %q ist keine Zahl: %v", paar[1], err)
		}
		erwartet, err := strconv.Atoi(paar[3])
		if err != nil {
			t.Fatalf("Prozentsatz %q ist keine Zahl: %v", paar[3], err)
		}

		// Basis 100/100: Der Prozentsatz ist dann der Betrag, unabhängig davon, welche
		// Basis die Staffel für dieses Jahr wählt. Geprüft wird die Staffel, nicht die
		// Wahl zwischen Kauf- und Neupreis (dafür gibt es ersatzwert_test.go).
		pruefe := func(j int) {
			t.Helper()
			ist := ersatzwert.Berechne(j, 10000, 10000, 0, ersatzwert.PreisquelleListenpreis).Prozent
			if ist != erwartet {
				t.Errorf("mittel_konzept.md schreibt %s→%d %%, ersatzwert.Berechne rechnet für "+
					"das %d. Verleihjahr %d %% — eine der beiden Stellen ist falsch, und die "+
					"Zahl steht in einem Bescheid an Eltern",
					paar[1]+paar[2], erwartet, j, ist)
			}
		}
		pruefe(jahr)

		// `6+` heißt „und jedes weitere Jahr" — also muss der Satz auch danach halten.
		if paar[2] == "+" {
			for _, weiter := range []int{jahr + 1, jahr + 5} {
				t.Run(fmt.Sprintf("%d+_gilt_auch_im_%d._Jahr", jahr, weiter), func(t *testing.T) {
					pruefe(weiter)
				})
			}
		}
	}
}
