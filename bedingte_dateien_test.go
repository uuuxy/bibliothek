package main

import (
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// bedingungDesBaus findet die Zeile //go:build mit einem einzelnen Tag.
var bedingungDesBaus = regexp.MustCompile(`(?m)^//go:build (\w+)$`)

// Eine Datei mit einer Bedingung für den Bau übersetzen weder go build ./... noch go test ./...:
// Ein umbenannter Name bricht sie, und erst ihr nächster Lauf merkt es. Diese Prüfung übersetzt
// jede solche Datei mit go vet, ein Skript mit ignore (go run scripts/…) einzeln, eine Datei mit
// eigenem Tag in ihrem Paket mit diesem Tag.
//
// Blindheit: Eine Bedingung aus mehreren Teilen (linux && amd64) liest sie nicht.
func TestBedingteDateien_LassenSichUebersetzen(t *testing.T) {
	skripte, tags := bedingteDateien(t)

	// Gegenprobe am Detektor: Die zwei Skripte und das Tag raster muss er finden.
	if len(skripte) < 2 || !tags["raster"]["repository"] {
		t.Fatalf("%d Skripte und die Tags %v gefunden: Der Detektor liest den falschen Baum", len(skripte), tags)
	}
	for _, skript := range skripte {
		pruefeMitVet(t, "./"+skript)
	}
	for tag, ordner := range tags {
		for o := range ordner {
			pruefeMitVet(t, "-tags", tag, "./"+o)
		}
	}
}

// bedingteDateien liefert die Skripte mit //go:build ignore und je anderem Tag die Ordner, in
// denen es steht.
func bedingteDateien(t *testing.T) (skripte []string, tags map[string]map[string]bool) {
	t.Helper()
	tags = map[string]map[string]bool{}
	for pfad, quelle := range leseGoQuellen(t, ".") {
		treffer := bedingungDesBaus.FindStringSubmatch(quelle)
		switch {
		case treffer == nil:
		case treffer[1] == "ignore":
			skripte = append(skripte, pfad)
		default:
			if tags[treffer[1]] == nil {
				tags[treffer[1]] = map[string]bool{}
			}
			tags[treffer[1]][filepath.Dir(pfad)] = true
		}
	}
	sort.Strings(skripte)
	return skripte, tags
}

func pruefeMitVet(t *testing.T, argumente ...string) {
	t.Helper()
	ausgabe, err := exec.Command("go", append([]string{"vet"}, argumente...)...).CombinedOutput()
	if err != nil {
		t.Errorf("go vet %s = %v, erwartet ohne Fund:\n%s", strings.Join(argumente, " "), err, ausgabe)
	}
}
