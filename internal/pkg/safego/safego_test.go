package safego

import (
	"bytes"
	"log/slog"
	"strings"
	"sync"
	"testing"
)

// Ohne FangePanic beendete das Panic der Goroutine den ganzen Testlauf.
func TestFangePanic_ProzessUeberlebt(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		defer FangePanic("test-goroutine")
		panic("simulierter Parser-Absturz")
	}()

	wg.Wait() // kehrt nur zurück, wenn das Panic gefangen ist
}

func TestFangePanic_LaesstNormalenAblaufDurch(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(1)

	erledigt := false
	go func() {
		defer wg.Done()
		defer FangePanic("test-goroutine")
		erledigt = true
	}()

	wg.Wait()
	if !erledigt {
		t.Error("FangePanic hat den normalen Ablauf unterbrochen")
	}
}

// Ein Lauf in einer Schleife: Das Panic endet in FuehreAus, die Schleife läuft weiter, und das Log
// nennt den Lauf mit Stufe ERROR, damit er auf der Seite „Fehler im Log“ steht.
func TestFuehreAus_NurDerLaufEndet(t *testing.T) {
	var protokoll bytes.Buffer
	vorher := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&protokoll, nil)))
	t.Cleanup(func() { slog.SetDefault(vorher) })

	laeufe := 0
	for range 2 {
		FuehreAus("probe-lauf", func() {
			laeufe++
			panic("simulierter Absturz")
		})
	}
	if laeufe != 2 {
		t.Errorf("%d Läufe, erwartet 2: Der zweite Lauf kam nach dem Panic des ersten nicht", laeufe)
	}
	if n := strings.Count(protokoll.String(), "level=ERROR"); n != 2 || !strings.Contains(protokoll.String(), "goroutine=probe-lauf") {
		t.Errorf("Log mit %d Zeilen der Stufe ERROR, erwartet 2 mit dem Namen des Laufs:\n%s", n, protokoll.String())
	}
}
