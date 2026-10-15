package bereitschaft

import (
	"context"
	"strings"
	"testing"

	"bibliothek/db"
	"bibliothek/internal/pgtest"
	"bibliothek/repository"
)

// Die Kette an Postgres: Der Seed schreibt die Vorgabe, die Abfrage liest sie zurück, die
// Prüfung meldet Deckung, und nach einem von Hand verstellten Wert meldet sie genau diese
// eine Zeile. Dass Seed, Tabelle und Abfrage zusammenpassen, zeigt nur dieser Test.
func TestRechteVorgabe_DriftKetteAmEchtenPostgres(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()

	datenbank := &db.Verbindung{Pool: pool}
	if err := datenbank.SchreibeRechteVorgabe(ctx); err != nil {
		t.Fatalf("Seed: %v", err)
	}

	zustand := repository.NewBetriebszustandAbfragen(pool)
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

	// (2) Abweichung nachstellen: Eine Zeile kippt, wie auf einer älteren Anlage nach einer
	// Änderung im Code oder nach einem Eingriff der Verwaltung.
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
