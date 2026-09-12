package api

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"testing"
)

// Der Rahmen eines neuen Plans steht in Go und im Planer — mit denselben Zahlen (#593).
//
// Der Server schlägt den Rahmen vor (2. Stunde Beginn, 4. Stunde Ende, 6 Stunden je Tag —
// die 1. gehört der Klassenleitung). Der Planer hält denselben Rahmen als Zustand VOR dem
// ersten Laden: `leererEntwurf()`. Dort stand die Startstunde auf 1 — verdeckt, solange
// der Vorschlag des Servers eintrifft und gewinnt, und sichtbar in dem Moment, in dem er
// ausbleibt.
//
// Zwei Zahlen für dieselbe Vorgabe brauchen deshalb ein Gate, nicht nur einen Fix: Ohne
// es zieht der nächste Wechsel der Schulorganisation eine Seite nach und die andere nicht.
func TestLmfPlanVorgabeIstInGoUndImPlanerDieselbe(t *testing.T) {
	pfad := filepath.Join("..", "frontend", "src", "lib", "lmfplanDienst.js")
	roh, err := os.ReadFile(pfad)
	if err != nil {
		t.Fatalf("%s lesen: %v", pfad, err)
	}
	quelle := ohneKommentare(string(roh))

	// Der leere Entwurf steht als Objektliteral in leererEntwurf(); gelesen werden die
	// drei Zahlen, die auch der Server vorschlägt.
	block := regexp.MustCompile(`(?s)export function leererEntwurf\(\)\s*\{.*?\n\}`).FindString(quelle)
	if block == "" {
		t.Fatal("leererEntwurf() nicht gefunden — Parser oder lmfplanDienst.js prüfen; " +
			"ein Gate, das nichts findet, meldet ewig „alles gut“")
	}

	zahl := func(feld string) int {
		t.Helper()
		m := regexp.MustCompile(feld + `:\s*(\d+)`).FindStringSubmatch(block)
		if m == nil {
			t.Fatalf("%s steht nicht im leeren Entwurf: %s", feld, block)
		}
		n, err := strconv.Atoi(m[1])
		if err != nil {
			t.Fatalf("%s ist keine Zahl: %v", feld, err)
		}
		return n
	}

	faelle := []struct {
		feld     string
		gefunden int
		erwartet int
	}{
		{"startstunde", zahl("startstunde"), lmfPlanStartstundeVorgabe},
		{"letzte_stunde", zahl("letzte_stunde"), lmfPlanLetzteStundeVorgabe},
		{"stunden_je_tag", zahl("stunden_je_tag"), lmfPlanStundenJeTagVorgabe},
	}
	for _, f := range faelle {
		if f.gefunden != f.erwartet {
			t.Errorf("Der Planer beginnt mit %s = %d, der Server schlägt %d vor — "+
				"bleibt der Vorschlag aus, steht im Formular eine andere Zahl als die Schule vereinbart hat.",
				f.feld, f.gefunden, f.erwartet)
		}
	}
}
