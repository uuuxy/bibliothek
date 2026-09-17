package docs

import (
	"encoding/json"
	"os"
	"regexp"
	"testing"
	"time"
)

// Die Form der Ausnahmeliste — geprüft ohne Netz und ohne govulncheck.
//
// scripts/govulncheck-gate.sh prüft dasselbe zur Laufzeit, aber erst, wenn jemand pusht
// und govulncheck etwas meldet. Dieser Test hier läuft in jeder Suite und fängt den
// Tippfehler beim Eintragen.
//
// Die Prüfung auf „datei.go:" ist kein Schönheitsfehler: actions/setup-go registriert in
// der CI einen Problem-Matcher für Go. Der hält JEDE Logzeile der Form „datei.go: Text"
// für eine Compiler-Meldung und hängt eine rote Fehler-Markierung an einen grünen Job.
// Genau das ist am 17.09.2026 passiert, und ein roter Haken an einem grünen Lauf ist
// schlimmer als keiner: Beim nächsten Mal sieht niemand mehr hin.
type vulnAusnahme struct {
	ID            string   `json:"id"`
	Grund         string   `json:"grund"`
	Nachweis      []string `json:"nachweis"`
	GeprueftAm    string   `json:"geprueft_am"`
	Wiedervorlage string   `json:"wiedervorlage"`
}

var musterGoMeldung = regexp.MustCompile(`\.go:\s`)
var musterOSVID = regexp.MustCompile(`^GO-\d{4}-\d+$`)

func TestVulnAusnahmen_Form(t *testing.T) {
	roh, err := os.ReadFile("../security/vuln-ausnahmen.json")
	if err != nil {
		t.Fatalf("Ausnahmeliste lesen: %v", err)
	}
	var datei struct {
		Ausnahmen []vulnAusnahme `json:"ausnahmen"`
	}
	if err := json.Unmarshal(roh, &datei); err != nil {
		t.Fatalf("Ausnahmeliste ist kein gültiges JSON: %v", err)
	}

	for _, a := range datei.Ausnahmen {
		if !musterOSVID.MatchString(a.ID) {
			t.Errorf("Ausnahme mit unbrauchbarer Kennung %q — erwartet wird die Form GO-JJJJ-NNNN.", a.ID)
		}
		if a.Grund == "" {
			t.Errorf("%s: Ausnahme ohne Grund. Eine Ausnahme ohne Begründung ist ein Loch mit Namen.", a.ID)
		}
		if len(a.Nachweis) == 0 {
			t.Errorf("%s: Ausnahme ohne Nachweis. Der Nachweis muss ein Test sein, der rot wird, "+
				"wenn die Annahme fällt — keine Einschätzung.", a.ID)
		}
		for _, feld := range []struct{ name, wert string }{
			{"geprueft_am", a.GeprueftAm}, {"wiedervorlage", a.Wiedervorlage},
		} {
			if _, err := time.Parse("2006-01-02", feld.wert); err != nil {
				t.Errorf("%s: %s = %q ist kein Datum der Form JJJJ-MM-TT.", a.ID, feld.name, feld.wert)
			}
		}
		for _, n := range a.Nachweis {
			if musterGoMeldung.MatchString(n) {
				t.Errorf("%s: Der Nachweis %q sieht aus wie eine Go-Compiler-Meldung "+
					"(\"datei.go: Text\"). Der Problem-Matcher von actions/setup-go macht daraus "+
					"eine rote Markierung an einem grünen CI-Job. Bitte als \"Testname in pfad.go\" "+
					"schreiben.", a.ID, n)
			}
		}
	}
	if len(datei.Ausnahmen) == 0 {
		t.Log("keine Ausnahmen eingetragen — das ist der beste Zustand.")
	}
}
