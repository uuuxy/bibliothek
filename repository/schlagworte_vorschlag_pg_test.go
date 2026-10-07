package repository

import (
	"context"
	"slices"
	"strings"
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
	if _, err := SetzeSchlagwortVerweis(ctx, pool, "Science Fiction", wortID("Science-Fiction")); err != nil {
		t.Fatal(err)
	}
	if _, err := SetzeSchlagwortVerweis(ctx, pool, "Weltall", wortID("Weltraum")); err != nil {
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

// Die Normdatei-Wörter eines DNB-Satzes (docs/OFFEN.md 4.25, entschieden am 30.09.2026): Was die
// Liste kennt — in einer der Formen, als Wort mit Titeln oder als Verweis —, kommt als
// vorhanden, aufgelöst zum Ziel. Alles andere kommt als neu, in der Anzeige-Form; ein Wort
// ohne Titel zählt als neu, ein zu langes Wort wird nicht angeboten, Doppelte fallen.
func TestSchlagworteAusNormdaten_VorhandenUndNeu(t *testing.T) {
	pool := pgTestPool(t)
	resetSchlagworte(t, pool)
	ctx := context.Background()

	prozess := seedSchlagwortTitel(t, pool, "Der Process")
	if _, err := SetzeSchlagworte(ctx, pool, prozess, []string{"Kafka <Franz>", "Zweiter Weltkrieg"}); err != nil {
		t.Fatal(err)
	}
	var weltkrieg string
	if err := pool.QueryRow(ctx, `SELECT id FROM schlagworte WHERE wort = 'Zweiter Weltkrieg'`).Scan(&weltkrieg); err != nil {
		t.Fatal(err)
	}
	if _, err := SetzeSchlagwortVerweis(ctx, pool, "Weltkrieg <1939-1945>", weltkrieg); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO schlagworte (wort) VALUES ('Judo')`); err != nil { // ohne Titel
		t.Fatal(err)
	}

	begriff := func(anzeige string, formen ...string) Normdatenbegriff {
		return Normdatenbegriff{Anzeige: anzeige, Formen: append([]string{anzeige}, formen...)}
	}
	vorhanden, neu, err := SchlagworteAusNormdaten(ctx, pool, []Normdatenbegriff{
		begriff("Kafka <Franz>", "Kafka, Franz"),
		begriff("weltkrieg <1939-1945>"), // der Verweis, klein geschrieben
		begriff("Judo"),
		begriff("Schulstress"),
		begriff("schulstress"),
		begriff(strings.Repeat("x", SchlagwortMaxZeichen+1)),
		begriff("Oslo"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"Kafka <Franz>", "Zweiter Weltkrieg"}; !slices.Equal(vorhanden, want) {
		t.Errorf("vorhanden %q, erwartet %q", vorhanden, want)
	}
	if want := []string{"Judo", "Oslo", "Schulstress"}; !slices.Equal(neu, want) {
		t.Errorf("neu %q, erwartet %q", neu, want)
	}

	// Ein Begriff, dessen Anzeige ein Wort nennt, das über einen anderen Begriff schon unter
	// den vorhandenen steht, wird nicht zusätzlich als neu angeboten.
	doppeltV, doppeltN, err := SchlagworteAusNormdaten(ctx, pool, []Normdatenbegriff{
		begriff("weltkrieg <1939-1945>"),
		{Anzeige: "Zweiter Weltkrieg", Formen: []string{"Weltkrieg II"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(doppeltV, []string{"Zweiter Weltkrieg"}) || len(doppeltN) != 0 {
		t.Errorf("vorhanden %q, neu %q — erwartet nur „Zweiter Weltkrieg“ unter den vorhandenen", doppeltV, doppeltN)
	}

	leerV, leerN, err := SchlagworteAusNormdaten(ctx, pool, nil)
	if err != nil || leerV == nil || leerN == nil || len(leerV)+len(leerN) != 0 {
		t.Errorf("ohne Begriffe: %q, %q, %v — erwartet zwei leere Listen", leerV, leerN, err)
	}
}
