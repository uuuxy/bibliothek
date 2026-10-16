package repository

import (
	"context"
	"errors"
	"testing"
)

func sp(s string) *string { return &s }
func ip(i int) *int       { return &i }

// Die Regeln: „global" schließt jeden anderen Lauf aus, Signaturen mit gemeinsamem Präfix
// enthalten einander, vereinbare Filter überlappen, getrennte Bereiche nicht (sonst wären Läufe
// nebeneinander gesperrt). Was sich an den Feldern nicht entscheiden lässt (Signatur gegen
// Filter, eine unbekannte Art), geht an den Abgleich mit dem Bestand.
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

// Am echten Postgres: Ein offener Lauf über den ganzen Bestand sperrt jeden weiteren Start
// (ErrInventurUeberlappt); zwei getrennte Signaturen laufen nebeneinander.
func TestInventurlaufUeberlappungWirdAbgewiesen(t *testing.T) {
	pool := pgTestPool(t)
	resetInventurDaten(t, pool)
	ctx := context.Background()
	repo := NewInventurAbfragen(pool)

	// Ein offener Lauf über den ganzen Bestand.
	if _, err := repo.LegeLaufAn(ctx, "global", InventurBereich{}, "Gesamtbestand", ""); err != nil {
		t.Fatalf("globale Session anlegen: %v", err)
	}

	// Signatur-Start muss jetzt an der Überlappung scheitern.
	sig := "BIB Deu"
	_, err := repo.LegeLaufAn(ctx, "signature", InventurBereich{Signatur: &sig}, "Signatur BIB Deu", "")
	if !errors.Is(err, ErrInventurUeberlappt) {
		t.Fatalf("repo.LegeLaufAn(…) = %v, erwartet ErrInventurUeberlappt neben globaler Session", err)
	}

	// Den Lauf über den ganzen Bestand abschließen; die Zeile wird frei.
	if _, err := pool.Exec(ctx, `UPDATE inventur_sessions SET abgeschlossen_am = now() WHERE abgeschlossen_am IS NULL`); err != nil {
		t.Fatalf("globale Session schließen: %v", err)
	}

	// Zwei getrennte Signaturen sind nebeneinander offen.
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

// Am echten Postgres: Signatur gegen Filter wird am Bestand geprüft. Trägt ein Titel Signatur
// und Fach, umfassen beide Bereiche dieselben Exemplare; ein Scan landet aber nur in einem
// Lauf, und der Abschluss des anderen sonderte das Buch als Verlust aus. Ohne gemeinsamen
// Bestand bleiben beide Läufe nebeneinander erlaubt.
func TestInventurSignaturGegenFilterUeberlappungAmBestand(t *testing.T) {
	pool := pgTestPool(t)
	resetInventurDaten(t, pool)
	ctx := context.Background()
	repo := NewInventurAbfragen(pool)

	// Ein Titel, der in beiden Bereichen liegt: Signatur „BIB Deu" und Fach Deutsch.
	legeSignaturFachExemplarAn(t, pool, "BIB Deu 5 KRÜ", "Deutsch", 5, 6, "BC-KREUZ-1")

	sigDeu := "BIB Deu"
	if _, err := repo.LegeLaufAn(ctx, "signature", InventurBereich{Signatur: &sigDeu}, "Signatur BIB Deu", ""); err != nil {
		t.Fatalf("Signatur-Session anlegen: %v", err)
	}

	// Der Filter „Deutsch" trifft denselben Titel und wird als Überlappung abgewiesen.
	deutsch := "Deutsch"
	_, err := repo.LegeLaufAn(ctx, "filter", InventurBereich{Fach: &deutsch}, "Fach Deutsch", "")
	if !errors.Is(err, ErrInventurUeberlappt) {
		t.Fatalf("Filter Deutsch trifft den Signatur-Bestand, erwartet ErrInventurUeberlappt, war: %v", err)
	}

	// Auch der Jahrgang allein reicht: 5 liegt im Jahrgangsbereich 5–6.
	kl5 := 5
	_, err = repo.LegeLaufAn(ctx, "filter", InventurBereich{Jahrgang: &kl5}, "Klasse 5", "")
	if !errors.Is(err, ErrInventurUeberlappt) {
		t.Fatalf("Filter Klasse 5 trifft den Signatur-Bestand, erwartet ErrInventurUeberlappt, war: %v", err)
	}

	// Getrennt: Kein Titel trägt Mathematik, beide Läufe sind erlaubt.
	mathe := "Mathematik"
	if _, err := repo.LegeLaufAn(ctx, "filter", InventurBereich{Fach: &mathe}, "Fach Mathematik", ""); err != nil {
		t.Fatalf("Filter Mathematik hat keinen gemeinsamen Bestand und darf NICHT blockiert werden, war: %v", err)
	}

	// Umgekehrte Richtung: Der Lauf mit Filter ist offen, der Start mit Signatur scheitert.
	resetInventurDaten(t, pool)
	legeSignaturFachExemplarAn(t, pool, "BIB Deu 5 KRÜ", "Deutsch", 5, 6, "BC-KREUZ-2")
	if _, err := repo.LegeLaufAn(ctx, "filter", InventurBereich{Fach: &deutsch}, "Fach Deutsch", ""); err != nil {
		t.Fatalf("Filter-Session anlegen: %v", err)
	}
	_, err = repo.LegeLaufAn(ctx, "signature", InventurBereich{Signatur: &sigDeu}, "Signatur BIB Deu", "")
	if !errors.Is(err, ErrInventurUeberlappt) {
		t.Fatalf("Signatur-Start neben Filter Deutsch muss überlappen, war: %v", err)
	}
}
