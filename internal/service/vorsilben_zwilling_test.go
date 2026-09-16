package service

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// Die Vorsilben eines Scans stehen ZWEIMAL im Baum — und müssen dasselbe heißen.
//
// Ohne Netz ordnet der Theken-Rechner jeden Scan selbst ein (frontend/src/lib/
// scanEinordnen.js): `A-`/`S-`/`L-` sind ein Ausweis, `B-`/`LMF-` ein Buch, `G-` ein
// Gerät. Mit Netz entscheidet der Switch in omnibox_service.go dieselbe Frage. Bis zum
// 17.09.2026 hielt die beiden Seiten nur ein Kommentar zusammen („dieselbe Menge wie im
// Server-Switch"), und schon beim Aufschreiben dieses Tests stimmte das nicht mehr:
// `LMF-` fehlte im Switch und kam nur über den Umweg resolveOhnePraefix zum selben
// Ergebnis.
//
// Was eine Abweichung kostet: Eine neue Vorsilbe an EINER Stelle fällt zwar laut aus
// (ohne Netz „unklar", beim Nachbuchen „nicht gebucht") — aber erst im Betrieb, an der
// Theke, vor einem Kind. Das Muster für dieses Gate steht im Haus:
// internal/littera/platzhalter_domain_test.go hält zwei Konstanten gegeneinander; hier
// sind es zwei Sprachen, also wird die JS-Datei gelesen.
//
// Der Test prüft die BEDEUTUNG, nicht die Schreibweise: Er ordnet jeder Vorsilbe auf
// beiden Seiten eine Art zu (ausweis · buch · geraet) und vergleicht die Zuordnungen.

func TestVorsilben_OhneNetzUndMitNetzBedeutenDasselbe(t *testing.T) {
	js := vorsilbenAusScanEinordnen(t)
	go_ := vorsilbenAusOmniboxSwitch(t)

	for vorsilbe, art := range js {
		if go_[vorsilbe] == "" {
			t.Errorf("%q ist ohne Netz ein %s, der Server-Switch kennt die Vorsilbe nicht — "+
				"derselbe Scan bedeutet mit und ohne Netz Verschiedenes", vorsilbe, art)
			continue
		}
		if go_[vorsilbe] != art {
			t.Errorf("%q ist ohne Netz ein %s, mit Netz ein %s", vorsilbe, art, go_[vorsilbe])
		}
	}
	for vorsilbe, art := range go_ {
		if js[vorsilbe] == "" {
			t.Errorf("%q ist mit Netz ein %s, ohne Netz wird der Scan als unklar abgewiesen — "+
				"die Theke nimmt ihn dann nur an, solange das Netz steht", vorsilbe, art)
		}
	}

	if len(js) == 0 || len(go_) == 0 {
		t.Fatalf("nichts gemessen (js=%d, go=%d) — der Detektor liest ins Leere, und dieser "+
			"Test wäre ab sofort immer grün", len(js), len(go_))
	}
}

// Jede Vorsilbe muss auch in der Vereinheitlichung stehen: Sonst bleibt ein klein
// getippter oder klein gescannter Wert (`lmf-1234`) unerkannt und wird „unklar".
func TestVorsilben_JedeStehtAuchInDerVereinheitlichung(t *testing.T) {
	quelle := leseScanEinordnen(t)
	liste := listeHinter(t, quelle, `for \(const v of \[([^\]]*)\]\)`)
	bekannt := map[string]bool{}
	for _, v := range liste {
		bekannt[v] = true
	}
	for vorsilbe := range vorsilbenAusScanEinordnen(t) {
		if !bekannt[vorsilbe] {
			t.Errorf("%q wird eingeordnet, aber nicht vereinheitlicht — `%s` in Kleinschreibung "+
				"bliebe unklar", vorsilbe, strings.ToLower(vorsilbe)+"1234")
		}
	}
}

func leseScanEinordnen(t *testing.T) string {
	t.Helper()
	pfad := filepath.Join("..", "..", "frontend", "src", "lib", "scanEinordnen.js")
	roh, err := os.ReadFile(pfad)
	if err != nil {
		t.Fatalf("%s nicht lesbar: %v", pfad, err)
	}
	return string(roh)
}

// listeHinter zieht die Zeichenketten aus der ersten Klammer eines Treffers.
func listeHinter(t *testing.T, quelle, muster string) []string {
	t.Helper()
	m := regexp.MustCompile(muster).FindStringSubmatch(quelle)
	if m == nil {
		t.Fatalf("Muster %q steht nicht mehr in scanEinordnen.js — der Detektor misst nichts", muster)
	}
	var out []string
	for _, s := range regexp.MustCompile(`'([^']+)'`).FindAllStringSubmatch(m[1], -1) {
		out = append(out, s[1])
	}
	sort.Strings(out)
	return out
}

// vorsilbenAusScanEinordnen liest die Einordnung der Theke ohne Netz.
func vorsilbenAusScanEinordnen(t *testing.T) map[string]string {
	t.Helper()
	quelle := leseScanEinordnen(t)
	arten := map[string]string{}
	for _, v := range listeHinter(t, quelle, `AUSWEIS_VORSILBEN = \[([^\]]*)\]`) {
		arten[v] = "ausweis"
	}
	for _, v := range listeHinter(t, quelle, `BUCH_VORSILBEN = \[([^\]]*)\]`) {
		arten[v] = "buch"
	}
	// Das Gerät steht als einzelner Vergleich da, nicht als Liste.
	for _, m := range regexp.MustCompile(`scan\.startsWith\('([^']+)'\)`).FindAllStringSubmatch(quelle, -1) {
		arten[m[1]] = "geraet"
	}
	return arten
}

// vorsilbenAusOmniboxSwitch liest denselben Entscheid aus dem Server-Switch: je Zweig
// alle Vorsilben und den Handler, an den er weitergibt.
func vorsilbenAusOmniboxSwitch(t *testing.T) map[string]string {
	t.Helper()
	roh, err := os.ReadFile("omnibox_service.go")
	if err != nil {
		t.Fatalf("omnibox_service.go nicht lesbar: %v", err)
	}
	quelle := string(roh)
	start := strings.Index(quelle, "func (s *defaultOmniboxService) ProcessQuery")
	if start < 0 {
		t.Fatal("ProcessQuery nicht gefunden — der Detektor misst nichts")
	}
	ende := strings.Index(quelle[start:], "\n}\n")
	if ende < 0 {
		t.Fatal("Ende von ProcessQuery nicht gefunden")
	}
	rumpf := quelle[start : start+ende]

	handlerZuArt := map[string]string{
		"handleAusweisAction":     "ausweis",
		"handleBookAction":        "buch",
		"HandleDeviceAction":      "geraet",
		"handleLeserIDAction":     "", // keine Scanner-Vorsilbe, sondern die Trefferauswahl
		"resolveOhnePraefix":      "",
		"mapDeviceResult":         "geraet",
		"strings.TrimPrefix(q.Qu": "",
	}

	arten := map[string]string{}
	praefix := regexp.MustCompile(`strings\.HasPrefix\(q\.Query, "([^"]+)"\)`)
	for _, zweig := range strings.Split(rumpf, "\tcase ")[1:] {
		art := ""
		for handler, a := range handlerZuArt {
			if strings.Contains(zweig, handler) && a != "" {
				art = a
				break
			}
		}
		if art == "" {
			continue
		}
		for _, m := range praefix.FindAllStringSubmatch(zweig, -1) {
			arten[m[1]] = art
		}
	}
	return arten
}
