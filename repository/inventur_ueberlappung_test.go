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
// (sonst würde legitime Parallel-Inventur blockiert). Alles syntaktisch nicht
// Entscheidbare (Signatur vs. Filter, unbekannte Typen) geht an den Bestandsabgleich.
func TestInventurScopesUeberlappen(t *testing.T) {
	faelle := []struct {
		name     string
		aType    string
		a        InventurScope
		bType    string
		b        InventurScope
		erwartet scopeUeberlappung
	}{
		{"global vs signatur", "global", InventurScope{}, "signature", InventurScope{Signatur: sp("BIB Deu")}, ueberlapptImmer},
		{"signatur vs global", "signature", InventurScope{Signatur: sp("BIB Deu")}, "global", InventurScope{}, ueberlapptImmer},
		{"global vs global", "global", InventurScope{}, "global", InventurScope{}, ueberlapptImmer},
		{"signatur präfix enthalten", "signature", InventurScope{Signatur: sp("BIB")}, "signature", InventurScope{Signatur: sp("BIB Deu")}, ueberlapptImmer},
		{"signatur präfix umgekehrt", "signature", InventurScope{Signatur: sp("BIB Deu")}, "signature", InventurScope{Signatur: sp("bib")}, ueberlapptImmer},
		{"signatur disjunkt", "signature", InventurScope{Signatur: sp("BIB Deu")}, "signature", InventurScope{Signatur: sp("BIB Mat")}, ueberlapptNie},
		{"filter gleiches fach alle klassen vs eine", "filter", InventurScope{Subject: sp("Deutsch")}, "filter", InventurScope{Subject: sp("Deutsch"), Grade: ip(5)}, ueberlapptImmer},
		{"filter fach disjunkt", "filter", InventurScope{Subject: sp("Deutsch")}, "filter", InventurScope{Subject: sp("Mathe")}, ueberlapptNie},
		{"filter klasse disjunkt", "filter", InventurScope{Grade: ip(5)}, "filter", InventurScope{Grade: ip(6)}, ueberlapptNie},
		{"filter klasse-wildcard vs fach-wildcard", "filter", InventurScope{Grade: ip(5)}, "filter", InventurScope{Subject: sp("Deutsch")}, ueberlapptImmer},
		{"signatur vs filter entscheidet der bestand", "signature", InventurScope{Signatur: sp("BIB Deu")}, "filter", InventurScope{Subject: sp("Deutsch")}, ueberlapptJeNachBestand},
		{"filter vs signatur entscheidet der bestand", "filter", InventurScope{Subject: sp("Deutsch")}, "signature", InventurScope{Signatur: sp("BIB Deu")}, ueberlapptJeNachBestand},
		{"unbekannter scope-typ fällt konservativ auf den bestand", "signature", InventurScope{Signatur: sp("BIB Deu")}, "kuenftig", InventurScope{}, ueberlapptJeNachBestand},
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

// TestInventurSignaturVsFilterUeberlappungAmBestand belegt am echten Postgres, dass die
// Kreuz-Kombination Signatur-Scope vs. Filter-Scope am BESTAND geprüft wird: Trägt ein
// Titel Signatur UND Fach, umfassen beide Scopes dieselben Exemplare — ein Scan landet
// aber nur in EINER Session, und der Abschluss der anderen würde das Buch als Verlust
// aussondern. Disjunkte Kreuz-Kombinationen (kein gemeinsamer Bestand) bleiben erlaubt.
func TestInventurSignaturVsFilterUeberlappungAmBestand(t *testing.T) {
	pool := pgTestPool(t)
	resetInventurDaten(t, pool)
	ctx := context.Background()
	repo := NewInventoryRepository(pool)

	// Ein Titel, der in beiden Welten liegt: Regalbereich "BIB Deu" UND Fach Deutsch.
	seedSignaturFachExemplar(t, pool, "BIB Deu 5 KRÜ", "Deutsch", 5, 6, "BC-KREUZ-1")

	sigDeu := "BIB Deu"
	if _, err := repo.CreateInventurSession(ctx, "signature", InventurScope{Signatur: &sigDeu}, "Signatur BIB Deu", ""); err != nil {
		t.Fatalf("Signatur-Session anlegen: %v", err)
	}

	// Filter "Deutsch" trifft denselben Titel → muss als Überlappung abgewiesen werden.
	deutsch := "Deutsch"
	_, err := repo.CreateInventurSession(ctx, "filter", InventurScope{Subject: &deutsch}, "Fach Deutsch", "")
	if !errors.Is(err, ErrInventurUeberlappt) {
		t.Fatalf("Filter Deutsch trifft den Signatur-Bestand, erwartet ErrInventurUeberlappt, war: %v", err)
	}

	// Auch die Klassen-Dimension allein reicht: Kl. 5 liegt im Jahrgangsbereich 5–6.
	kl5 := 5
	_, err = repo.CreateInventurSession(ctx, "filter", InventurScope{Grade: &kl5}, "Klasse 5", "")
	if !errors.Is(err, ErrInventurUeberlappt) {
		t.Fatalf("Filter Klasse 5 trifft den Signatur-Bestand, erwartet ErrInventurUeberlappt, war: %v", err)
	}

	// Disjunkt: kein Titel trägt Mathematik → parallel erlaubt.
	mathe := "Mathematik"
	if _, err := repo.CreateInventurSession(ctx, "filter", InventurScope{Subject: &mathe}, "Fach Mathematik", ""); err != nil {
		t.Fatalf("Filter Mathematik hat keinen gemeinsamen Bestand und darf NICHT blockiert werden, war: %v", err)
	}

	// Umgekehrte Richtung: Filter-Session offen, Signatur-Start muss scheitern.
	resetInventurDaten(t, pool)
	seedSignaturFachExemplar(t, pool, "BIB Deu 5 KRÜ", "Deutsch", 5, 6, "BC-KREUZ-2")
	if _, err := repo.CreateInventurSession(ctx, "filter", InventurScope{Subject: &deutsch}, "Fach Deutsch", ""); err != nil {
		t.Fatalf("Filter-Session anlegen: %v", err)
	}
	_, err = repo.CreateInventurSession(ctx, "signature", InventurScope{Signatur: &sigDeu}, "Signatur BIB Deu", "")
	if !errors.Is(err, ErrInventurUeberlappt) {
		t.Fatalf("Signatur-Start neben Filter Deutsch muss überlappen, war: %v", err)
	}
}
