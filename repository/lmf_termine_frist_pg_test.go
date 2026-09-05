package repository

import (
	"context"
	"testing"
	"time"

	"bibliothek/pkg/schulzeit"
)

// Der Lookup, den der Ausleihdienst beim Ausleihen eines Schulbuchs macht: der nächste
// Rückgabe-Termin der Klasse ab heute — Schreibvariante egal, Ausgabe-Zeilen zählen nicht,
// vergangene Termine auch nicht.
func TestRueckgabeTerminFuerKlasse(t *testing.T) {
	pool := pgTestPool(t)
	ctx := context.Background()
	repo := NewLmfTerminRepository(pool)
	t.Cleanup(func() {
		if _, err := pool.Exec(ctx, `DELETE FROM lmf_termine WHERE vermerk LIKE 'PGLOOKUP%'`); err != nil {
			t.Logf("Aufräumen: %v", err)
		}
	})
	for _, x := range []LmfTermin{
		{Datum: "2027-06-28", Stunde: 3, Art: LmfTerminRueckgabe, Klassen: []string{"9H1"}, Vermerk: "PGLOOKUP a"},
		{Datum: "2027-07-05", Stunde: 1, Art: LmfTerminRueckgabe, Klassen: []string{"9H1"}, Vermerk: "PGLOOKUP b (später)"},
		{Datum: "2026-06-01", Stunde: 1, Art: LmfTerminRueckgabe, Klassen: []string{"9H1"}, Vermerk: "PGLOOKUP vergangen"},
		{Datum: "2027-08-10", Stunde: 2, Art: LmfTerminAusgabe, Klassen: []string{"7G1"}, Vermerk: "PGLOOKUP Ausgabe"},
	} {
		if _, err := repo.SaveLmfTermin(ctx, x); err != nil {
			t.Fatal(err)
		}
	}
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
