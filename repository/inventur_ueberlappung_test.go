package repository

import (
	"context"
	"errors"
	"testing"
)

func sp(s string) *string { return &s }
func ip(i int) *int       { return &i }

// TestInventurScopesUeberlappen deckt die Regeln ab: global ist exklusiv, Signatur-
// Präfixe enthalten sich, kompatible Filter überlappen — disjunkte Bereiche NICHT
// (sonst würde legitime Parallel-Inventur blockiert). Signatur vs. Filter bleibt
// bewusst erlaubt (nicht sicher bestimmbar).
func TestInventurScopesUeberlappen(t *testing.T) {
	faelle := []struct {
		name     string
		aType    string
		a        InventurScope
		bType    string
		b        InventurScope
		erwartet bool
	}{
		{"global vs signatur", "global", InventurScope{}, "signature", InventurScope{Signatur: sp("BIB Deu")}, true},
		{"signatur vs global", "signature", InventurScope{Signatur: sp("BIB Deu")}, "global", InventurScope{}, true},
		{"global vs global", "global", InventurScope{}, "global", InventurScope{}, true},
		{"signatur präfix enthalten", "signature", InventurScope{Signatur: sp("BIB")}, "signature", InventurScope{Signatur: sp("BIB Deu")}, true},
		{"signatur präfix umgekehrt", "signature", InventurScope{Signatur: sp("BIB Deu")}, "signature", InventurScope{Signatur: sp("bib")}, true},
		{"signatur disjunkt", "signature", InventurScope{Signatur: sp("BIB Deu")}, "signature", InventurScope{Signatur: sp("BIB Mat")}, false},
		{"filter gleiches fach alle klassen vs eine", "filter", InventurScope{Subject: sp("Deutsch")}, "filter", InventurScope{Subject: sp("Deutsch"), Grade: ip(5)}, true},
		{"filter fach disjunkt", "filter", InventurScope{Subject: sp("Deutsch")}, "filter", InventurScope{Subject: sp("Mathe")}, false},
		{"filter klasse disjunkt", "filter", InventurScope{Grade: ip(5)}, "filter", InventurScope{Grade: ip(6)}, false},
		{"filter klasse-wildcard vs fach-wildcard", "filter", InventurScope{Grade: ip(5)}, "filter", InventurScope{Subject: sp("Deutsch")}, true},
		{"signatur vs filter bleibt erlaubt", "signature", InventurScope{Signatur: sp("BIB Deu")}, "filter", InventurScope{Subject: sp("Deutsch")}, false},
	}
	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			if got := inventurScopesUeberlappen(f.aType, f.a, f.bType, f.b); got != f.erwartet {
				t.Errorf("überlappt=%v, erwartet %v", got, f.erwartet)
			}
		})
	}
}

// TestCreateInventurSessionUeberlappungWirdAbgewiesen belegt die Wirkung am echten
// Postgres: Eine offene GLOBALE Inventur blockiert jeden weiteren Start (ErrInventur-
// Ueberlappt); zwei DISJUNKTE Signatur-Scopes dürfen dagegen parallel laufen.
func TestCreateInventurSessionUeberlappungWirdAbgewiesen(t *testing.T) {
	pool := pgTestPool(t)
	resetInventurDaten(t, pool)
	ctx := context.Background()
	repo := NewInventoryRepository(pool)

	// Offene globale Session.
	if _, err := repo.CreateInventurSession(ctx, "global", InventurScope{}, "Gesamtbestand", ""); err != nil {
		t.Fatalf("globale Session anlegen: %v", err)
	}

	// Signatur-Start muss jetzt an der Überlappung scheitern.
	sig := "BIB Deu"
	_, err := repo.CreateInventurSession(ctx, "signature", InventurScope{Signatur: &sig}, "Signatur BIB Deu", "")
	if !errors.Is(err, ErrInventurUeberlappt) {
		t.Fatalf("erwartet ErrInventurUeberlappt neben globaler Session, war: %v", err)
	}

	// Globale Session abschließen (Zeile freimachen).
	if _, err := pool.Exec(ctx, `UPDATE inventur_sessions SET abgeschlossen_am = now() WHERE abgeschlossen_am IS NULL`); err != nil {
		t.Fatalf("globale Session schließen: %v", err)
	}

	// Zwei DISJUNKTE Signatur-Scopes dürfen parallel offen sein.
	sigDeu, sigMat := "BIB Deu", "BIB Mat"
	if _, err := repo.CreateInventurSession(ctx, "signature", InventurScope{Signatur: &sigDeu}, "Signatur BIB Deu", ""); err != nil {
		t.Fatalf("Signatur Deu: %v", err)
	}
	if _, err := repo.CreateInventurSession(ctx, "signature", InventurScope{Signatur: &sigMat}, "Signatur BIB Mat", ""); err != nil {
		t.Fatalf("disjunkte Signatur Mat darf NICHT blockiert werden, war: %v", err)
	}

	// Aber "BIB" (Präfix von beiden) muss scheitern.
	sigBib := "BIB"
	_, err = repo.CreateInventurSession(ctx, "signature", InventurScope{Signatur: &sigBib}, "Signatur BIB", "")
	if !errors.Is(err, ErrInventurUeberlappt) {
		t.Fatalf("Präfix-Signatur BIB muss überlappen, war: %v", err)
	}
}
