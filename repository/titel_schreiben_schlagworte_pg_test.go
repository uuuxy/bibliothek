package repository

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"testing"
)

// Das Katalogisat bringt die Littera-Schlagworte mit (docs/OFFEN.md 4.20, Stufe 3, entschieden am
// 30.09.2026) — aber nur an Titel, die noch keine tragen: Ein erneuter Import überschreibt keine
// Pflege, wie beim Fach. Über SetzeSchlagworte, aufbereitet wie in der Übernahme aus der Sicherung.
func TestSchreibeTitel_SchlagworteNurAnTitelOhne(t *testing.T) {
	pool := pgTestPool(t)
	resetSchlagworte(t, pool)
	ctx := context.Background()
	repo := NewKatalogAbfragen(pool)

	// Vorher: ein gepflegter Titel mit eigenem Schlagwort und einer ohne.
	gepflegt := seedSchlagwortTitel(t, pool, "Die Republik von Weimar")
	if _, err := SetzeSchlagworte(ctx, pool, gepflegt, []string{"Weimar (gepflegt)"}); err != nil {
		t.Fatal(err)
	}
	ohne := seedSchlagwortTitel(t, pool, "Krabat")

	viele := make([]string, SchlagworteJeTitelMax+1)
	for i := range viele {
		viele[i] = fmt.Sprintf("Pflanze %03d", i)
	}
	if _, err := repo.SchreibeTitel(ctx, []Titel{
		{Titel: "Die Republik von Weimar", Schlagworte: []string{"Weimarer Republik", "Deutsche Geschichte"}},
		{Titel: "Krabat", Schlagworte: []string{"Schwarze Magie", "schwarze magie", "  ", "Lehrling"}},
		{Titel: "Tintenherz", ISBN: "9783791504650", Schlagworte: []string{"Abenteuer", "Italien"}},
		{Titel: "Pflanzen und Umwelt", Schlagworte: viele},
		// Derselbe Titel ein zweites Mal in der Datei: Der Upsert nimmt den ersten, die
		// Schlagworte auch.
		{Titel: "Tintenherz", ISBN: "9783791504650", Schlagworte: []string{"Zweiter Datensatz"}},
	}); err != nil {
		t.Fatal(err)
	}

	id := func(titel string) string {
		var id string
		if err := pool.QueryRow(ctx, `SELECT id FROM buecher_titel WHERE titel = $1`, titel).Scan(&id); err != nil {
			t.Fatalf("Titel %q: %v", titel, err)
		}
		return id
	}
	for titelID, will := range map[string][]string{
		gepflegt:         {"Weimar (gepflegt)"},
		ohne:             {"Lehrling", "Schwarze Magie"},
		id("Tintenherz"): {"Abenteuer", "Italien"},
	} {
		got, err := LadeSchlagworteDesTitels(ctx, pool, titelID)
		if err != nil {
			t.Fatal(err)
		}
		if !slices.Equal(got, will) {
			t.Errorf("Titel %s trägt %q, erwartet %q", titelID, got, will)
		}
	}
	pflanzen, err := LadeSchlagworteDesTitels(ctx, pool, id("Pflanzen und Umwelt"))
	if err != nil {
		t.Fatal(err)
	}
	if len(pflanzen) != SchlagworteJeTitelMax {
		t.Errorf("„Pflanzen und Umwelt“ trägt %d Schlagworte, erwartet die Grenze %d", len(pflanzen), SchlagworteJeTitelMax)
	}

	// Ein zweiter Lauf derselben Datei ändert nichts mehr: Jetzt tragen alle welche.
	if _, err := repo.SchreibeTitel(ctx, []Titel{
		{Titel: "Krabat", Schlagworte: []string{"Anderes Wort"}},
	}); err != nil {
		t.Fatal(err)
	}
	got, err := LadeSchlagworteDesTitels(ctx, pool, ohne)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(got, []string{"Lehrling", "Schwarze Magie"}) {
		t.Errorf("nach dem zweiten Lauf trägt „Krabat“ %q — ein Import darf vorhandene Schlagworte nicht ersetzen", got)
	}
}

func TestSchlagworteAusFremddaten_WortFuerWort(t *testing.T) {
	woerter, auf := SchlagworteAusFremddaten([]string{
		" Weimarer   Republik ", "", "weimarer republik", strings.Repeat("x", SchlagwortMaxZeichen+1), "Geschichte",
	})
	if !slices.Equal(woerter, []string{"Weimarer Republik", "Geschichte"}) {
		t.Errorf("Wörter %q", woerter)
	}
	if auf.Leer != 1 || auf.ZuLang != 1 || auf.Gekuerzt {
		t.Errorf("Aufbereitung %+v, erwartet 1 leer, 1 zu lang, nicht gekürzt", auf)
	}
}

