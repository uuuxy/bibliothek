package theke

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"bibliothek/internal/service"
)

// TestStatusDesFehlers: Der Status entscheidet an der Theke, welche Meldung erscheint, und
// landet mit der Antwort in der Ablage. Der Test hält jede Zuordnung fest, damit sich 404,
// 400 und 409 nicht unbemerkt vertauschen.
func TestStatusDesFehlers(t *testing.T) {
	faelle := []struct {
		name     string
		err      error
		erwartet int
	}{
		{"nicht gefunden", service.ErrNichtGefunden, http.StatusNotFound},
		{"gesperrt", service.ErrGesperrt, http.StatusForbidden},
		{"unzulässiger Zustand", service.ErrUngueltigerZustand, http.StatusBadRequest},
		{"Konflikt", service.ErrKonflikt, http.StatusConflict},
		{"unbekannter Fehler", errors.New("irgendein Fehler"), http.StatusInternalServerError},
		// So kommt ein Sperrfehler an, wenn ohneSperrgrund ihm den Freitext nimmt: der Kern
		// (selbst eine ErrGesperrt-Kette) mit dem Hinweis dahinter.
		{"eingewickelt, gesperrt", (&service.SperrgrundFehler{Kern: service.Sperre("Manuelle Sperre"), Grund: "x"}).OhneGrund(),
			http.StatusForbidden},
		{"eingewickelt, nicht gefunden", fmt.Errorf("leser: %w", service.ErrNichtGefunden), http.StatusNotFound},
	}

	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			if got := statusDesFehlers(f.err); got != f.erwartet {
				t.Errorf("statusDesFehlers(%v) = %d, erwartet %d", f.err, got, f.erwartet)
			}
		})
	}
}
