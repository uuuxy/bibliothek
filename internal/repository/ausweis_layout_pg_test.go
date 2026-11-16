package repository

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
)

// Das Design speichert nur, wer auf dem Stand schreibt, den er geladen hat. Prüfen und
// Schreiben sind eine Anweisung: Von mehreren Plätzen, die gleichzeitig auf demselben Stand
// speichern, gewinnt genau einer, und die anderen erfahren es. Geprüft für das erste Design
// (noch keine Zeile) und für das Ersetzen.
func TestSchreibeAusweisLayout_VonGleichzeitigenGewinntEiner(t *testing.T) {
	pool := pgTestPool(t)
	ctx := context.Background()
	entferne := func() {
		if _, err := pool.Exec(context.Background(),
			`DELETE FROM system_einstellungen WHERE schluessel = $1`, AusweisLayoutSchluessel); err != nil {
			t.Errorf("Design entfernen: %v", err)
		}
	}
	entferne()
	t.Cleanup(entferne)

	// wettlauf lässt acht Plätze auf demselben Stand speichern und liefert den neuen Stand.
	wettlauf := func(t *testing.T, stand, runde string) string {
		t.Helper()
		const plaetze = 8
		var wg sync.WaitGroup
		staende := make([]string, plaetze)
		fehler := make([]error, plaetze)
		for i := range plaetze {
			wg.Add(1)
			go func() {
				defer wg.Done()
				staende[i], fehler[i] = SchreibeAusweisLayout(ctx, pool, fmt.Sprintf(`{"runde":%q,"platz":%d}`, runde, i), stand)
			}()
		}
		wg.Wait()

		sieger := -1
		for i, err := range fehler {
			switch {
			case err == nil && sieger == -1:
				sieger = i
			case err == nil:
				t.Errorf("Platz %d und Platz %d haben beide auf dem Stand %q gespeichert", sieger, i, stand)
			case !errors.Is(err, ErrAusweisLayoutVeraltet):
				t.Errorf("Platz %d: %v, erwartet ErrAusweisLayoutVeraltet", i, err)
			}
		}
		if sieger == -1 {
			t.Fatalf("kein Platz hat auf dem Stand %q gespeichert", stand)
		}
		wert, geladen, err := LadeAusweisLayout(ctx, pool)
		if err != nil {
			t.Fatalf("LadeAusweisLayout: %v", err)
		}
		if soll := fmt.Sprintf(`{"runde":%q,"platz":%d}`, runde, sieger); wert != soll {
			t.Errorf("gespeichert ist %s, erwartet das Design des Siegers %s", wert, soll)
		}
		if geladen != staende[sieger] || geladen == stand {
			t.Errorf("Stand nach dem Speichern %q, der Sieger bekam %q, vorher %q", geladen, staende[sieger], stand)
		}
		return geladen
	}

	erster := wettlauf(t, "", "anlegen")
	zweiter := wettlauf(t, erster, "ersetzen")

	// Der überholte Stand schreibt nicht mehr, der leere auch nicht.
	for _, alt := range []string{erster, ""} {
		if _, err := SchreibeAusweisLayout(ctx, pool, `{"runde":"zu spät"}`, alt); !errors.Is(err, ErrAusweisLayoutVeraltet) {
			t.Errorf("Speichern auf dem Stand %q: %v, erwartet ErrAusweisLayoutVeraltet", alt, err)
		}
	}
	if _, geladen, err := LadeAusweisLayout(ctx, pool); err != nil || geladen != zweiter {
		t.Errorf("Stand %q, Fehler %v; erwartet unverändert %q", geladen, err, zweiter)
	}
}
