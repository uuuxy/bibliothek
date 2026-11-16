package repository

import (
	"sync"
	"testing"
)

// Der LUSD-Import bildet seine Namensschlüssel mit suchnorm, und zwei Importe oder Vorschauen
// können gleichzeitig laufen. Teilen sich die Aufrufe einen Wert mit Puffern, meldet der
// Race-Detektor der CI (go test -race) den Wettlauf, auch wenn die Schlüssel zufällig stimmen.
func TestSuchnorm_GleichzeitigeAufrufe(t *testing.T) {
	namen := map[string]string{
		"Öztürk":       "ozturk",
		"García":       "garcia",
		"Nguyễn":       "nguyen",
		"Łukasz":       "lukasz",
		"Anna Müller":  "anna muller",
		"Anna Mueller": "anna muller",
	}
	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 50 {
				for name, want := range namen {
					if got := suchnorm(name); got != want {
						t.Errorf("suchnorm(%q) = %q, erwartet %q", name, got, want)
						return
					}
				}
			}
		}()
	}
	wg.Wait()
}
