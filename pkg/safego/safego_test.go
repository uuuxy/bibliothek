package safego

import (
	"sync"
	"testing"
)

// TestGuardFaengtPanic beweist den Kern: Eine Goroutine, die mit Guard geschützt ist,
// nimmt den Prozess bei einem Panic NICHT mit. Ohne Guard würde dieser Test das
// gesamte Testbinary zum Absturz bringen (unrecovered panic in goroutine).
func TestGuardFaengtPanic(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		defer Guard("test-goroutine")
		panic("simulierter Parser-Absturz")
	}()

	wg.Wait() // Kehrt nur zurück, wenn das Panic gefangen wurde.
}

// TestGuardLaesstNormalenAblaufDurch stellt sicher, dass Guard im Normalfall (kein
// Panic) nichts stört und der Code hinter ihm ganz durchläuft.
func TestGuardLaesstNormalenAblaufDurch(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(1)

	erledigt := false
	go func() {
		defer wg.Done()
		defer Guard("test-goroutine")
		erledigt = true
	}()

	wg.Wait()
	if !erledigt {
		t.Error("Guard hat den normalen Ablauf unterbrochen")
	}
}
