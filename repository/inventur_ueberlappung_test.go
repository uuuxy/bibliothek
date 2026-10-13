package repository

import (
	"context"
	"errors"
	"testing"
)

func sp(s string) *string { return &s }
func ip(i int) *int       { return &i }

// TestInventurBereicheUeberlappen deckt die Regeln ab: global ist exklusiv, Signatur-
// Präfixe enthalten sich, kompatible Filter überlappen — disjunkte Bereiche NICHT
// (sonst würde legitime Parallel-Inventur blockiert). Alles syntaktisch nicht
// Entscheidbare (Signatur vs. Filter, unbekannte Typen) geht an den Bestandsabgleich.
func TestInventurBereicheUeberlappen(t *testing.T) {
	faelle := []struct {
		name     string
		aType    string
		a        InventurBereich
		bType    string
		b        InventurBereich
		erwartet bereichUeberlappung
	}{
		{"global vs signatur", "global", InventurBereich{}, "signature", InventurBereich{Signatur: sp("BIB Deu")}, ueberlapptImmer},
		{"signatur vs global", "signature", InventurBereich{Signatur: sp("BIB Deu")}, "global", InventurBereich{}, ueberlapptImmer},
		{"global vs global", "global", InventurBereich{}, "global", InventurBereich{}, ueberlapptImmer},
		{"signatur präfix enthalten", "signature", InventurBereich{Signatur: sp("BIB")}, "signature", InventurBereich{Signatur: sp("BIB Deu")}, ueberlapptImmer},
		{"signatur präfix umgekehrt", "signature", InventurBereich{Signatur: sp("BIB Deu")}, "signature", InventurBereich{Signatur: sp("bib")}, ueberlapptImmer},
		{"signatur disjunkt", "signature", InventurBereich{Signatur: sp("BIB Deu")}, "signature", InventurBereich{Signatur: sp("BIB Mat")}, ueberlapptNie},
		{"filter gleiches fach alle klassen vs eine", "filter", InventurBereich{Fach: sp("Deutsch")}, "filter", InventurBereich{Fach: sp("Deutsch"), Jahrgang: ip(5)}, ueberlapptImmer},
		{"filter fach disjunkt", "filter", InventurBereich{Fach: sp("Deutsch")}, "filter", InventurBereich{Fach: sp("Mathe")}, ueberlapptNie},
		{"filter klasse disjunkt", "filter", InventurBereich{Jahrgang: ip(5)}, "filter", InventurBereich{Jahrgang: ip(6)}, ueberlapptNie},
		{"filter klasse-wildcard vs fach-wildcard", "filter", InventurBereich{Jahrgang: ip(5)}, "filter", InventurBereich{Fach: sp("Deutsch")}, ueberlapptImmer},
		{"signatur vs filter entscheidet der bestand", "signature", InventurBereich{Signatur: sp("BIB Deu")}, "filter", InventurBereich{Fach: sp("Deutsch")}, ueberlapptJeNachBestand},
		{"filter vs signatur entscheidet der bestand", "filter", InventurBereich{Fach: sp("Deutsch")}, "signature", InventurBereich{Signatur: sp("BIB Deu")}, ueberlapptJeNachBestand},
		{"unbekannter scope-typ fällt konservativ auf den bestand", "signature", InventurBereich{Signatur: sp("BIB Deu")}, "kuenftig", InventurBereich{}, ueberlapptJeNachBestand},
	}
	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			if got := bestimmeUeberlappung(f.aType, f.a, f.bType, f.b); got != f.erwartet {
				t.Errorf("überlappt=%v, erwartet %v", got, f.erwartet)
			}
		})
	}
}

