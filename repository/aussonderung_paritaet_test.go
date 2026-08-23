package repository

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Aussonderungs-Paritäts-Gate: Wer ein Exemplar aussondert, muss es auch unausleihbar
// machen.
//
// `ist_ausgesondert` und `ist_ausleihbar` sind zwei Spalten für einen Zustand. Drei der
// vier Aussonderungswege setzten immer beide; `syncBookStock` (inventur) setzte bis zum
// 23.08.2026 nur `ist_ausgesondert = true` und ließ das Exemplar "ausleihbar" zurück.
// Schaden richtete das nicht an — aber nur, weil jeder heutige Leser BEIDE Spalten
// prüft. Genau diese Art Übereinstimmung hält nichts: Der erste Leser, der nur
// `ist_ausleihbar` abfragt, verleiht ein ausgesondertes Exemplar, und niemand sieht es
// kommen.
//
// Das Gate liest die SQL-Anweisungen lexikalisch. Die Gegenrichtung
// (`ist_ausgesondert = false` beim Zurückholen eines Fundes) ist ausdrücklich nicht
// gemeint — dort gehört `ist_ausleihbar = true` dazu, und das prüft der zweite Fall.
func TestAussonderungSetztBeideSpalten(t *testing.T) {
	// Anweisungen, die buecher_exemplare anfassen — grob am UPDATE-Schlüsselwort
	// aufgetrennt, damit jede Anweisung für sich beurteilt wird.
	anweisung := regexp.MustCompile(`(?is)UPDATE\s+buecher_exemplare\s+SET\s+(.*?)(?:WHERE|RETURNING|` + "`" + `)`)

	geprueft := 0
	for _, pfad := range goDateien(t) {
		roh, err := os.ReadFile(pfad)
		if err != nil {
			t.Fatalf("%s lesen: %v", pfad, err)
		}
		for _, m := range anweisung.FindAllStringSubmatch(string(roh), -1) {
			satz := m[1]
			switch {
			case strings.Contains(satz, "ist_ausgesondert = true"):
				geprueft++
				if !strings.Contains(satz, "ist_ausleihbar = false") {
					t.Errorf("%s: sondert aus (ist_ausgesondert = true), setzt aber ist_ausleihbar nicht auf false:\n    %s",
						pfad, einzeilig(satz))
				}
			case strings.Contains(satz, "ist_ausgesondert = false"):
				geprueft++
				if !strings.Contains(satz, "ist_ausleihbar = true") {
					t.Errorf("%s: holt ein Exemplar zurück (ist_ausgesondert = false), macht es aber nicht wieder ausleihbar:\n    %s",
						pfad, einzeilig(satz))
				}
			}
		}
	}

	// Gegenprobe am Detektor: Findet er gar nichts, prüft er auch nichts.
	if geprueft < 4 {
		t.Fatalf("nur %d Aussonderungs-Anweisungen gefunden — der Detektor greift ins Leere", geprueft)
	}
}

// goDateien liefert alle Go-Quellen des Baums (ohne Tests) — die Aussonderungswege
// liegen in repository/ UND in inventur/.
func goDateien(t *testing.T) []string {
	t.Helper()
	var aus []string
	wurzel := filepath.Join("..")
	err := filepath.Walk(wurzel, func(pfad string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if info.Name() == "node_modules" || info.Name() == ".git" || info.Name() == "frontend" {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(pfad, ".go") && !strings.HasSuffix(pfad, "_test.go") {
			aus = append(aus, pfad)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Baum durchlaufen: %v", err)
	}
	return aus
}

func einzeilig(s string) string {
	return strings.Join(strings.Fields(s), " ")
}
