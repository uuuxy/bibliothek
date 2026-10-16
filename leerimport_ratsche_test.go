package main

import (
	"bytes"
	"encoding/base64"
	"go/parser"
	"go/token"
	"image"
	"sort"
	"strings"
	"testing"

	"github.com/swaggo/swag"
)

// Ein Paket, das nur für seine Nebenwirkung eingebunden wird (import _), gehört in ein Programm
// oder in einen Test. So verlangt es Googles Regelwerk für Go (Decisions, „Import blank"):
// „Avoid blank imports in library packages, even if the library indirectly depends on them."
// Das Programm entscheidet dann, welche Bildformate es liest, und ein Test kann andere wählen.
//
// Ausgenommen ist embed, wie bei Google: Eine Datei mit //go:embed braucht den Import.
// Blindheit: Eine Nebenwirkung hinter einem Import mit Namen sieht die Prüfung nicht
// (chai2010/webp meldet in pkg/imageutil auch seinen Decoder an), Testdateien liest sie nicht.
func TestLeerimporte_NurInMainUndTests(t *testing.T) {
	quellen := leseGoQuellen(t, ".")
	fset := token.NewFileSet()
	var funde []string
	programme := 0
	for pfad, quelle := range quellen {
		if strings.HasSuffix(pfad, "_test.go") {
			continue
		}
		datei, err := parser.ParseFile(fset, pfad, quelle, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("%s lesen: %v", pfad, err)
		}
		if datei.Name.Name == "main" {
			programme++
			continue
		}
		for _, imp := range datei.Imports {
			ziel := strings.Trim(imp.Path.Value, `"`)
			if imp.Name != nil && imp.Name.Name == "_" && ziel != "embed" {
				funde = append(funde, fset.Position(imp.Pos()).String()+": "+ziel)
			}
		}
	}

	// Gegenprobe am Detektor: Ohne die Programme unter cmd/ liest er den falschen Baum.
	if programme < 5 {
		t.Fatalf("nur %d Dateien in package main gefunden: Der Detektor liest den falschen Baum", programme)
	}
	sort.Strings(funde)
	for _, fund := range funde {
		t.Errorf("%s ist ein Leerimport in einer Bibliothek. Ihn in main.go oder in den Test setzen, der ihn braucht.", fund)
	}
}

// Die Bibliotheken dekodieren mit image.Decode und finden nur die Formate, deren Decoder das
// Programm einbindet. Fehlt einer, scheitert erst das Hochladen oder Abrufen eines Covers mit
// „image: unknown format". Die Proben sind fertige Bytes, weil ein Encoder im Test seinen
// Decoder selbst anmeldete.
//
// Blindheit: GIF, JPEG und PNG melden auch die PDF-Bibliotheken an (gofpdf, pdfcpu, maroto),
// WebP auch pkg/imageutil. Rot wird die Prüfung erst, wenn keiner das Format mehr einbindet.
func TestServer_LiestDieBildformate(t *testing.T) {
	proben := map[string]string{
		"gif":  "R0lGODlhAQABAIAAAAAAAP///ywAAAAAAQABAAACAkQBADs=",
		"jpeg": "/9j/2wCEAAgGBgcGBQgHBwcJCQgKDBQNDAsLDBkSEw8UHRofHh0aHBwgJC4nICIsIxwcKDcpLDAxNDQ0Hyc5PTgyPC4zNDIBCQkJDAsMGA0NGDIhHCEyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMv/AAAsIAAEAAQEBEQD/xADSAAABBQEBAQEBAQAAAAAAAAAAAQIDBAUGBwgJCgsQAAIBAwMCBAMFBQQEAAABfQECAwAEEQUSITFBBhNRYQcicRQygZGhCCNCscEVUtHwJDNicoIJChYXGBkaJSYnKCkqNDU2Nzg5OkNERUZHSElKU1RVVldYWVpjZGVmZ2hpanN0dXZ3eHl6g4SFhoeIiYqSk5SVlpeYmZqio6Slpqeoqaqys7S1tre4ubrCw8TFxsfIycrS09TV1tfY2drh4uPk5ebn6Onq8fLz9PX29/j5+v/aAAgBAQAAPwD5/r//2Q==",
		"png":  "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAIAAACQd1PeAAAAEUlEQVR4nAAEAPv/AshkMgMAA14BYWV63BUAAAAASUVORK5CYII=",
		"webp": "UklGRh4AAABXRUJQVlA4TBEAAAAvAAAAAAdQsiJXpv+BiOh/AAA=",
	}
	for format, probe := range proben {
		daten, err := base64.StdEncoding.DecodeString(probe)
		if err != nil {
			t.Fatalf("Probe %s entpacken: %v", format, err)
		}
		bild, gelesen, err := image.Decode(bytes.NewReader(daten))
		if err != nil {
			t.Errorf("image.Decode(%s-Probe) = %v, erwartet ein Bild: Kein Paket des Servers bindet den Decoder ein, er gehört in main.go", format, err)
			continue
		}
		if gelesen != format || bild.Bounds().Dx() != 1 {
			t.Errorf("image.Decode(%s-Probe) = Format %q mit %d Pixel Breite, erwartet %q mit 1", format, gelesen, bild.Bounds().Dx(), format)
		}
	}
}

// Die Swagger-Seite (nur lokal und in der Entwicklung) liefert die Beschreibung, die
// bibliothek/docs beim Start anmeldet; ohne den Import in main.go bliebe sie leer.
func TestServer_KenntDieAPIBeschreibung(t *testing.T) {
	beschreibung, err := swag.ReadDoc()
	if err != nil {
		t.Fatalf("swag.ReadDoc() = %v, erwartet die Beschreibung: main.go bindet bibliothek/docs nicht ein", err)
	}
	if !strings.Contains(beschreibung, `"paths"`) {
		t.Errorf("swag.ReadDoc() liefert %d Zeichen ohne \"paths\", erwartet die Beschreibung der Adressen", len(beschreibung))
	}
}
