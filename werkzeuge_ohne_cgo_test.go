package main

import (
	"os"
	"os/exec"
	"regexp"
	"strings"
	"testing"
)

// Der Dockerfile baut vier Werkzeuge mit CGO_ENABLED=0. Zieht eines davon über seine Importe
// ein Paket mit cgo mit (die WebP-Bibliothek hinter pkg/imageutil), scheitert erst der Bau
// des Images: `go build ./...`, die Tests und die Hooks laufen mit cgo und merken nichts.
// Der Weg dorthin ist kurz — ein neuer Import in repository genügt.
func TestWerkzeugeOhneCgo_BindenKeinCgoPaketEin(t *testing.T) {
	dockerfile, err := os.ReadFile("Dockerfile")
	if err != nil {
		t.Fatalf("Dockerfile lesen: %v", err)
	}
	ohneCgo := regexp.MustCompile(`(?m)^RUN CGO_ENABLED=0 .*\bgo build\b.* (\./cmd/[a-z0-9-]+)\s*$`).
		FindAllStringSubmatch(string(dockerfile), -1)
	if len(ohneCgo) < 4 {
		t.Fatalf("nur %d Zeilen „CGO_ENABLED=0 … go build … ./cmd/…\" im Dockerfile gefunden — der Leser greift nicht mehr", len(ohneCgo))
	}

	// Gegenprobe: Den Server selbst baut der Dockerfile mit cgo, und der Detektor sieht es.
	if cgo := cgoPakete(t, "."); len(cgo) == 0 {
		t.Fatal("der Server bindet kein cgo-Paket ein — der Detektor sieht nichts, oder die Prüfung ist überholt")
	}
	for _, zeile := range ohneCgo {
		if cgo := cgoPakete(t, zeile[1]); len(cgo) > 0 {
			t.Errorf("%s wird ohne cgo gebaut, bindet aber %v ein — der Bau des Images scheitert", zeile[1], cgo)
		}
	}
}

// cgoPakete nennt die Abhängigkeiten eines Pakets außerhalb der Standardbibliothek, die
// C einbinden. Die Standardbibliothek hat für ihre cgo-Dateien einen Weg ohne.
func cgoPakete(t *testing.T, paket string) []string {
	t.Helper()
	ausgabe, err := exec.Command("go", "list", "-deps", "-f",
		`{{if and (not .Standard) .CgoFiles}}{{.ImportPath}}{{end}}`, paket).Output()
	if err != nil {
		t.Fatalf("go list %s: %v", paket, err)
	}
	return strings.Fields(string(ausgabe))
}
