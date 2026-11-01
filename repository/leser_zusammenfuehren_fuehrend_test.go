package repository

import (
	"testing"
	"time"
)

// Welcher Datensatz die Stammdaten stellt: Der aktive schlägt den Abgänger, sonst die jüngere
// LUSD-Bestätigung. Ohne Bestätigung und bei Gleichstand bleibt das Ziel führend.
func TestFuehrend(t *testing.T) {
	frueher := time.Date(2026, 5, 1, 10, 0, 0, 0, time.UTC)
	spaeter := frueher.Add(time.Hour)
	faelle := []struct {
		name         string
		ziel, quelle zusammenfuehrenZeile
		quelleFuehrt bool
	}{
		{name: "beide ohne Bestätigung"},
		{name: "nur die Quelle bestätigt", quelle: zusammenfuehrenZeile{bestaetigtAm: &frueher}, quelleFuehrt: true},
		{name: "nur das Ziel bestätigt", ziel: zusammenfuehrenZeile{bestaetigtAm: &frueher}},
		{
			name:         "Quelle jünger bestätigt",
			ziel:         zusammenfuehrenZeile{bestaetigtAm: &frueher},
			quelle:       zusammenfuehrenZeile{bestaetigtAm: &spaeter},
			quelleFuehrt: true,
		},
		{
			name:   "Ziel jünger bestätigt",
			ziel:   zusammenfuehrenZeile{bestaetigtAm: &spaeter},
			quelle: zusammenfuehrenZeile{bestaetigtAm: &frueher},
		},
		{
			name:   "Gleichstand",
			ziel:   zusammenfuehrenZeile{bestaetigtAm: &frueher},
			quelle: zusammenfuehrenZeile{bestaetigtAm: &frueher},
		},
		{
			name:         "aktive Quelle schlägt das jünger bestätigte Abgänger-Ziel",
			ziel:         zusammenfuehrenZeile{abgaenger: true, bestaetigtAm: &spaeter},
			quelle:       zusammenfuehrenZeile{bestaetigtAm: &frueher},
			quelleFuehrt: true,
		},
		{
			name:   "aktives Ziel schlägt die jünger bestätigte Abgänger-Quelle",
			ziel:   zusammenfuehrenZeile{bestaetigtAm: &frueher},
			quelle: zusammenfuehrenZeile{abgaenger: true, bestaetigtAm: &spaeter},
		},
	}
	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			fuehrt, andere := fuehrend(&f.ziel, &f.quelle)
			if f.quelleFuehrt && (fuehrt != &f.quelle || andere != &f.ziel) {
				t.Error("führend ist das Ziel, erwartet die Quelle")
			}
			if !f.quelleFuehrt && (fuehrt != &f.ziel || andere != &f.quelle) {
				t.Error("führend ist die Quelle, erwartet das Ziel")
			}
		})
	}
}
