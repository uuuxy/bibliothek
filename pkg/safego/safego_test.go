package safego

import (
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
