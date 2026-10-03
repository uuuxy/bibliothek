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

// Ein freier Tag des Plans wird übersprungen und mit seinem Grund als Ausfall genannt; ohne
// Grund heißt er „freier Tag". Der 10.09.2026 ist ein Donnerstag, bei einer Stunde je Tag
// liegt jede Zeile auf einem eigenen Schultag.
func TestVerteileLmfPlan_FreieTageWerdenUebersprungenUndGenannt(t *testing.T) {
	s := &Server{}
	entwurf := func(frei []repository.LmfFreierTag) lmfPlanEntwurf {
		return lmfPlanEntwurf{
			Plan: repository.LmfPlan{
				Art: repository.LmfTerminAusgabe, ErsterTag: "2026-09-10",
				Startstunde: 1, StundenJeTag: 1, FreieTage: frei,
			},
			Zeilen: []repository.LmfPlanZeile{{Klassen: []string{"05A"}}, {Klassen: []string{"05B"}}, {Klassen: []string{"05C"}}},
			Fest:   []*lmfplan.Platz{nil, nil, nil},
		}
	}

	e := entwurf([]repository.LmfFreierTag{{Datum: "2026-09-11", Grund: "Pädagogischer Tag"}, {Datum: "2026-09-14"}})
	plaetze, ausfaelle, err := s.verteileLmfPlan(&e)
	if err != nil {
		t.Fatalf("Plan mit freien Tagen abgelehnt: %v", err)
	}
	var tage []string
	for _, p := range plaetze {
		tage = append(tage, p.Datum.Format("2006-01-02"))
	}
	if got := strings.Join(tage, ","); got != "2026-09-10,2026-09-15,2026-09-16" {
		t.Errorf("Plätze %s, erwartet 2026-09-10,2026-09-15,2026-09-16", got)
	}
	var genannt []string
	for _, a := range ausfaelle {
		genannt = append(genannt, a.Datum+" "+a.Grund)
	}
	if got := strings.Join(genannt, ","); got != "2026-09-11 Pädagogischer Tag,2026-09-14 freier Tag" {
		t.Errorf("Ausfälle %q, erwartet den pädagogischen Tag und den freien Tag ohne Grund", got)
	}

	e = entwurf([]repository.LmfFreierTag{{Datum: "kein Datum"}})
	if _, _, err := s.verteileLmfPlan(&e); err == nil {
		t.Error("ein freier Tag ohne lesbares Datum muss die Verteilung abbrechen")
	}
}