// Im Katalogisat vom Juni 2026 steht mancher Titel zweimal, der erste Eintrag ohne Schlagworte,
// der zweite mit. Der Upsert übergeht den zweiten als Dublette; die Schlagworte müssen trotzdem
// schon mit dem ersten Lauf kommen, und ein zweiter Lauf derselben Datei ändert nichts mehr.
// Gemessen am 30.09.2026: Mit nur den eingereihten Datensätzen brachte erst der zweite Lauf 253
// Zuordnungen.
func TestSchreibeTitel_SchlagworteAuchAusDerDublette(t *testing.T) {
	pool := pgTestPool(t)
	resetSchlagworte(t, pool)
	ctx := context.Background()
	repo := NewKatalogAbfragen(pool)

	datei := []Titel{
		{Titel: "Momo"},
		{Titel: "Momo", ISBN: "9783522202107", Schlagworte: []string{"Zeit", "Freundschaft"}},
	}
	stand := func() string {
		var s string
		if err := pool.QueryRow(ctx, `
			SELECT (SELECT count(*) FROM buecher_titel) || ' ' || (SELECT count(*) FROM titel_schlagworte) || ' ' ||
			       (SELECT coalesce(string_agg(s.wort, ',' ORDER BY s.wort), '') FROM titel_schlagworte ts
			        JOIN schlagworte s ON s.id = ts.schlagwort_id)`).Scan(&s); err != nil {
			t.Fatal(err)
		}
		return s
	}
	if _, err := repo.SchreibeTitel(ctx, datei); err != nil {
		t.Fatal(err)
	}
	nachEins := stand()
	if nachEins != "1 2 Freundschaft,Zeit" {
		t.Errorf("nach dem ersten Lauf: %q, erwartet 1 Titel mit Freundschaft und Zeit", nachEins)
	}
	if _, err := repo.SchreibeTitel(ctx, datei); err != nil {
		t.Fatal(err)
	}
	if nachZwei := stand(); nachZwei != nachEins {
		t.Errorf("der zweite Lauf hat etwas geändert: %q, vorher %q", nachZwei, nachEins)
	}
}

// Zwei Auflagen mit demselben Titeltext und verschiedener ISBN: Jede bekommt die Schlagworte
// ihres Datensatzes. Die Zuordnung der Schlagworte fragt den Bestand wie der Upsert davor,
// zuerst über die ISBN; nur über den Titeltext träfen beide Datensätze dieselbe Auflage, und
// die andere bliebe ohne.
func TestSchreibeTitel_SchlagworteJeAuflageUeberDieISBN(t *testing.T) {
	pool := pgTestPool(t)
	resetSchlagworte(t, pool)
	ctx := context.Background()
	const titel, isbnA, isbnB = "Schlagwort-Probe zweier Auflagen", "9780000562104", "9780000562111"
	raeume := func() {
		if _, err := pool.Exec(context.Background(), `DELETE FROM buecher_titel WHERE isbn IN ($1, $2)`, isbnA, isbnB); err != nil {
			t.Errorf("aufräumen: %v", err)
		}
	}
	raeume()
	t.Cleanup(raeume)
	lege := func(isbn string) string {
		var id string
		if err := pool.QueryRow(ctx, `INSERT INTO buecher_titel (titel, isbn) VALUES ($1, $2) RETURNING id`, titel, isbn).Scan(&id); err != nil {
			t.Fatalf("Auflage %s anlegen: %v", isbn, err)
		}
		return id
	}
	idA, idB := lege(isbnA), lege(isbnB)

	if _, err := NewKatalogAbfragen(pool).SchreibeTitel(ctx, []Titel{
		{Titel: titel, ISBN: isbnA, Schlagworte: []string{"Auflage A"}},
		{Titel: titel, ISBN: isbnB, Schlagworte: []string{"Auflage B"}},
	}); err != nil {
		t.Fatal(err)
	}
	for id, will := range map[string][]string{idA: {"Auflage A"}, idB: {"Auflage B"}} {
		got, err := LadeSchlagworteDesTitels(ctx, pool, id)
		if err != nil {
			t.Fatal(err)
		}
		if !slices.Equal(got, will) {
			t.Errorf("Auflage %s trägt %q, erwartet %q", id, got, will)
		}
	}
}
