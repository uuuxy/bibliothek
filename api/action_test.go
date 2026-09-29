package api

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"bibliothek/internal/service"
)

// TestMapServiceErrorToStatus: Der Status entscheidet an der Theke, welche Meldung
// erscheint, und landet mit der Antwort im Idempotenz-Cache. Über die Handler-Tests war
// bis zum 29.09.2026 nur „gesperrt → 403" abgesichert; 404, 400 und 409 ließen sich
// vertauschen, ohne dass ein Test rot wurde.
func TestMapServiceErrorToStatus(t *testing.T) {
	faelle := []struct {
		name     string
		err      error
		erwartet int
	}{
		{"nicht gefunden", service.ErrNotFound, http.StatusNotFound},
		{"gesperrt", service.ErrBlocked, http.StatusForbidden},
		{"unzulässiger Zustand", service.ErrInvalidState, http.StatusBadRequest},
		{"Konflikt", service.ErrConflict, http.StatusConflict},
		{"unbekannter Fehler", errors.New("irgendein Fehler"), http.StatusInternalServerError},
		// So kommt ein Sperrfehler an, wenn ohneSperrgrund ihm den Freitext nimmt: der Kern
		// (selbst eine ErrBlocked-Kette) mit dem Hinweis dahinter.
		{"eingewickelt, gesperrt", fmt.Errorf("%w — bitte an die Bibliotheksleitung wenden",
			fmt.Errorf("%w: Manuelle Sperre", service.ErrBlocked)), http.StatusForbidden},
		{"eingewickelt, nicht gefunden", fmt.Errorf("Leser: %w", service.ErrNotFound), http.StatusNotFound},
	}

	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			if got := mapServiceErrorToStatus(f.err); got != f.erwartet {
				t.Errorf("mapServiceErrorToStatus(%v) = %d, erwartet %d", f.err, got, f.erwartet)
			}
		})
	}
}
