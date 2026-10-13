package docs

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Der Wegweiser (WEGWEISER.md) führt vom Menüpunkt zum Code. Er taugt nur, solange er
// vollständig ist: Ein neuer Menüpunkt oder ein neues Thema unter api/, das dort fehlt, lässt
// den Leser an der Stelle allein, an der er sucht.
//
// Geprüft wird: Jede Beschriftung aus dem Menü der Oberfläche (frontend/src/lib/menu.js) steht
// im Wegweiser, jeder Ordner unter api/ mit Programmcode ist dort genannt, und jeder Bildschirm,
// den die Tabelle der Menüpunkte nennt, liegt unter frontend/src/lib/. Die Dateien mit ganzem
// Pfad hält schon dateiverweise_test.go.
//
// Blindheit: ob eine Zeile den richtigen Ordner zum Menüpunkt nennt; eine Beschriftung, die das
// Menü nicht als label: '…' schreibt; die Tabelle „Ich will etwas ändern", soweit sie Ordner
// ohne Datei nennt.
func TestWegweiser_NenntJedenMenuepunktUndJedesThema(t *testing.T) {
	roh, err := os.ReadFile("WEGWEISER.md")
	if err != nil {
		t.Fatalf("WEGWEISER.md lesen: %v", err)
	}
	wegweiser := string(roh)

	menue, err := os.ReadFile(filepath.Join("..", "frontend", "src", "lib", "menu.js"))
	if err != nil {
		t.Fatalf("menu.js lesen: %v", err)
	}
	beschriftungen := regexp.MustCompile(`label: '([^']+)'`).FindAllStringSubmatch(string(menue), -1)
	if len(beschriftungen) < 10 {
		t.Fatalf("nur %d Beschriftungen in menu.js gefunden — das Muster greift nicht mehr", len(beschriftungen))
	}
	for _, b := range beschriftungen {
		if !strings.Contains(wegweiser, b[1]) {
			t.Errorf("Menüpunkt %q steht nicht in WEGWEISER.md, Abschnitt: Vom Menüpunkt zum Code", b[1])
		}
	}

	themen := 0
	ordner, err := os.ReadDir(filepath.Join("..", "api"))
	if err != nil {
		t.Fatalf("api/ lesen: %v", err)
	}
	for _, o := range ordner {
		if !o.IsDir() {
			continue
		}
		quellen, err := filepath.Glob(filepath.Join("..", "api", o.Name(), "*.go"))
		if err != nil {
			t.Fatalf("api/%s lesen: %v", o.Name(), err)
		}
		if len(quellen) == 0 {
			continue
		}
		themen++
		if !strings.Contains(wegweiser, "`api/"+o.Name()+"/") {
			t.Errorf("Ordner api/%s steht nicht in WEGWEISER.md", o.Name())
		}
	}
	if themen < 10 {
		t.Fatalf("nur %d Ordner mit Programmcode unter api/ gefunden — die Prüfung mäße an einem leeren Baum", themen)
	}

	bildschirme := regexp.MustCompile("`([A-Za-z/]+\\.svelte)`").FindAllStringSubmatch(wegweiser, -1)
	if len(bildschirme) < 15 {
		t.Fatalf("nur %d Bildschirme in WEGWEISER.md gefunden — das Muster greift nicht mehr", len(bildschirme))
	}
	for _, b := range bildschirme {
		pfad := strings.TrimPrefix(b[1], "frontend/src/lib/")
		if _, err := os.Stat(filepath.Join("..", "frontend", "src", "lib", filepath.FromSlash(pfad))); err != nil {
			t.Errorf("WEGWEISER.md nennt den Bildschirm %s, unter frontend/src/lib/ gibt es ihn nicht", b[1])
		}
	}
}
