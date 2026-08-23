package pdf

import (
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
func TestKeinRohesTimeNowInGedrucktenDokumenten(t *testing.T) {
	dateien, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("glob: %v", err)
	}

	muster := regexp.MustCompile(`\btime\.Now\(\)`)
	geprueft := 0
	for _, d := range dateien {
		if strings.HasSuffix(d, "_test.go") {
			continue
		}
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
	if geprueft < 3 {
		t.Fatalf("nur %d Quelldateien gefunden — der Detektor greift ins Leere", geprueft)
	}
}
