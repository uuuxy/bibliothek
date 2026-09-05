package repository

import (
	"context"
	"testing"
	"time"

	"bibliothek/pkg/lmfplan"
	"bibliothek/pkg/schulzeit"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Der Lookup, den der Ausleihdienst beim Ausleihen eines Schulbuchs macht: der nächste
// Rückgabe-Termin der Klasse ab heute — Schreibvariante egal, Ausgabe-Zeilen zählen nicht,
// vergangene Termine auch nicht. Die Termine kommen aus Plänen (Migration 097): ein
// vergangener Rückgabe-Plan, ein künftiger mit 9H1 an zwei Tagen, ein Ausgabe-Plan.
func TestRueckgabeTerminFuerKlasse(t *testing.T) {
	pool := pgTestPool(t)
	ctx := context.Background()
	repo := NewLmfTerminRepository(pool)
	t.Cleanup(func() { raeumeLmfPlaene(t, pool) })

	speicherePlan(t, repo, LmfTerminRueckgabe, "2026-06-01", 1, 6,
		[]LmfPlanZeile{{Klassen: []string{"9H1"}, Vermerk: "vergangen"}}, nil)
	// 9H1 am 28.06. (Zeile 1) und noch einmal am 05.07. (Zeile 6, hinter dem Wochenende).
	speicherePlan(t, repo, LmfTerminRueckgabe, "2027-06-28", 1, 1,
		[]LmfPlanZeile{{Klassen: []string{"9H1"}}, {Klassen: []string{"9H2"}}, {Klassen: []string{"10R1"}},
			{Klassen: []string{"10R2"}}, {Klassen: []string{"10R3"}}, {Klassen: []string{"9H1"}, Vermerk: "zweiter Termin"}}, nil)
	speicherePlan(t, repo, LmfTerminAusgabe, "2027-08-10", 2, 6,
		[]LmfPlanZeile{{Klassen: []string{"7G1"}, Vermerk: "neu"}}, nil)
	heute := time.Date(2026, time.September, 5, 12, 0, 0, 0, schulzeit.Zone())

	termin, ok, err := repo.RueckgabeTerminFuerKlasse(ctx, "09h1", heute)
	if err != nil || !ok {
		t.Fatalf("Termin für 09h1: ok=%v err=%v", ok, err)
	}
	if termin.Format("2006-01-02") != "2027-06-28" {
		t.Errorf("nächster Termin = %v, erwartet 28.06.2027 (nicht der spätere, nicht der vergangene)", termin)
	}
	if _, ok, err := repo.RueckgabeTerminFuerKlasse(ctx, "7G1", heute); ok || err != nil {
		t.Errorf("eine Ausgabe-Zeile darf keine Frist liefern (ok=%v err=%v)", ok, err)
	}
	if _, ok, err := repo.RueckgabeTerminFuerKlasse(ctx, "5F1", heute); ok || err != nil {
		t.Errorf("Klasse ohne Termin liefert einen (ok=%v err=%v)", ok, err)
	}
}

// speicherePlan verteilt die Zeilen wie der Handler (Mo–Fr, ohne Ferien) und speichert.
func speicherePlan(t *testing.T, repo *LmfTerminRepository, art, ersterTag string, startstunde, stundenJeTag int, zeilen []LmfPlanZeile, ausgelassen []string) LmfPlanStand {
	t.Helper()
	tag, err := time.ParseInLocation("2006-01-02", ersterTag, schulzeit.Zone())
	if err != nil {
		t.Fatal(err)
	}
	plaetze := lmfplan.VerteileMit(lmfplan.Rahmen{ErsterTag: tag, Startstunde: startstunde, StundenJeTag: stundenJeTag},
		make([]*lmfplan.Platz, len(zeilen)), lmfplan.Schultage(nil))
	st, err := repo.SaveLmfPlan(context.Background(),
		LmfPlan{Art: art, ErsterTag: ersterTag, Startstunde: startstunde, StundenJeTag: stundenJeTag},
		zeilen, plaetze, ausgelassen)
	if err != nil {
		t.Fatalf("Plan %s ab %s speichern: %v", art, ersterTag, err)
	}
	return st
}

// raeumeLmfPlaene löscht alle Pläne (CASCADE nimmt Zeilen, Klassen und Auslassungen mit).
func raeumeLmfPlaene(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), `DELETE FROM lmf_plaene`); err != nil {
		t.Logf("Aufräumen: %v", err)
	}
}
