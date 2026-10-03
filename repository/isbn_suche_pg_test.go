package repository

import (
	"context"
	"testing"
)

// Die Suchen vergleichen eine getippte ISBN als Teilstring. Die zehnstellige vom Titelblatt
// und die dreizehnstellige, unter der die Datenbank den Titel führt (Migration 157), enden
// auf verschiedene Prüfzeichen: Ohne den Vergleich mit der Normalform des Suchtexts
// (SQLSuchtextIstISBN) fände die getippte zehnstellige den Titel nicht.
func TestSuche_FindetDenTitelUeberDieZehnstelligeISBN(t *testing.T) {
	pool := pgTestPool(t)
	resetInventurDaten(t, pool)
	ctx := context.Background()
	repo := NewBookRepository(pool)

	const titel = "Advanced Organic Chemistry"
	var id string
	if err := pool.QueryRow(ctx, `INSERT INTO buecher_titel (titel, isbn) VALUES ($1, '0306406152') RETURNING id::text`,
		titel).Scan(&id); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO buecher_exemplare (titel_id, barcode_id) VALUES ($1, 'B-ISBN-SUCHE-1')`, id); err != nil {
		t.Fatal(err)
	}

	for _, suchtext := range []string{"0306406152", "0-306-40615-2", "9780306406157", "978-0-306-40615-7"} {
		volltext, err := repo.SearchTitles(ctx, suchtext)
		if err != nil {
			t.Fatalf("SearchTitles(%q): %v", suchtext, err)
		}
		if len(volltext) != 1 || volltext[0].Titel != titel {
			t.Errorf("SearchTitles(%q): %d Treffer, erwartet den Titel %q", suchtext, len(volltext), titel)
		}
		tokenweise, gesamt, err := repo.SearchTitlesFuzzy(ctx, suchtext, 10)
		if err != nil {
			t.Fatalf("SearchTitlesFuzzy(%q): %v", suchtext, err)
		}
		if len(tokenweise) != 1 || gesamt != 1 || tokenweise[0].Titel != titel {
			t.Errorf("SearchTitlesFuzzy(%q): %d Treffer (gesamt %d), erwartet den Titel %q", suchtext, len(tokenweise), gesamt, titel)
		}
	}

	// Gegenprobe: Dieselben Ziffern mit falschem Prüfzeichen sind eine andere Nummer.
	if treffer, err := repo.SearchTitles(ctx, "0306406153"); err != nil || len(treffer) != 0 {
		t.Errorf("SearchTitles mit falschem Prüfzeichen: %d Treffer, %v — erwartet keinen", len(treffer), err)
	}
}
