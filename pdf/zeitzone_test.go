package pdf

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Ratsche gegen die Rückkehr der Container-Zeit in gedruckte Dokumente.
//
// Ein PDF aus diesem Paket ist ein Schreiben an einen Menschen: Schadensbescheid mit
// Zahlungsfrist, Rechnung, Kontoauszug. Sein Datum ist ein KALENDERTAG, und der hängt
// an der Zeitzone der Schule — nicht an der des Containers (im Image UTC). Zwischen 22
// und 24 Uhr UTC ist in Berlin bereits der Folgetag; ein `time.Now()` hier trüge dann
// den Vortag ins Schreiben und die 14-Tage-Frist einen Tag zu kurz.
//
// Gefunden am 23.08.2026 beim Raster-Durchgang (Frage 6, Zeit und Reihenfolge) an vier
// Stellen. Die Zeitzone selbst steht in pkg/schulzeit — eine Quelle, kein Nachbau.
// Erweitert am 03.09.2026: Bis dahin globte die Ratsche nur `*.go` im Paket pdf/ und war
// damit blind für JEDEN Erzeuger außerhalb — beim Rasterdurchgang lagen zwölf rohe
// time.Now() in sieben Dateien unter api/ und inventur/, darunter das „Stand"-Datum der
// neuen Schulbuch-Bestandsliste. Geprüft wird jetzt jede Quelldatei, die einen
// PDF-Baukasten einbindet; die Selbstprobe unten zählt sie.
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
		if bytes.Contains(roh, []byte("jung-kurt/gofpdf")) || bytes.Contains(roh, []byte("johnfercher/maroto")) {
			treffer = append(treffer, pfad)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("PDF-Erzeuger suchen: %v", err)
	}
	return treffer
}
