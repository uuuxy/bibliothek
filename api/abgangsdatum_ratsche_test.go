package api

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Quelltext-Ratsche: Das Datum setzt der Trigger, sonst niemand. Schriebe ein Schreibpfad
// es selbst, hätte das Abgangsbuch wieder so viele Wahrheiten wie Türen — und die nächste
// Tür vergäße es. Erlaubt ist nur, was ein Test zum Zurückdatieren tut.
func TestAbgangsdatum_KeinSchreiberSetztEsSelbst(t *testing.T) {
	wurzel := ".."
	var fundstellen []string
	genannt := 0
	err := filepath.Walk(wurzel, func(pfad string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(pfad, ".go") || strings.HasSuffix(pfad, "_test.go") {
			return nil
		}
		roh, err := os.ReadFile(pfad) //nolint:gosec // Testlauf über den eigenen Quellbaum
		if err != nil {
			return err
		}
		for _, zeile := range strings.Split(string(roh), "\n") {
			if !strings.Contains(zeile, "ausgesondert_am") {
				continue
			}
			genannt++
			if strings.Contains(zeile, "SET") || strings.Contains(zeile, "ausgesondert_am =") {
				fundstellen = append(fundstellen, pfad+": "+strings.TrimSpace(zeile))
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Quellbaum lesen: %v", err)
	}
	// Die Abfragen des Abgangsbuchs lesen die Spalte. Sieht der Lauf keine Zeile mit ihr, reicht
	// er nicht bis zu ihnen und fände dort auch keinen Schreiber.
	if genannt == 0 {
		t.Fatalf("keine Go-Datei unter %q nennt ausgesondert_am: Der Lauf erreicht die Abfragen "+
			"des Abgangsbuchs nicht und prüft nichts", wurzel)
	}
	if len(fundstellen) > 0 {
		t.Errorf("ausgesondert_am wird im Code geschrieben — das Datum gehört dem Trigger "+
			"(Migration 128), sonst vergisst es die nächste Tür:\n%s", strings.Join(fundstellen, "\n"))
	}
}
