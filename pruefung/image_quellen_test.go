package pruefung_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// Das Image bekommt jede Quelle, die seine Bauschritte brauchen, und keine, die es nicht gibt.
// Der Dockerfile kopiert die Ordner des Programms einzeln, internal/ und cmd/. Fehlt ein Ordner im Baum, bricht der
// Bau ab („"/inventur": not found"); fehlt einer in der Liste, findet go build sein Paket nicht.
// Beides merken go build ./..., die Tests und die Hooks nicht, weil sie im ganzen Baum bauen: Erst
// der Bau des Images in der CI scheitert, und aus main lässt sich kein Image bauen.
//
// Blindheit: Liest nur die Bau-Stufe des Backends und nur COPY ohne --from. Was ein Paket zur
// Laufzeit aus einem Ordner liest (Migrationen, Vorlagen), sieht sie nicht; das zeigt der Start
// des Containers.
func TestImageKopiertWasDerBauBraucht(t *testing.T) {
	wechsleInDieWurzel(t)
	roh, err := os.ReadFile("Dockerfile")
	if err != nil {
		t.Fatalf("Dockerfile lesen: %v", err)
	}
	stufe := bauStufeDesBackends(string(roh))
	if stufe == "" {
		t.Fatal("die Stufe „AS backend-builder“ fehlt im Dockerfile — der Leser greift nicht mehr")
	}

	kopiert := map[string]bool{}
	for _, m := range regexp.MustCompile(`(?m)^COPY ([^-\s]\S*) \./`).FindAllStringSubmatch(stufe, -1) {
		for _, quelle := range strings.Fields(m[1]) {
			kopiert[strings.TrimSuffix(quelle, "/")] = true
		}
	}
	ziele := regexp.MustCompile(`(?m)^RUN .*\bgo build\b.* (\./cmd/[a-z0-9-]+)\s*$`).FindAllStringSubmatch(stufe, -1)
	if len(kopiert) < 2 || len(ziele) < 4 {
		t.Fatalf("nur %d Quellen und %d Bauziele im Dockerfile gefunden — der Leser greift nicht mehr", len(kopiert), len(ziele))
	}

	for quelle := range kopiert {
		if _, err := os.Stat(quelle); err != nil {
			t.Errorf("Dockerfile kopiert %s, das es im Baum nicht gibt — der Bau des Images bricht ab", quelle)
		}
	}

	pakete := make([]string, 0, len(ziele))
	for _, z := range ziele {
		pakete = append(pakete, z[1])
	}
	ausgabe, err := exec.Command("go", append([]string{"list", "-deps", "-f", `{{if not .Standard}}{{.Dir}}{{end}}`}, pakete...)...).Output()
	if err != nil {
		t.Fatalf("go list -deps %v: %v", pakete, err)
	}
	ordnerDerPakete := strings.Fields(string(ausgabe))
	wurzel, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	gebraucht := map[string]bool{}
	for _, ordner := range ordnerDerPakete {
		rel, err := filepath.Rel(wurzel, ordner)
		if err != nil || strings.HasPrefix(rel, "..") {
			continue // ein Modul aus dem Zwischenspeicher, das go mod download holt
		}
		oben, _, _ := strings.Cut(filepath.ToSlash(rel), "/")
		gebraucht[oben] = true
	}
	if !gebraucht["internal"] || !gebraucht["cmd"] {
		t.Fatalf("internal/ und cmd/ fehlen in den Abhängigkeiten (%v) — der Leser greift nicht mehr", gebraucht)
	}
	var fehlen []string
	for oben := range gebraucht {
		if !kopiert[oben] {
			fehlen = append(fehlen, oben)
		}
	}
	sort.Strings(fehlen)
	for _, oben := range fehlen {
		t.Errorf("die Bauziele des Images binden Pakete aus %s ein, der Dockerfile kopiert den Ordner nicht", oben)
	}
}

// bauStufeDesBackends liefert den Text der Stufe „AS backend-builder“ bis zum nächsten FROM.
func bauStufeDesBackends(dockerfile string) string {
	anfang := regexp.MustCompile(`(?m)^FROM \S+ AS backend-builder\s*$`).FindStringIndex(dockerfile)
	if anfang == nil {
		return ""
	}
	rest := dockerfile[anfang[1]:]
	if ende := regexp.MustCompile(`(?m)^FROM `).FindStringIndex(rest); ende != nil {
		rest = rest[:ende[0]]
	}
	return rest
}
