package main

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Von außerhalb des Schulnetzes lässt Caddy nur die Bestellseite des Lieferanten durch
// (update_caddy.sh, Abschnitt „Bibliothek"; ARCHITEKTUR.md A29). Die Seite ist dieselbe index.html
// wie das Programm: Jede Datei, die sie vom Wurzelpfad lädt, muss in der Liste des Matchers
// @draussen stehen, sonst antwortet Caddy dem Lieferanten mit 403. So fehlte /darstellung.js, als
// das dunkle Design dazukam.
//
// Blindheit: Liest index.html und frontend/public/. Was der Bau hinzufügt (manifest.webmanifest,
// sw.js) und was die Seite erst zur Laufzeit anfragt, sieht sie nicht; das zeigt die Probe von
// außen in DEPLOYMENT.md, Abschnitt 3. Die Muster liest sie wie Caddy, genau oder mit * am Ende als
// Präfix; ein Muster anderer Form lässt sie scheitern, statt es falsch zu lesen.
func TestCaddyLaesstDieDateienDerBestellseiteDurch(t *testing.T) {
	roh, err := os.ReadFile("update_caddy.sh")
	if err != nil {
		t.Fatalf("update_caddy.sh lesen: %v", err)
	}
	muster := caddyMusterVonAussen(string(roh))
	if len(muster) < 5 {
		t.Fatalf("nur %d Muster in @draussen gefunden — der Leser greift nicht mehr", len(muster))
	}

	var pfade []string
	err = filepath.WalkDir(filepath.Join("frontend", "public"), func(pfad string, eintrag fs.DirEntry, err error) error {
		if err != nil || eintrag.IsDir() {
			return err
		}
		rel, err := filepath.Rel(filepath.Join("frontend", "public"), pfad)
		pfade = append(pfade, "/"+filepath.ToSlash(rel))
		return err
	})
	if err != nil {
		t.Fatalf("frontend/public lesen: %v", err)
	}
	seite, err := os.ReadFile(filepath.Join("frontend", "index.html"))
	if err != nil {
		t.Fatalf("frontend/index.html lesen: %v", err)
	}
	// Was unter /src/ steht, bündelt der Bau nach /assets/.
	for _, m := range regexp.MustCompile(`(?:src|href)="(/[^"]*)"`).FindAllStringSubmatch(string(seite), -1) {
		if !strings.HasPrefix(m[1], "/src/") {
			pfade = append(pfade, m[1])
		}
	}
	if len(pfade) < 2 {
		t.Fatalf("nur %d Dateien der Seite gefunden — der Leser greift nicht mehr", len(pfade))
	}

	geprueft := map[string]bool{}
	for _, pfad := range pfade {
		if geprueft[pfad] {
			continue
		}
		geprueft[pfad] = true
		gedeckt, fehler := caddyDecktPfad(muster, pfad)
		if fehler != "" {
			t.Fatal(fehler)
		}
		if !gedeckt {
			t.Errorf("%s fehlt in „not path“ des Matchers @draussen (update_caddy.sh): Von außen bekäme die Bestellseite die Datei mit 403", pfad)
		}
	}
}

// caddyMusterVonAussen liest die Pfade aus der Zeile „not path …" im Matcher @draussen.
func caddyMusterVonAussen(caddyfile string) []string {
	block := regexp.MustCompile(`(?s)@draussen \{(.*?)\n\s*\}`).FindStringSubmatch(caddyfile)
	if block == nil {
		return nil
	}
	zeile := regexp.MustCompile(`(?m)^\s*not path (.+)$`).FindStringSubmatch(block[1])
	if zeile == nil {
		return nil
	}
	return strings.Fields(zeile[1])
}

// caddyDecktPfad sagt, ob eines der Muster den Pfad trifft. fehler nennt ein Muster, das dieser
// Leser nicht kennt.
func caddyDecktPfad(muster []string, pfad string) (gedeckt bool, fehler string) {
	for _, m := range muster {
		switch {
		case strings.Count(m, "*") == 0:
			gedeckt = gedeckt || m == pfad
		case strings.Count(m, "*") == 1 && strings.HasSuffix(m, "*"):
			gedeckt = gedeckt || strings.HasPrefix(pfad, strings.TrimSuffix(m, "*"))
		default:
			return false, "Muster " + m + " in @draussen hat eine Form, die dieser Test nicht liest"
		}
	}
	return gedeckt, ""
}

func TestCaddyMusterDetektorErkenntDieFormen(t *testing.T) {
	muster := caddyMusterVonAussen("x {\n    @draussen {\n        not remote_ip 10.0.0.0/8\n        not path /a/* /b.js /health\n    }\n}")
	if strings.Join(muster, " ") != "/a/* /b.js /health" {
		t.Fatalf("Muster %q gelesen, erwartet /a/* /b.js /health", muster)
	}
	faelle := map[string]bool{"/a/x.js": true, "/b.js": true, "/b.jsx": false, "/c.js": false, "/health": true}
	for pfad, erwartet := range faelle {
		if gedeckt, fehler := caddyDecktPfad(muster, pfad); gedeckt != erwartet || fehler != "" {
			t.Errorf("%s: gedeckt=%v (%s), erwartet %v", pfad, gedeckt, fehler, erwartet)
		}
	}
	if _, fehler := caddyDecktPfad([]string{"/a/*.js"}, "/a/x.js"); fehler == "" {
		t.Error("ein Muster mit * in der Mitte muss den Test scheitern lassen")
	}
}
