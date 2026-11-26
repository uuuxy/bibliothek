package repository

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"testing"
)

// Jedes Zahlenfeld der Einstellungen hat in der Maske dieselben Grenzen wie in der Prüfung des
// Servers (zahlenFelder). Mit einer engeren Grenze ließ die Maske einen Wert nicht zu, den der
// Server annimmt (Karenzzeit 365 gegen 3650 Tage, Max. Ausleihen 50 gegen 100); ohne Grenze
// schickte sie einen Wert ab, den erst der Server mit 400 abweist. Das Feld findet der Test über
// sein Etikett, das in beiden Listen gleich lautet.
func TestEinstellungen_MaskeHatDieGrenzenDesServers(t *testing.T) {
	ordner := filepath.Join("..", "..", "frontend", "src", "lib", "components", "einstellungen", "kategorien")
	dateien, err := filepath.Glob(filepath.Join(ordner, "*.svelte"))
	if err != nil || len(dateien) == 0 {
		t.Fatalf("keine Masken unter %s (Fehler: %v)", ordner, err)
	}
	var masken string
	for _, datei := range dateien {
		inhalt, err := os.ReadFile(datei)
		if err != nil {
			t.Fatal(err)
		}
		masken += string(inhalt)
	}

	felder := zahlenFelder(&EinstellungenPatch{})
	for _, f := range felder {
		element := regexp.MustCompile(`<Feld\b[^>]*?etikett="` + regexp.QuoteMeta(f.beschriftung) + `"[^>]*?/>`).FindString(masken)
		if element == "" {
			t.Errorf("%s: kein Feld mit dem Etikett %q in den Masken", f.schluessel, f.beschriftung)
			continue
		}
		for _, grenze := range []struct {
			name string
			soll int
		}{{"min", f.min}, {"max", f.max}} {
			treffer := regexp.MustCompile(`\b` + grenze.name + `=(?:\{|")(\d+)(?:\}|")`).FindStringSubmatch(element)
			if treffer == nil {
				t.Errorf("%s: Feld %q ohne %s, der Server erlaubt %d", f.schluessel, f.beschriftung, grenze.name, grenze.soll)
				continue
			}
			ist, err := strconv.Atoi(treffer[1])
			if err != nil {
				t.Fatalf("%s: %s %q ist keine Zahl: %v", f.schluessel, grenze.name, treffer[1], err)
			}
			if ist != grenze.soll {
				t.Errorf("%s: Feld %q hat %s %d, der Server %d", f.schluessel, f.beschriftung, grenze.name, ist, grenze.soll)
			}
		}
	}
}
