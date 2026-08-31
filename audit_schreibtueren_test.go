package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Append-only auf audit_log/audit_logs ist seit Migration 083 bewusst KEIN Trigger mehr
// (die DSGVO-Tilgung braucht die Schreibtür), sondern Konvention. Eine Konvention ohne
// Ratsche ist keine (Prüfung 22.08.2026): Dieser Test hält die Liste der Dateien fest, die
// audit_log(s) verändern oder löschen dürfen. Wer eine neue Tür baut, trägt sie hier ein —
// und begründet sie im Commit.
func TestAuditLogSchreibtueren_NurBekannteDateien(t *testing.T) {
	erlaubt := map[string]string{
		// Seit 31.08.2026 fährt auch der DSGVO-Cron (jobs/cron_dsgvo.go) diese Statements —
		// über repository.SpurTilgungen, die eine Liste; er hat keine eigene Tür mehr.
		"repository/audit_users.go":       "SpurTilgungen (DSGVO: Purge + LUSD-Anonymisierung + Cron)",
		"jobs/cron_dsgvo_lesehistorie.go": "tilgeAusleihProtokoll (Lesehistorie-Frist)",
		"jobs/cron_audit_retention.go":    "Aufbewahrung 24 Monate",
	}
	muster := regexp.MustCompile(`(?i)\b(UPDATE|DELETE\s+FROM)\s+audit_logs?\b`)
	var verstoesse []string
	err := filepath.WalkDir(".", func(pfad string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == "node_modules" || d.Name() == ".git" || d.Name() == "frontend" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(pfad, ".go") || strings.HasSuffix(pfad, "_test.go") {
			return nil
		}
		inhalt, err := os.ReadFile(pfad) // #nosec G304 -- Repo-Dateien
		if err != nil {
			return err
		}
		if !muster.Match(inhalt) {
			return nil
		}
		if _, ok := erlaubt[filepath.ToSlash(pfad)]; !ok {
			verstoesse = append(verstoesse, pfad)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(verstoesse) > 0 {
		t.Fatalf("neue Schreibtür auf audit_log/audit_logs in %v — Append-only ist Konvention; "+
			"nur DSGVO-Tilgung und Aufbewahrung dürfen das. Begründen und in die Liste in diesem Test aufnehmen.", verstoesse)
	}
	for datei := range erlaubt {
		if _, err := os.Stat(datei); err != nil {
			t.Errorf("erlaubte Datei %s existiert nicht mehr — Liste pflegen", datei)
		}
	}
}
