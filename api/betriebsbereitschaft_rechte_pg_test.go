package api

import (
	"context"
	"strings"
	"testing"

	"bibliothek/db"
	"bibliothek/repository"
)

// Die Drift-Kette am echten Postgres: Seed schreibt die Vorgabe, das Repository
// liest sie zurück, die Prüfung meldet Deckung — und nach EINEM von Hand
// verstellten Wert (genau so entsteht Drift: DO NOTHING lässt die alte Zeile
// stehen) meldet sie exakt diese eine Zeile. Reine Funktionstests können die
// Paarung Seed↔Tabelle↔Leser nicht beweisen; genau die ist hier die Aussage.
func TestRechteVorgabe_DriftKetteAmEchtenPostgres(t *testing.T) {
	pool := pgTestPool(t)
	ctx := context.Background()

	datenbank := &db.Database{Pool: pool}
	if err := datenbank.InitPermissions(ctx); err != nil {
		t.Fatalf("Seed: %v", err)
	}

	zustand := repository.NewBetriebszustandRepository(pool)
	lage := lageEingerichtet()

	// (1) Frisch geseedet: Live deckt sich mit der Vorgabe.
	live, err := zustand.LadeRollenRechte(ctx)
	if err != nil {
		t.Fatalf("Live-Rechte lesen: %v", err)
	}
	lage.RechteLive = live
	if b := befundZu(t, Pruefe(lage), "Rechte-Vorgabe"); b.Stufe != StufeOK {
		t.Fatalf("frisch geseedet muss OK sein, ist %q: %s", b.Stufe, b.Befund)
	}

	// (2) Drift nachstellen: eine Live-Zeile kippt (wie eine Alt-Anlage nach
	// einer Code-Änderung — oder ein Admin-Eingriff).
	if _, err := pool.Exec(ctx,
		`UPDATE role_permissions SET allowed = true WHERE role = 'HELFER' AND permission = 'view_students'`); err != nil {
		t.Fatalf("Drift stellen: %v", err)
	}
	live, err = zustand.LadeRollenRechte(ctx)
	if err != nil {
		t.Fatalf("Live-Rechte erneut lesen: %v", err)
	}
	lage.RechteLive = live
	b := befundZu(t, Pruefe(lage), "Rechte-Vorgabe")
	if b.Stufe != StufeWarnung {
		t.Fatalf("Drift muss Warnung sein, ist %q: %s", b.Stufe, b.Befund)
	}
	if !strings.Contains(b.Befund, "HELFER/view_students live an, Vorgabe aus") {
		t.Errorf("die Meldung nennt die gedriftete Zeile nicht: %s", b.Befund)
	}

	// Zurückdrehen, damit nachfolgende Tests im Paket die Vorgabe vorfinden.
	if _, err := pool.Exec(ctx,
		`UPDATE role_permissions SET allowed = false WHERE role = 'HELFER' AND permission = 'view_students'`); err != nil {
		t.Fatalf("Drift zurückdrehen: %v", err)
	}
}
