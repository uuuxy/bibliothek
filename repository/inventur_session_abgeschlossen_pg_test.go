package repository

import (
	"context"
	"testing"
)

// ListAbgeschlosseneInventurSessions speist die Liste, über die ein Fehlbestandsbericht
// einer FERTIGEN Inventur wieder aufrufbar wird. Sie hatte bis hierher keinen Test —
// ausgerechnet die Funktion, deren SQL am 09.08.2026 auf CTE + LATERAL umgebaut wurde.
// Ein vertauschtes Zählpaar oder eine gedrehte Sortierung wäre niemandem aufgefallen:
// Beide Spalten sind Zahlen, beide Sortierrichtungen liefern Zeilen.
func TestListAbgeschlosseneInventurSessions(t *testing.T) {
	pool := pgTestPool(t)
	resetInventurDaten(t, pool)
	ctx := context.Background()
	repo := NewInventoryRepository(pool)

	// Drei Inventuren mit UNTERSCHIEDLICHEN Erfassungs- und Verlustzahlen — sonst
	// belegt der Test nicht, dass die beiden Zählungen an der richtigen Spalte hängen.
	// Je Scope: n Exemplare, davon k gescannt; der Rest wird beim Abschluss Verlust.
	faelle := []struct {
		signatur          string
		exemplare, gescan int
	}{
		{"Erdkunde", 6, 5}, // 5 erfasst, 1 Verlust
		{"Physik", 5, 2},   // 2 erfasst, 3 Verluste
		{"Chemie", 4, 4},   // 4 erfasst, 0 Verluste
	}

	for _, f := range faelle {
		ex := seedSignaturMitExemplaren(t, pool, f.signatur, f.exemplare)
		signatur := f.signatur
		sess, err := repo.CreateInventurSession(ctx, "signature", InventurScope{Signatur: &signatur}, f.signatur, "")
		if err != nil {
			t.Fatalf("Session %s anlegen: %v", f.signatur, err)
		}
		for _, id := range ex[:f.gescan] {
			if err := repo.RecordInventurScan(ctx, sess.ID, id); err != nil {
				t.Fatalf("Scan %s: %v", f.signatur, err)
			}
		}
		if _, err := repo.FinishInventurSession(ctx, sess.ID, InventurScope{Signatur: &signatur}); err != nil {
			t.Fatalf("Abschluss %s: %v", f.signatur, err)
		}
	}

	sessions, err := repo.ListAbgeschlosseneInventurSessions(ctx, 10)
	if err != nil {
		t.Fatalf("ListAbgeschlosseneInventurSessions: %v", err)
	}
	if len(sessions) != 3 {
		t.Fatalf("Anzahl abgeschlossener Sessions: erwartet 3, war %d", len(sessions))
	}

	// Neueste zuerst: Chemie wurde zuletzt abgeschlossen.
	if sessions[0].ScopeLabel != "Chemie" {
		t.Errorf("Sortierung: erwartet Chemie zuerst (zuletzt abgeschlossen), war %q", sessions[0].ScopeLabel)
	}

	erwartet := map[string]struct{ erfasst, verluste int }{
		"Erdkunde": {5, 1},
		"Physik":   {2, 3},
		"Chemie":   {4, 0},
	}
	for _, s := range sessions {
		will, bekannt := erwartet[s.ScopeLabel]
		if !bekannt {
			t.Errorf("unerwartete Session %q", s.ScopeLabel)
			continue
		}
		if s.Erfasst != will.erfasst {
			t.Errorf("%s: erfasst = %d, erwartet %d", s.ScopeLabel, s.Erfasst, will.erfasst)
		}
		if s.Verluste != will.verluste {
			t.Errorf("%s: verluste = %d, erwartet %d", s.ScopeLabel, s.Verluste, will.verluste)
		}
	}
}

// TestListAbgeschlosseneInventurSessions_Kappung sichert, dass das LIMIT die NEUESTEN
// Sessions behält. Seit dem CTE-Umbau kappt die Abfrage VOR dem Zählen — nimmt die CTE
// die falschen Zeilen, fällt es ohne diesen Test nicht auf.
func TestListAbgeschlosseneInventurSessions_Kappung(t *testing.T) {
	pool := pgTestPool(t)
	resetInventurDaten(t, pool)
	ctx := context.Background()
	repo := NewInventoryRepository(pool)

	namen := []string{"Alpha", "Beta", "Gamma", "Delta"}
	for _, name := range namen {
		seedSignaturMitExemplaren(t, pool, name, 2)
		signatur := name
		sess, err := repo.CreateInventurSession(ctx, "signature", InventurScope{Signatur: &signatur}, name, "")
		if err != nil {
			t.Fatalf("Session %s anlegen: %v", name, err)
		}
		if _, err := repo.FinishInventurSession(ctx, sess.ID, InventurScope{Signatur: &signatur}); err != nil {
			t.Fatalf("Abschluss %s: %v", name, err)
		}
	}

	sessions, err := repo.ListAbgeschlosseneInventurSessions(ctx, 2)
	if err != nil {
		t.Fatalf("ListAbgeschlosseneInventurSessions: %v", err)
	}
	if len(sessions) != 2 {
		t.Fatalf("Kappung auf 2: erwartet 2 Sessions, war %d", len(sessions))
	}
	if sessions[0].ScopeLabel != "Delta" || sessions[1].ScopeLabel != "Gamma" {
		t.Errorf("gekappt wurden die falschen: erwartet [Delta Gamma], war [%s %s]",
			sessions[0].ScopeLabel, sessions[1].ScopeLabel)
	}
}
