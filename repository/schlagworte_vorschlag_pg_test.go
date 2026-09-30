package repository

import (
	"context"
	"slices"
	"testing"
)

// Der Schlagwort-Vorschlag beim Bestellen per ISBN (docs/OFFEN.md 4.20): Aus bis zu 98
// Verlagswörtern eines DNB-Satzes bleibt nur, was es in der eigenen Liste gibt — als Wort
// mit Titeln oder als Verweis darauf. Der Vergleich ist das ganze Wort, nicht ein Teil
// davon, und ein Wort ohne Titel wird nicht vorgeschlagen, auch nicht über einen Verweis.
func TestSchlagworteAusStichwoertern_NurWasDieEigeneListeKennt(t *testing.T) {
	pool := pgTestPool(t)
	resetSchlagworte(t, pool)
	ctx := context.Background()

	drachen := seedSchlagwortTitel(t, pool, "Drachenreiter")
	boie := seedSchlagwortTitel(t, pool, "Dunkelnacht")
	if _, err := SetzeSchlagworte(ctx, pool, drachen, []string{"Fantasy", "Science-Fiction"}); err != nil {
		t.Fatal(err)
	}
	if _, err := SetzeSchlagworte(ctx, pool, boie, []string{"Krieg", "Erste Liebe"}); err != nil {
		t.Fatal(err)
	}
	// Wörter ohne Titel: ein Rest ohne Bedeutung und das Ziel eines Verweises.
	if _, err := pool.Exec(ctx, `INSERT INTO schlagworte (wort) VALUES ('Burgen'), ('Weltraum')`); err != nil {
		t.Fatal(err)
	}
	wortID := func(wort string) string {
		t.Helper()
		var id string
		if err := pool.QueryRow(ctx, `SELECT id FROM schlagworte WHERE wort = $1`, wort).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	if err := SetzeSchlagwortVerweis(ctx, pool, "Science Fiction", wortID("Science-Fiction")); err != nil {
		t.Fatal(err)
	}
	if err := SetzeSchlagwortVerweis(ctx, pool, "Weltall", wortID("Weltraum")); err != nil {
		t.Fatal(err)
	}

	got, err := SchlagworteAusStichwoertern(ctx, pool, []string{
		"Jugendbuch", "Burgen", "  erste   liebe ", "KRIEG",
		"Science Fiction", "Weltall", "TikTok", "Fantasy", "fantasy",
	})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"Erste Liebe", "Fantasy", "Krieg", "Science-Fiction"}
	if !slices.Equal(got, want) {
		t.Errorf("Vorschlag %q, erwartet %q", got, want)
	}

	// Das ganze Wort, kein Teil davon: „Kriegsende" ist nicht „Krieg".
	if teil, err := SchlagworteAusStichwoertern(ctx, pool, []string{"Kriegsende"}); err != nil || len(teil) != 0 {
		t.Errorf("„Kriegsende“: %q, %v — erwartet nichts", teil, err)
	}

	leer, err := SchlagworteAusStichwoertern(ctx, pool, nil)
	if err != nil || leer == nil || len(leer) != 0 {
		t.Errorf("ohne Stichwörter: %q, %v — erwartet eine leere Liste", leer, err)
	}
}

// Die DNB liefert Umlaute zerlegt („o" + U+0308, gemessen am 30.09.2026). Das zerlegte „Vögel"
// traf das „Vögel" der Liste nie: Für lower() und den eindeutigen Index sind es zwei Wörter.
// Beide Wege nehmen dieselbe Normalform (schlagwortNormalform): der Vorschlag und das
// Speichern — ein zerlegt eingetragenes Wort landet beim vorhandenen, nicht daneben.
func TestSchlagworte_ZerlegteUmlauteSindDasselbeWort(t *testing.T) {
	pool := pgTestPool(t)
	resetSchlagworte(t, pool)
	ctx := context.Background()

	const zusammengesetzt, zerlegt = "Vögel", "Vo\u0308gel"
	amsel := seedSchlagwortTitel(t, pool, "Die Amsel")
	if _, err := SetzeSchlagworte(ctx, pool, amsel, []string{zusammengesetzt}); err != nil {
		t.Fatal(err)
	}

	got, err := SchlagworteAusStichwoertern(ctx, pool, []string{zerlegt, "  gar\u0308ten "})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(got, []string{zusammengesetzt}) {
		t.Errorf("Vorschlag %+q, erwartet %+q", got, []string{zusammengesetzt})
	}

	meise := seedSchlagwortTitel(t, pool, "Die Meise")
	if _, err := SetzeSchlagworte(ctx, pool, meise, []string{zerlegt}); err != nil {
		t.Fatal(err)
	}
	var woerter []string
	rows, err := pool.Query(ctx, `SELECT wort FROM schlagworte ORDER BY wort`)
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var w string
		if err := rows.Scan(&w); err != nil {
			t.Fatal(err)
		}
		woerter = append(woerter, w)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(woerter, []string{zusammengesetzt}) {
		t.Errorf("Liste nach dem zweiten Titel: %+q — erwartet ein Wort, zusammengesetzt", woerter)
	}
}
