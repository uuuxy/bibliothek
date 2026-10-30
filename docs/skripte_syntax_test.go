package docs

import (
	"bufio"
	"errors"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// Jedes Shell-Skript des Repositorys lässt sich lesen: bash -n (sh -n bei #!/bin/sh) prüft die
// Syntax, ohne etwas auszuführen.
//
// Ein Syntaxfehler fällt sonst erst auf, wenn das Skript läuft: im Hook vor dem Push bei jedem
// Push, auf dem Server beim Update. Ein gerades Anführungszeichen, das ein deutsches „…"
// schließt, beendet in bash die Zeichenkette.
//
// Sieht nicht: Fehler, die erst beim Ausführen auffallen (ein fehlendes Programm, ein falscher
// Pfad), und Skripte ohne Kennzeile, deren Name nicht auf .sh endet.
func TestSkripte_HabenKeinenSyntaxfehler(t *testing.T) {
	var geprueft []string
	for _, datei := range projektDateien(t) {
		shell := shellVon(t, datei)
		if shell == "" {
			continue
		}
		geprueft = append(geprueft, datei)
		ausgabe, err := exec.Command(shell, "-n", filepath.Join("..", datei)).CombinedOutput() //nolint:gosec // Dateien aus dem Repo
		if err != nil {
			t.Errorf("%s -n %s: %v\n%s", shell, datei, err, ausgabe)
		}
	}
	// Nicht-leer-Garantie: Ein Sammler, der die Skripte nicht mehr findet, wäre still grün.
	if len(geprueft) < 15 || !slices.Contains(geprueft, "scripts/git-hooks/pre-push") {
		t.Fatalf("nur %d Skripte geprüft (%v) — der Sammler greift ins Leere", len(geprueft), geprueft)
	}
}

// shellVon nennt die Shell, mit der die Datei geprüft wird, oder "", wenn sie kein Shell-Skript
// ist. Gelesen wird nur die erste Zeile von Dateien auf .sh oder ohne Endung (die Hooks).
func shellVon(t *testing.T, datei string) string {
	t.Helper()
	endung := path.Ext(datei)
	if endung != ".sh" && endung != "" {
		return ""
	}
	f, err := os.Open(filepath.Join("..", datei)) //nolint:gosec // Dateien aus dem Repo
	if err != nil {
		t.Fatalf("%s lesen: %v", datei, err)
	}
	defer f.Close() //nolint:errcheck // nur gelesen
	erste, err := bufio.NewReader(f).ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		t.Fatalf("%s lesen: %v", datei, err)
	}
	erste = strings.TrimSpace(erste)
	switch {
	case strings.HasPrefix(erste, "#!") && strings.Contains(erste, "bash"):
		return "bash"
	case strings.HasPrefix(erste, "#!") && (strings.HasSuffix(erste, "/sh") || strings.HasSuffix(erste, " sh")):
		return "sh"
	case strings.HasPrefix(erste, "#!"):
		return ""
	case endung == ".sh":
		return "bash"
	}
	return ""
}
