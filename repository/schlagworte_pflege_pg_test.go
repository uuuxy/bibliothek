package repository

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Pflege der Schlagworte (Migration 143, docs/OFFEN.md 4.20 Stufe 1) gegen echtes Postgres:
// Die Regeln der Verweise stehen doppelt — in repository/schlagworte_pflege.go und als
// Trigger —, und beides sieht nur ein echter Lauf.

// pflegeStand legt Titel mit Schlagworten an und liefert die Kennungen der Wörter.
func pflegeStand(t *testing.T, pool *pgxpool.Pool, titelWoerter map[string][]string) map[string]string {
	t.Helper()
	ctx := context.Background()
	for titel, woerter := range titelWoerter {
		if _, err := SetzeSchlagworte(ctx, pool, seedSchlagwortTitel(t, pool, titel), woerter); err != nil {
			t.Fatalf("%s: %v", titel, err)
		}
	}
	ids := map[string]string{}
	rows, err := pool.Query(ctx, `SELECT wort, id::text FROM schlagworte`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var w, id string
		if err := rows.Scan(&w, &id); err != nil {
			t.Fatal(err)
		}
		ids[w] = id
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return ids
}

func woerterAm(t *testing.T, pool *pgxpool.Pool, titel string) []string {
	t.Helper()
	var id string
	if err := pool.QueryRow(context.Background(), `SELECT id FROM buecher_titel WHERE titel = $1`, titel).Scan(&id); err != nil {
		t.Fatalf("Titel %q: %v", titel, err)
	}
	w, err := SchlagworteDesTitels(context.Background(), pool, id)
	if err != nil {
		t.Fatal(err)
	}
	return w
}

func pflegeZeile(t *testing.T, pool *pgxpool.Pool, wort string) SchlagwortPflegeZeile {
	t.Helper()
	liste, err := SchlagworteZurPflege(context.Background(), pool)
	if err != nil {
		t.Fatal(err)
	}
	for _, z := range liste.Zeilen {
		if z.Wort == wort {
			return z
		}
	}
	t.Fatalf("„%s“ fehlt in der Pflegeliste", wort)
	return SchlagwortPflegeZeile{}
}

func TestSchlagwortPflege_ZusammenfuehrenMachtDasAlteWortZumVerweis(t *testing.T) {
	pool := pgTestPool(t)
	resetSchlagworte(t, pool)
	ctx := context.Background()
	ids := pflegeStand(t, pool, map[string][]string{
		"Tintenherz":    {"Tierfantasy", "Abenteuer"},
		"Woodwalkers":   {"Tierfantasy", "Fantasy"},
		"Drachenreiter": {"Fantasy"},
	})
	if err := SetzeSchlagwortFilter(ctx, pool, ids["Tierfantasy"], true); err != nil {
		t.Fatal(err)
	}

	titel, err := FuehreSchlagworteZusammen(ctx, pool, ids["Tierfantasy"], ids["Fantasy"])
	if err != nil {
		t.Fatalf("zusammenführen: %v", err)
	}
	if titel != 2 {
		t.Errorf("gemeldet %d Titel, „Tierfantasy“ trug 2", titel)
	}
	// Woodwalkers trug beide — danach genau einmal „Fantasy".
	for titelName, want := range map[string][]string{
		"Tintenherz":  {"Abenteuer", "Fantasy"},
		"Woodwalkers": {"Fantasy"},
	} {
		if got := woerterAm(t, pool, titelName); !slices.Equal(got, want) {
			t.Errorf("%s: %q, want %q", titelName, got, want)
		}
	}
	alt := pflegeZeile(t, pool, "Tierfantasy")
	if alt.VerweisAuf != "Fantasy" || alt.Titel != 0 || alt.IstFilter {
		t.Errorf("„Tierfantasy“ nach dem Zusammenführen: %+v — erwartet Verweis auf Fantasy, 0 Titel, kein Filter", alt)
	}
	ziel := pflegeZeile(t, pool, "Fantasy")
	if ziel.Titel != 3 || !ziel.IstFilter || !slices.Equal(ziel.Verweise, []string{"Tierfantasy"}) {
		t.Errorf("„Fantasy“ nach dem Zusammenführen: %+v — erwartet 3 Titel, Filter übernommen, Verweis Tierfantasy", ziel)
	}

	// Wer den Verweis tippt, bekommt das Ziel.
	neu := seedSchlagwortTitel(t, pool, "Warrior Cats")
	got, err := SetzeSchlagworte(ctx, pool, neu, []string{"tierfantasy", "Fantasy"})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(got, []string{"Fantasy"}) {
		t.Errorf("getippter Verweis: %q, want [Fantasy]", got)
	}

	if _, err := FuehreSchlagworteZusammen(ctx, pool, ids["Fantasy"], ids["Tierfantasy"]); !errors.Is(err, ErrSchlagwortRegel) {
		t.Errorf("mit dem eigenen Verweis zusammenführen: %v, want ErrSchlagwortRegel", err)
	}
}

func TestSchlagwortPflege_UmbenennenLoeschenVerweisFilter(t *testing.T) {
	pool := pgTestPool(t)
	resetSchlagworte(t, pool)
	ctx := context.Background()
	ids := pflegeStand(t, pool, map[string][]string{
		"Die Welle": {"gewalt", "Schule"},
		"Krabat":    {"Magie"},
	})

	if neu, err := BenenneSchlagwortUm(ctx, pool, ids["gewalt"], "  Gewalt "); err != nil || neu != "Gewalt" {
		t.Errorf("Schreibweise ändern: %q, %v", neu, err)
	}
	if _, err := BenenneSchlagwortUm(ctx, pool, ids["Schule"], "magie"); !errors.Is(err, ErrSchlagwortGibtEs) {
		t.Errorf("umbenennen auf ein vorhandenes Wort: %v, want ErrSchlagwortGibtEs", err)
	}
	if _, err := BenenneSchlagwortUm(ctx, pool, "00000000-0000-0000-0000-000000000000", "X"); !errors.Is(err, ErrSchlagwortNichtGefunden) {
		t.Errorf("unbekanntes Wort: %v, want ErrSchlagwortNichtGefunden", err)
	}

	// Ein neuer Verweis entsteht ohne Titel; als Filter lässt er sich nicht markieren.
	if err := SetzeSchlagwortVerweis(ctx, pool, "Zauberei", ids["Magie"]); err != nil {
		t.Fatalf("Verweis anlegen: %v", err)
	}
	verweis := pflegeZeile(t, pool, "Zauberei")
	if verweis.VerweisAuf != "Magie" || verweis.Titel != 0 {
		t.Errorf("Verweis: %+v", verweis)
	}
	if err := SetzeSchlagwortFilter(ctx, pool, verweis.ID, true); !errors.Is(err, ErrSchlagwortRegel) {
		t.Errorf("Verweis als Filter: %v, want ErrSchlagwortRegel", err)
	}

	// Verweis auf eine Schreibweise, die Titel trägt: das ist ein Zusammenführen.
	if err := SetzeSchlagwortVerweis(ctx, pool, "Schule", ids["gewalt"]); err != nil {
		t.Fatalf("Verweis auf vorhandenes Wort: %v", err)
	}
	if got := woerterAm(t, pool, "Die Welle"); !slices.Equal(got, []string{"Gewalt"}) {
		t.Errorf("Die Welle nach dem Verweis: %q, want [Gewalt]", got)
	}

	// Löschen: Titel verlieren das Wort, Verweise darauf fallen mit.
	titel, verweise, err := LoescheSchlagwort(ctx, pool, ids["Magie"])
	if err != nil || titel != 1 || verweise != 1 {
		t.Errorf("löschen: titel=%d verweise=%d err=%v, want 1 und 1", titel, verweise, err)
	}
	if got := woerterAm(t, pool, "Krabat"); len(got) != 0 {
		t.Errorf("Krabat nach dem Löschen: %q", got)
	}
	var rest int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM schlagworte WHERE wort IN ('Magie', 'Zauberei')`).Scan(&rest); err != nil {
		t.Fatal(err)
	}
	if rest != 0 {
		t.Errorf("%d Zeilen von Magie/Zauberei übrig, want 0", rest)
	}
}

// Die Regeln stehen auch in der Datenbank — eine Tür, die an der Pflege vorbei schreibt,
// kommt nicht durch.
func TestSchlagwortPflege_DatenbankHaeltDieRegeln(t *testing.T) {
	pool := pgTestPool(t)
	resetSchlagworte(t, pool)
	ctx := context.Background()
	ids := pflegeStand(t, pool, map[string][]string{"Momo": {"Zeit", "Freundschaft"}})
	if err := SetzeSchlagwortVerweis(ctx, pool, "Uhren", ids["Zeit"]); err != nil {
		t.Fatal(err)
	}
	uhren := pflegeZeile(t, pool, "Uhren").ID
	var momo string
	if err := pool.QueryRow(ctx, `SELECT id FROM buecher_titel WHERE titel = 'Momo'`).Scan(&momo); err != nil {
		t.Fatal(err)
	}

	for name, sql := range map[string]string{
		"Titel am Verweis":             `INSERT INTO titel_schlagworte (titel_id, schlagwort_id) VALUES ('` + momo + `', '` + uhren + `')`,
		"Kette":                        `UPDATE schlagworte SET verweis_auf = '` + uhren + `' WHERE id = '` + ids["Freundschaft"] + `'`,
		"Wort mit Titeln wird Verweis": `UPDATE schlagworte SET verweis_auf = '` + ids["Zeit"] + `' WHERE id = '` + ids["Freundschaft"] + `'`,
		"Verweis auf sich selbst":      `UPDATE schlagworte SET verweis_auf = id WHERE id = '` + ids["Zeit"] + `'`,
		"Verweis als Filter":           `UPDATE schlagworte SET ist_filter = true WHERE id = '` + uhren + `'`,
	} {
		if _, err := pool.Exec(ctx, sql); err == nil {
			t.Errorf("%s: die Datenbank hat es angenommen", name)
		}
	}
}
