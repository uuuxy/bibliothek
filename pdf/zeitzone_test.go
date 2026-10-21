package pdf

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Ratsche gegen die Zeit des Containers in gedruckten Dokumenten.
//
// Ein PDF ist ein Schreiben an einen Menschen: Bescheid mit Zahlungsfrist, Rechnung,
// Kontoauszug. Sein Datum ist ein Kalendertag, und der hängt an der Zeitzone der Schule, nicht an
// der des Containers (im Image UTC). Zwischen 22 und 24 Uhr UTC ist in Berlin schon der
// Folgetag; ein `time.Now()` trüge dann den Vortag ins Schreiben und die Frist von 14 Tagen einen
// Tag zu kurz. Die Zeitzone steht in pkg/schulzeit, eine Quelle, kein Nachbau.
//
// Geprüft wird jede Quelldatei, die einen Baukasten für PDFs einbindet, nicht nur das Paket pdf/:
// Erzeuger stehen auch unter api/ und inventur/. Die Selbstprobe unten zählt sie.
func TestKeinRohesTimeNowInGedrucktenDokumenten(t *testing.T) {
	dateien := pdfErzeuger(t)

	muster := regexp.MustCompile(`\btime\.Now\(\)`)
	geprueft := 0
	for _, d := range dateien {
		roh, err := os.ReadFile(d)
		if err != nil {
			t.Fatalf("%s lesen: %v", d, err)
		}
		geprueft++
		for i, zeile := range strings.Split(string(roh), "\n") {
			nackt := strings.TrimSpace(zeile)
			if strings.HasPrefix(nackt, "//") {
				continue
			}
			if muster.MatchString(zeile) {
				t.Errorf("%s:%d benutzt time.Now() — in einem gedruckten Dokument gehört "+
					"schulzeit.Jetzt() hin (Container läuft in UTC):\n    %s", d, i+1, nackt)
			}
		}
	}
	if geprueft < 8 {
		t.Fatalf("nur %d PDF-Erzeuger gefunden — der Detektor greift ins Leere. Beim Bau "+
			"der Regel waren es 16 in pdf/, api/ und inventur/.", geprueft)
	}
}

// pdfErzeuger sammelt alle Quelldateien des Projekts, die gofpdf oder maroto einbinden —
// also alles, was ein Dokument zum Ausdrucken erzeugt. Nach dem Verzeichnis zu suchen
// wäre falsch: Die Erzeuger sitzen in pdf/, api/ und inventur/.
func pdfErzeuger(t *testing.T) []string {
	t.Helper()
	wurzel := filepath.Join("..")
	var treffer []string
	err := filepath.WalkDir(wurzel, func(pfad string, eintrag os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if eintrag.IsDir() {
			if name := eintrag.Name(); name == "node_modules" || name == ".git" || name == "frontend" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(pfad, ".go") || strings.HasSuffix(pfad, "_test.go") {
			return nil
		}
		roh, err := os.ReadFile(pfad)
		if err != nil {
			return err
		}
		if bytes.Contains(roh, []byte("phpdave11/gofpdf")) || bytes.Contains(roh, []byte("johnfercher/maroto")) {
			treffer = append(treffer, pfad)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("PDF-Erzeuger suchen: %v", err)
	}
	return treffer
}
