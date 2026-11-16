package pruefung

import (
	"fmt"
	"os"
	"testing"
)

// Die Prüfungen lesen das Repository mit Pfaden ab der Wurzel, go test startet sie aber im
// Ordner des Pakets; deshalb wechselt TestMain einmal in die Wurzel und bricht ohne go.mod ab.
func TestMain(m *testing.M) {
	if err := os.Chdir(".."); err != nil {
		fmt.Fprintln(os.Stderr, "Wechsel in die Wurzel des Repositorys gescheitert:", err)
		os.Exit(1)
	}
	if _, err := os.Stat("go.mod"); err != nil {
		fmt.Fprintln(os.Stderr, "keine go.mod im Arbeitsverzeichnis, die Prüfungen liefen am falschen Ort:", err)
		os.Exit(1)
	}
	os.Exit(m.Run())
}
