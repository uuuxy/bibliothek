package pruefung_test

import (
	"os"
	"testing"
)

// wechsleInDieWurzel wechselt für die Dauer des Tests in die Wurzel des Repositorys: Die Prüfungen
// lesen den Baum mit Pfaden ab dort, go test startet sie aber im Ordner des Pakets.
func wechsleInDieWurzel(t *testing.T) {
	t.Helper()
	t.Chdir("..")
	if _, err := os.Stat("go.mod"); err != nil {
		t.Fatalf("keine go.mod im Arbeitsverzeichnis, die Prüfung liefe am falschen Ort: %v", err)
	}
}
