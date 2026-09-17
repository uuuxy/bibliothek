package docs

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"testing"

	"bibliothek/pkg/ersatzwert"
)

// Gate gegen eine Staffel, die im Konzept anders steht als im Code.
//
// Anlass (17.09.2026): mittel_konzept.md nannte die Staffel an ZWEI Stellen und
// widersprach sich dabei selbst. Abschnitt 4.6 schrieb „6+→10 %" (richtig), Abschnitt 1.1
// schrieb in Prosa „ab dem 5. Jahr 10 %" (falsch). Die Arbeitshilfe zum Erlass vom
// 17.12.2014 sagt: „Nach 5 Jahren und für jedes weitere Jahr werden insgesamt 10 % des
// Neupreises berechnet" — also ab dem SECHSTEN Verleihjahr. pkg/ersatzwert rechnet seit
// jeher richtig; nur das erklärende Dokument war falsch.
//
// Warum das gefährlich ist: Diese Prozentzahlen stehen am Ende in einem Bescheid an
// Erziehungsberechtigte. Wer das Konzept liest, um eine Änderung zu bauen — etwa den
// Staffel-Vorschlag im Schaden-Dialog (OFFEN.md 5.4) — baut nach der Prosa und nicht nach
// pkg/ersatzwert. Ein Test am Code allein ist dagegen blind: ersatzwert_test.go war die
// ganze Zeit grün, während das Dokument daneben etwas anderes behauptete.
//
// Seitdem steht die Staffel an beiden Stellen in DERSELBEN maschinenlesbaren Form
// (`Verleihjahr→Prozent`), und dieses Gate hält jedes Paar gegen ersatzwert.Rechne.
// Eine Zahl im Dokument zu ändern, ohne den Code zu ändern, macht es rot.
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
			ist := ersatzwert.Rechne(j, 100, 100).Prozent
			if ist != erwartet {
				t.Errorf("mittel_konzept.md schreibt %s→%d %%, ersatzwert.Rechne rechnet für "+
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
