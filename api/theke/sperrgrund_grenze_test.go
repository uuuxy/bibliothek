package theke

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"bibliothek/internal/service"
)

// TestSperrgrundBleibtOhneViewStudentsGeheim: Der Freitext aus schueler.block_reason
// kann Zahlungsrückstände oder Familieninterna nennen (PII-Matrix Stufe 2). Die Theke
// erfährt DASS gesperrt ist — das WARUM nur mit view_students. Am 18.08.2026 live mit
// Helfer-Konto belegt: vor dem Fix kam der volle Text („… Fall Jugendamt") durch.
func TestSperrgrundBleibtOhneViewStudentsGeheim(t *testing.T) {
	grund := "Eltern zahlen Schadensrechnung nicht - Fall Jugendamt"
	sperrFehler := &service.SperrGrundFehler{
		Kern:  fmt.Errorf("%w: Manuelle Sperre", service.ErrBlocked),
		Grund: grund,
	}

	ohne := ohneSperrgrund(sperrFehler, false)
	if strings.Contains(ohne.Error(), grund) {
		t.Errorf("Sperrgrund erreicht Aufrufer ohne view_students: %s", ohne)
	}
	if !errors.Is(ohne, service.ErrBlocked) {
		t.Error("gekürzter Fehler verliert ErrBlocked — der HTTP-Status würde von 403 auf 500 kippen")
	}

	mit := ohneSperrgrund(sperrFehler, true)
	if !strings.Contains(mit.Error(), grund) {
		t.Errorf("berechtigte Aufrufer müssen den Grund sehen (Theken-Arbeitsfähigkeit): %s", mit)
	}

	// Andere Fehler (auch ErrBlocked ohne Freitext, z. B. Ausleihlimit) unverändert.
	limit := fmt.Errorf("%w: Ausleihlimit von 5 Büchern überschritten", service.ErrBlocked)
	if ohneSperrgrund(limit, false) != limit {
		t.Error("Fehler ohne Sperr-Freitext dürfen nicht angefasst werden")
	}
	if ohneSperrgrund(nil, false) != nil {
		t.Error("nil bleibt nil")
	}
}
