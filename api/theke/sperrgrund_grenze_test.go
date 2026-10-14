package theke

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"bibliothek/internal/service"
)

// TestSperrgrundBleibtOhneViewStudentsGeheim: Der Freitext aus schueler.block_reason kann
// Zahlungsrückstände oder Familieninterna nennen (PII-Matrix Stufe 2). Die Theke erfährt,
// dass gesperrt ist, den Grund aber nur mit view_students. Ohne die Kürzung käme der volle Text
// („… Fall Jugendamt") bei einem Helfer-Konto durch.
func TestSperrgrundBleibtOhneViewStudentsGeheim(t *testing.T) {
	grund := "Eltern zahlen Schadensrechnung nicht - Fall Jugendamt"
	sperrFehler := &service.SperrgrundFehler{
		Kern:  fmt.Errorf("%w: Manuelle Sperre", service.ErrGesperrt),
		Grund: grund,
	}

	ohne := ohneSperrgrund(sperrFehler, false)
	if strings.Contains(ohne.Error(), grund) {
		t.Errorf("Sperrgrund erreicht Aufrufer ohne view_students: %s", ohne)
	}
	if !errors.Is(ohne, service.ErrGesperrt) {
		t.Error("gekürzter Fehler verliert ErrGesperrt — der HTTP-Status würde von 403 auf 500 kippen")
	}

	mit := ohneSperrgrund(sperrFehler, true)
	if !strings.Contains(mit.Error(), grund) {
		t.Errorf("berechtigte Aufrufer müssen den Grund sehen (Theken-Arbeitsfähigkeit): %s", mit)
	}

	// Andere Fehler (auch ErrGesperrt ohne Freitext, z. B. Ausleihlimit) unverändert.
	limit := fmt.Errorf("%w: Ausleihlimit von 5 Büchern überschritten", service.ErrGesperrt)
	if ohneSperrgrund(limit, false) != limit {
		t.Error("Fehler ohne Sperr-Freitext dürfen nicht angefasst werden")
	}
	if ohneSperrgrund(nil, false) != nil {
		t.Error("nil bleibt nil")
	}
}
