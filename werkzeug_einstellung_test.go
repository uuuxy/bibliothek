package main

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

// Die versionierte Werkzeug-Einstellung .claude/settings.json erbt jeder, der das öffentliche
// Repository klont und mit demselben Werkzeug öffnet. Sie trägt deshalb nur den Hinweis-Hook
// und Freigaben, die für jeden gelten. Über Monate hatten sich dort rund hundert persönliche
// Freigaben gesammelt, darunter weit gefasste (docker exec, git config, npm run) und solche mit
// dem Heimatpfad des Entwicklers. Persönliche Freigaben gehören in die unversionierte
// .claude/settings.local.json.
func TestWerkzeugEinstellung_NurFreigabenFuerAlle(t *testing.T) {
	// fuerAlle sind die Freigaben, die jeder bekommen darf: Sie lesen oder prüfen nur, oder es
	// sind Skripte dieses Repositorys. Eine neue Zeile hier ist eine Entscheidung für alle.
	fuerAlle := map[string]bool{
		"Bash(go test *)":                  true,
		"Bash(./scripts/api_inventar.sh)":  true,
		"Bash(./scripts/deadcode_gate.sh)": true,
	}

	roh, err := os.ReadFile(".claude/settings.json")
	if err != nil {
		t.Fatalf(".claude/settings.json lesen: %v", err)
	}
	for _, pfad := range []string{"/Users/", "/home/"} {
		if strings.Contains(string(roh), pfad) {
			t.Errorf(".claude/settings.json nennt einen Heimatpfad (%s): Er verrät Benutzername und Ablageort und gilt bei niemand anderem", pfad)
		}
	}
	var einstellung struct {
		Permissions struct {
			Allow []string `json:"allow"`
		} `json:"permissions"`
		Hooks map[string]json.RawMessage `json:"hooks"`
	}
	if err := json.Unmarshal(roh, &einstellung); err != nil {
		t.Fatalf(".claude/settings.json ist kein gültiges JSON: %v", err)
	}
	for _, freigabe := range einstellung.Permissions.Allow {
		if !fuerAlle[freigabe] {
			t.Errorf("Freigabe %q steht in der versionierten .claude/settings.json. Eine persönliche "+
				"Freigabe gehört in .claude/settings.local.json; gilt sie für jeden, hier mit Grund eintragen.", freigabe)
		}
	}
	// Der Hook gehört zum Projekt (CLAUDE.md); ohne ihn wäre die Datei die falsche.
	if _, ok := einstellung.Hooks["PreToolUse"]; !ok {
		t.Error(".claude/settings.json trägt den Hinweis-Hook PreToolUse nicht mehr")
	}
}
