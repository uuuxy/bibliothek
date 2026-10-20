package docs

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"
)

// Gate: Ein Kommentar in Skripten, Workflows und Konfiguration nennt kein Datum (CLAUDE.md,
// Abschnitt 4), wie im Go-Code (stil_ratsche_test.go) und in der Oberfläche
// (frontend-hygiene-vorgeschichte.test.js). Ein Kommentar sagt, warum die Zeile so ist; wann
// sie so wurde und was vorher dastand, steht in der Commit-Nachricht. Das Datum einer Quelle
// zählt nicht, in derselben Form wie dort.
//
// Gelesen werden die Dateien des Repositorys mit einer bekannten Form des Kommentars, je Zeile,
// die mit ihr beginnt. Nicht gelesen werden Go und Markdown (eigene Prüfungen), frontend/src
// (die Prüfung der Oberfläche), Tests (ein Datum ist dort meist ein Testwert), die Migrationen
// (jede ist die Aufzeichnung einer Änderung und wird nach ihrem Lauf nicht mehr bearbeitet) und
// die eingefrorenen Kopien unter db/testdata.
//
// Sieht nicht: ein Datum ohne Jahr, Vorgeschichte ohne Datum („bis dahin", „vorher"), einen
// Kommentar hinter Code auf derselben Zeile und eine Datei ohne bekannte Form. Ob ein Kommentar
// das Warum nennt, liest ein Mensch.
func TestKommentare_NennenKeinDatum(t *testing.T) {
	gelesen := map[string]int{}
	var funde []string
	for _, datei := range projektDateien(t) {
		roh, err := os.ReadFile(filepath.Join("..", datei)) //nolint:gosec // Dateien aus dem Repo
		if err != nil {
			t.Fatalf("%s lesen: %v", datei, err)
		}
		zeilen := strings.Split(string(roh), "\n")
		formen := kommentarFormen(datei, zeilen[0])
		for i, zeile := range zeilen {
			form := kommentarForm(zeile, formen)
			if form == "" {
				continue
			}
			gelesen[form]++
			if vorgeschichteDatum.MatchString(vorgeschichteQuelle.ReplaceAllString(zeile, "")) {
				funde = append(funde, fmt.Sprintf("%s:%d: %s", datei, i+1, strings.TrimSpace(zeile)))
			}
		}
	}
	// Gelesen je Form: Ein Sammler, der eine Form verliert, fände dort kein Datum mehr und bliebe
	// grün.
	if gelesen["#"] < 1000 || gelesen["--"] < 500 || gelesen["//"]+gelesen["*"] < 50 {
		t.Fatalf("nur %v Kommentarzeilen gelesen — der Sammler greift ins Leere", gelesen)
	}
	for _, fund := range funde {
		t.Errorf("Datum in einem Kommentar: %s. Der Kommentar nennt das Warum, die Vorgeschichte "+
			"steht in der Commit-Nachricht (CLAUDE.md, Abschnitt 4).", fund)
	}
}

// kommentarFormen nennt, womit eine Kommentarzeile der Datei beginnt; nil heißt, die Prüfung
// liest die Datei nicht.
func kommentarFormen(datei, ersteZeile string) []string {
	name := path.Base(datei)
	for _, aus := range []string{"frontend/src/", "frontend/e2e/", "frontend/e2e-proben/", "migrations/", "db/testdata/"} {
		if strings.HasPrefix(datei, aus) {
			return nil
		}
	}
	if strings.Contains(name, "_test.") || strings.HasSuffix(name, ".test.js") || strings.HasSuffix(name, ".spec.js") {
		return nil
	}
	switch path.Ext(name) {
	case ".go", ".md":
		return nil
	case ".sql":
		return []string{"--"}
	case ".js", ".mjs", ".cjs":
		return []string{"//", "/*", "*"}
	case ".sh", ".yml", ".yaml", ".properties", ".py", ".toml":
		return []string{"#"}
	}
	switch name {
	case "Dockerfile", ".env.example", ".gitignore", ".dockerignore", ".git-blame-ignore-revs":
		return []string{"#"}
	}
	if strings.HasPrefix(ersteZeile, "#!") {
		return []string{"#"}
	}
	return nil
}

// kommentarForm nennt die Form, mit der die Zeile als Kommentar beginnt, oder "". Die Zeile mit
// dem Interpreter (#!) ist kein Kommentar.
func kommentarForm(zeile string, formen []string) string {
	z := strings.TrimSpace(zeile)
	if strings.HasPrefix(z, "#!") {
		return ""
	}
	for _, form := range formen {
		if strings.HasPrefix(z, form) {
			return form
		}
	}
	return ""
}

func TestKommentare_Selbstprobe(t *testing.T) {
	formen := []struct {
		datei, ersteZeile string
		liest             bool
	}{
		{"update.sh", "#!/usr/bin/env bash", true},
		{"scripts/git-hooks/pre-push", "#!/bin/bash", true},
		{"Dockerfile", "# Stage 1", true},
		{".github/workflows/ci.yml", "name: CI", true},
		{"schema.sql", "-- Schema", true},
		{"frontend/vite.config.js", "import x from 'y';", true},
		{"migrations/164_x.sql", "-- Migration", false},
		{"frontend/src/lib/menue.js", "// Menü", false},
		{"frontend/e2e/theke.spec.js", "import", false},
		{"docs/ARCHITEKTUR.md", "# Titel", false},
		{"main.go", "package main", false},
		{"go.work", "go 1.27", false},
	}
	for _, f := range formen {
		if liest := kommentarFormen(f.datei, f.ersteZeile) != nil; liest != f.liest {
			t.Errorf("kommentarFormen(%q) liest = %v, erwartet %v", f.datei, liest, f.liest)
		}
	}
	zeilen := []struct {
		zeile, form string
	}{
		{"# Seit 19.08.2026 gefahrlos setzbar", "#"},
		{"    -- gemessen am 2026-09-22", "--"},
		{" * Anlass (24.08.2026): …", "*"},
		{"#!/bin/bash", ""},
		{`echo "# kein Kommentar"`, ""},
	}
	for _, z := range zeilen {
		if form := kommentarForm(z.zeile, []string{"#", "--", "//", "/*", "*"}); form != z.form {
			t.Errorf("kommentarForm(%q) = %q, erwartet %q", z.zeile, form, z.form)
		}
	}
}