// TestInventurlaufUeberlappungWirdAbgewiesen belegt die Wirkung am echten
// Postgres: Eine offene GLOBALE Inventur blockiert jeden weiteren Start (ErrInventur-
// Ueberlappt); zwei DISJUNKTE Signatur-Scopes dürfen dagegen parallel laufen.
func TestInventurlaufUeberlappungWirdAbgewiesen(t *testing.T) {
	pool := pgTestPool(t)
	resetInventurDaten(t, pool)
	ctx := context.Background()
	repo := NewInventurAbfragen(pool)

	// Offene globale Session.
	if _, err := repo.LegeLaufAn(ctx, "global", InventurBereich{}, "Gesamtbestand", ""); err != nil {
		t.Fatalf("globale Session anlegen: %v", err)
	}

	// Signatur-Start muss jetzt an der Überlappung scheitern.
	sig := "BIB Deu"
	_, err := repo.LegeLaufAn(ctx, "signature", InventurBereich{Signatur: &sig}, "Signatur BIB Deu", "")
	if !errors.Is(err, ErrInventurUeberlappt) {
		t.Fatalf("erwartet ErrInventurUeberlappt neben globaler Session, war: %v", err)
	}

	// Globale Session abschließen (Zeile freimachen).
	if _, err := pool.Exec(ctx, `UPDATE inventur_sessions SET abgeschlossen_am = now() WHERE abgeschlossen_am IS NULL`); err != nil {
		t.Fatalf("globale Session schließen: %v", err)
	}

	// Zwei DISJUNKTE Signatur-Scopes dürfen parallel offen sein.
	sigDeu, sigMat := "BIB Deu", "BIB Mat"
	if _, err := repo.LegeLaufAn(ctx, "signature", InventurBereich{Signatur: &sigDeu}, "Signatur BIB Deu", ""); err != nil {
		t.Fatalf("Signatur Deu: %v", err)
	}
	if _, err := repo.LegeLaufAn(ctx, "signature", InventurBereich{Signatur: &sigMat}, "Signatur BIB Mat", ""); err != nil {
		t.Fatalf("disjunkte Signatur Mat darf NICHT blockiert werden, war: %v", err)
	}

	// Aber "BIB" (Präfix von beiden) muss scheitern.
	sigBib := "BIB"
	_, err = repo.LegeLaufAn(ctx, "signature", InventurBereich{Signatur: &sigBib}, "Signatur BIB", "")
	if !errors.Is(err, ErrInventurUeberlappt) {
		t.Fatalf("Präfix-Signatur BIB muss überlappen, war: %v", err)
	}
}

// TestInventurSignaturGegenFilterUeberlappungAmBestand belegt am echten Postgres, dass die
// Kreuz-Kombination Signatur-Scope vs. Filter-Scope am BESTAND geprüft wird: Trägt ein
// Titel Signatur UND Fach, umfassen beide Scopes dieselben Exemplare — ein Scan landet
// aber nur in EINER Session, und der Abschluss der anderen würde das Buch als Verlust
// aussondern. Disjunkte Kreuz-Kombinationen (kein gemeinsamer Bestand) bleiben erlaubt.
func TestInventurSignaturGegenFilterUeberlappungAmBestand(t *testing.T) {
	pool := pgTestPool(t)
	resetInventurDaten(t, pool)
	ctx := context.Background()
	repo := NewInventurAbfragen(pool)

	// Ein Titel, der in beiden Welten liegt: Regalbereich "BIB Deu" UND Fach Deutsch.
	seedSignaturFachExemplar(t, pool, "BIB Deu 5 KRÜ", "Deutsch", 5, 6, "BC-KREUZ-1")

	sigDeu := "BIB Deu"
	if _, err := repo.LegeLaufAn(ctx, "signature", InventurBereich{Signatur: &sigDeu}, "Signatur BIB Deu", ""); err != nil {
		t.Fatalf("Signatur-Session anlegen: %v", err)
	}

	// Filter "Deutsch" trifft denselben Titel → muss als Überlappung abgewiesen werden.
	deutsch := "Deutsch"
	_, err := repo.LegeLaufAn(ctx, "filter", InventurBereich{Fach: &deutsch}, "Fach Deutsch", "")
	if !errors.Is(err, ErrInventurUeberlappt) {
		t.Fatalf("Filter Deutsch trifft den Signatur-Bestand, erwartet ErrInventurUeberlappt, war: %v", err)
	}

	// Auch die Klassen-Dimension allein reicht: Kl. 5 liegt im Jahrgangsbereich 5–6.
	kl5 := 5
	_, err = repo.LegeLaufAn(ctx, "filter", InventurBereich{Jahrgang: &kl5}, "Klasse 5", "")
	if !errors.Is(err, ErrInventurUeberlappt) {
		t.Fatalf("Filter Klasse 5 trifft den Signatur-Bestand, erwartet ErrInventurUeberlappt, war: %v", err)
	}

	// Disjunkt: kein Titel trägt Mathematik → parallel erlaubt.
	mathe := "Mathematik"
	if _, err := repo.LegeLaufAn(ctx, "filter", InventurBereich{Fach: &mathe}, "Fach Mathematik", ""); err != nil {
		t.Fatalf("Filter Mathematik hat keinen gemeinsamen Bestand und darf NICHT blockiert werden, war: %v", err)
	}

	// Umgekehrte Richtung: Filter-Session offen, Signatur-Start muss scheitern.
	resetInventurDaten(t, pool)
	seedSignaturFachExemplar(t, pool, "BIB Deu 5 KRÜ", "Deutsch", 5, 6, "BC-KREUZ-2")
	if _, err := repo.LegeLaufAn(ctx, "filter", InventurBereich{Fach: &deutsch}, "Fach Deutsch", ""); err != nil {
		t.Fatalf("Filter-Session anlegen: %v", err)
	}
	_, err = repo.LegeLaufAn(ctx, "signature", InventurBereich{Signatur: &sigDeu}, "Signatur BIB Deu", "")
	if !errors.Is(err, ErrInventurUeberlappt) {
		t.Fatalf("Signatur-Start neben Filter Deutsch muss überlappen, war: %v", err)
	}
}
