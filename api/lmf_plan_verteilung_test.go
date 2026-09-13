package api

import (
	"strings"
	"testing"

	"bibliothek/pkg/lmfplan"
	"bibliothek/repository"
)

// Vierte Zusicherung des Planers (Register, Paket 5): je Zeile ein Platz.
//
// Sie stand nirgends — weder in der Datenbank (sie ist keine Struktur, sondern eine
// Bedingung zwischen zwei Scheiben im Speicher) noch im Code. Getragen hat sie die
// Prüfung davor: pruefeLmfPlan verlangt stunden_je_tag zwischen 1 und 12, und beide
// Verteiler liefern genau dann einen Platz je Eintrag. Bei 0 geben sie eine LEERE Liste
// zurück — und der nächste Schritt greift mit plaetze[i] in sie hinein, in der Vorschau
// wie beim Speichern. Ein Indexfehler mitten im Speichern ist kein Fehler, den jemand
// lesen kann; er ist ein Absturz der Anfrage.
func TestVerteileLmfPlan_PlatzJeZeile(t *testing.T) {
	s := &Server{}
	entwurf := func(stundenJeTag int) lmfPlanEntwurf {
		return lmfPlanEntwurf{
			Plan: repository.LmfPlan{
				Art: repository.LmfTerminAusgabe, ErsterTag: "2026-09-10",
				Startstunde: 1, StundenJeTag: stundenJeTag,
				FreieTage: []repository.LmfFreierTag{},
			},
			Zeilen: []repository.LmfPlanZeile{{Klassen: []string{"05A"}}, {Klassen: []string{"05B"}}},
			Fest:   []*lmfplan.Platz{nil, nil},
		}
	}

	t.Run("Verteiler liefert nichts", func(t *testing.T) {
		e := entwurf(0)
		plaetze, _, err := s.verteileLmfPlan(&e)
		if err == nil {
			t.Fatalf("kein Fehler bei %d Plätzen für %d Zeilen — der nächste Zugriff wäre plaetze[i]",
				len(plaetze), len(e.Zeilen))
		}
		if !strings.Contains(err.Error(), "Zeilen") {
			t.Errorf("Fehlermeldung nennt das Missverhältnis nicht: %v", err)
		}
	})

	t.Run("Normalfall bleibt unberührt", func(t *testing.T) {
		e := entwurf(6)
		plaetze, _, err := s.verteileLmfPlan(&e)
		if err != nil {
			t.Fatalf("gültiger Rahmen abgelehnt: %v", err)
		}
		if len(plaetze) != len(e.Zeilen) {
			t.Errorf("%d Plätze für %d Zeilen", len(plaetze), len(e.Zeilen))
		}
	})
}
