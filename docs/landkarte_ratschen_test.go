package docs

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Regel 7 aus sweeps.md: Jede neue Ratsche trägt ihre Blindheit im Kopfkommentar und
// eine Zeile in der Landkarte. Bis zum 21.09.2026 hatte die Regel selbst keine Ratsche
// (OFFEN.md 5.10) — und zwölf Gates standen ohne Zeile da: Wer bei „alles grün" nach der
// Blindheit suchte, fand für sie keine.
//
// Geprüft wird die Anwesenheit des Dateinamens in sweeps.md, nicht der Inhalt der Zeile:
// Ob die genannte Blindheit stimmt, liest ein Mensch. Umfang: die Ratschen im
// Wurzelpaket, in docs/ und die Frontend-Hygiene-Tests; die Ratschen in api/, repository/
// usw. stehen von Hand in der Landkarte, ohne Gate.
func TestSweeps_JedeRatscheHatEineZeile(t *testing.T) {
	sweeps, err := os.ReadFile("sweeps.md")
	if err != nil {
		t.Fatalf("sweeps.md lesen: %v", err)
	}
	// Testdateien, die keine Ratsche sind — je mit dem Grund.
	keineRatsche := map[string]string{
		"main_test.go":               "Start des Servers, kein Gate",
		"cookie_secure_test.go":      "Einheitstest der Vorgabe COOKIE_SECURE",
		"landkarte_ratschen_test.go": "dieser Test",
	}

	var kandidaten []string
	for _, muster := range []string{
		"../*_test.go",
		"*_test.go",
		"../frontend/src/lib/frontend-hygiene*.test.js",
		"../frontend/src/lib/fehlerausgang.test.js",
		"../frontend/src/lib/e2e-hygiene-waechter.test.js",
	} {
		treffer, err := filepath.Glob(muster)
		if err != nil {
			t.Fatal(err)
		}
		kandidaten = append(kandidaten, treffer...)
	}
	// Nicht-leer-Garantie: Findet der Sammler kaum Dateien, prüft er nichts.
	if len(kandidaten) < 20 {
		t.Fatalf("nur %d Ratschen-Dateien gefunden — die Muster greifen nicht mehr", len(kandidaten))
	}

	gesehen := map[string]bool{}
	for _, pfad := range kandidaten {
		name := filepath.Base(pfad)
		if _, ok := keineRatsche[name]; ok {
			gesehen[name] = true
			continue
		}
		if !strings.Contains(string(sweeps), name) {
			t.Errorf("%s hat keine Zeile in docs/sweeps.md (Landkarte der Ratschen) — Regel 7: Blindheit im "+
				"Kopfkommentar UND eine Zeile dort. Ist die Datei keine Ratsche, mit Grund in keineRatsche eintragen.", pfad)
		}
	}
	for name := range keineRatsche {
		if !gesehen[name] {
			t.Errorf("keineRatsche nennt %s, die Datei gibt es nicht mehr — Eintrag austragen", name)
		}
	}
}
