package repository

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Umbenennen und Zusammenführen mit und ohne Verweis von der alten Schreibweise (entschieden
// am 23.09.2026, docs/OFFEN.md 4.20; Rasterdurchgang V2). Vorher ließ das Zusammenführen das
// alte Wort immer als Verweis stehen, das Umbenennen nie; jetzt entscheidet die Pflegeseite
// je Aktion, vorbelegt mit „behalten".

func titelIDVon(t *testing.T, pool *pgxpool.Pool, titel string) string {
	t.Helper()
	var id string
	if err := pool.QueryRow(context.Background(), `SELECT id FROM buecher_titel WHERE titel = $1`, titel).Scan(&id); err != nil {
		t.Fatalf("Titel %q: %v", titel, err)
	}
	return id
}

func schreibweisenGibtEs(t *testing.T, pool *pgxpool.Pool, wort string) int {
	t.Helper()
	var n int
	if err := pool.QueryRow(context.Background(),
		`SELECT count(*)::int FROM schlagworte WHERE lower(wort) = lower($1)`, wort).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

// Der Fall, der den Verweis verlangt: Eine Maske war beim Umbenennen offen. Buchformular und
// Bestellkorb schicken beim Speichern die Menge zurück, die sie beim Öffnen gelesen haben —
// ohne Verweis entstünde „Krimi" neu, und der Titel verlöre „Kriminalroman".
func TestSchlagwortPflege_UmbenennenLaesstDieAlteSchreibweiseAlsVerweis(t *testing.T) {
	pool := pgTestPool(t)
	resetSchlagworte(t, pool)
	ctx := context.Background()
	ids := pflegeStand(t, pool, map[string][]string{"Emil": {"Krimi", "Berlin"}})
	gelesen := woerterAm(t, pool, "Emil") // die Maske öffnet: [Berlin Krimi]

	neu, verweis, err := BenenneSchlagwortUm(ctx, pool, ids["Krimi"], "Kriminalroman", true)
	if err != nil || neu != "Kriminalroman" {
		t.Fatalf("umbenennen: %q, %v — want Kriminalroman", neu, err)
	}
	if !verweis {
		t.Error("umbenennen meldet keinen Verweis — want die alte Schreibweise als Verweis")
	}
	if _, err := SetzeSchlagworte(ctx, pool, titelIDVon(t, pool, "Emil"), gelesen); err != nil {
		t.Fatal(err)
	}
	if got := woerterAm(t, pool, "Emil"); !slices.Equal(got, []string{"Berlin", "Kriminalroman"}) {
		t.Errorf("nach dem Speichern der offenen Maske trägt „Emil“ %v, want [Berlin Kriminalroman]", got)
	}
	if z := pflegeZeile(t, pool, "Krimi"); z.VerweisAufID != ids["Krimi"] || z.VerweisAuf != "Kriminalroman" {
		t.Errorf("„Krimi“ nach dem Umbenennen: %+v — erwartet Verweis auf dasselbe Wort, jetzt Kriminalroman", z)
	}
	liste, err := SchlagworteZurPflege(ctx, pool)
	if err != nil {
		t.Fatal(err)
	}
	if liste.Gesamt != 3 || liste.Verweise != 1 {
		t.Errorf("Pflegeliste zählt gesamt=%d verweise=%d, want 3 und 1 (Berlin, Kriminalroman, Verweis Krimi)",
			liste.Gesamt, liste.Verweise)
	}
}

// Ohne Verweis, wie in Littera: Die alte Schreibweise ist danach weg.
func TestSchlagwortPflege_UmbenennenOhneVerweis(t *testing.T) {
	pool := pgTestPool(t)
	resetSchlagworte(t, pool)
	ctx := context.Background()
	ids := pflegeStand(t, pool, map[string][]string{"Emil": {"Krimi"}})

	neu, verweis, err := BenenneSchlagwortUm(ctx, pool, ids["Krimi"], "Kriminalroman", false)
	if err != nil || neu != "Kriminalroman" || verweis {
		t.Fatalf("umbenennen: %q, verweis=%v, %v — want Kriminalroman ohne Verweis", neu, verweis, err)
	}
	if n := schreibweisenGibtEs(t, pool, "Krimi"); n != 0 {
		t.Errorf("%d Zeile(n) „Krimi“ nach dem Umbenennen ohne Verweis, want 0", n)
	}
	if got := woerterAm(t, pool, "Emil"); !slices.Equal(got, []string{"Kriminalroman"}) {
		t.Errorf("„Emil“ trägt %v, want [Kriminalroman]", got)
	}
}

// „Krimi" mit dem Verweis „Kriminalroman" wird „Kriminalroman": Die beiden tauschen. Vorher
// lehnte die Tür ab und riet zum Zusammenführen, und das scheiterte am eigenen Verweis.
// Danach lässt sich der neue Verweis selbst umbenennen, aber nur ohne weiteren Verweis.
func TestSchlagwortPflege_UmbenennenAufDenEigenenVerweisTauscht(t *testing.T) {
	pool := pgTestPool(t)
	resetSchlagworte(t, pool)
	ctx := context.Background()
	ids := pflegeStand(t, pool, map[string][]string{"Emil": {"Krimi"}})
	if err := SetzeSchlagwortVerweis(ctx, pool, "Kriminalroman", ids["Krimi"]); err != nil {
		t.Fatal(err)
	}

	neu, verweis, err := BenenneSchlagwortUm(ctx, pool, ids["Krimi"], "kriminalroman", true)
	if err != nil || neu != "kriminalroman" || !verweis {
		t.Fatalf("umbenennen auf den eigenen Verweis: %q, verweis=%v, %v — want kriminalroman mit Verweis", neu, verweis, err)
	}
	wort := pflegeZeile(t, pool, "kriminalroman")
	if wort.ID != ids["Krimi"] || wort.VerweisAufID != "" || !slices.Equal(wort.Verweise, []string{"Krimi"}) || wort.Titel != 1 {
		t.Errorf("nach dem Tausch: %+v — erwartet dasselbe Wort mit 1 Titel und dem Verweis Krimi", wort)
	}
	if n := schreibweisenGibtEs(t, pool, "Kriminalroman"); n != 1 {
		t.Errorf("%d Zeilen „kriminalroman“, want 1 — der alte Verweis geht im Wort auf", n)
	}

	krimi := pflegeZeile(t, pool, "Krimi")
	// Die Datenbank lehnte das auch ab (keine Kette), aber mit einem Satz, der der Pflegeseite
	// nicht sagt, was stattdessen geht — deshalb prüft die Tür vorher und nennt den Weg.
	if _, _, err := BenenneSchlagwortUm(ctx, pool, krimi.ID, "Krimis", true); !errors.Is(err, ErrSchlagwortRegel) ||
		!strings.Contains(err.Error(), "Verweis anlegen") {
		t.Errorf("Verweis umbenennen und behalten: %v, want ErrSchlagwortRegel mit dem Weg „Verweis anlegen“", err)
	}
	if neu, verweis, err := BenenneSchlagwortUm(ctx, pool, krimi.ID, "Krimis", false); err != nil || neu != "Krimis" || verweis {
		t.Errorf("Verweis umbenennen: %q, verweis=%v, %v — want Krimis ohne weiteren Verweis", neu, verweis, err)
	}
	if z := pflegeZeile(t, pool, "Krimis"); z.VerweisAufID != ids["Krimi"] {
		t.Errorf("„Krimis“: %+v — erwartet Verweis auf kriminalroman", z)
	}
	if n := schreibweisenGibtEs(t, pool, "Krimi"); n != 0 {
		t.Errorf("%d Zeilen „Krimi“ nach dem Umbenennen des Verweises, want 0", n)
	}
}

// Zusammenführen ohne Verweis: Das alte Wort fällt weg; seine Titel, Verweise und die
// Filter-Markierung gehen auf das Ziel über wie beim Zusammenführen mit Verweis.
func TestSchlagwortPflege_ZusammenfuehrenOhneVerweis(t *testing.T) {
	pool := pgTestPool(t)
	resetSchlagworte(t, pool)
	ctx := context.Background()
	ids := pflegeStand(t, pool, map[string][]string{
		"Tintenherz":  {"Tierfantasy", "Abenteuer"},
		"Woodwalkers": {"Tierfantasy", "Fantasy"},
	})
	if err := SetzeSchlagwortVerweis(ctx, pool, "Tierfantasie", ids["Tierfantasy"]); err != nil {
		t.Fatal(err)
	}
	if err := SetzeSchlagwortFilter(ctx, pool, ids["Tierfantasy"], true); err != nil {
		t.Fatal(err)
	}

	titel, err := FuehreSchlagworteZusammen(ctx, pool, ids["Tierfantasy"], ids["Fantasy"], false)
	if err != nil || titel != 2 {
		t.Fatalf("zusammenführen ohne Verweis: titel=%d, %v — want 2", titel, err)
	}
	if n := schreibweisenGibtEs(t, pool, "Tierfantasy"); n != 0 {
		t.Errorf("%d Zeile(n) „Tierfantasy“ nach dem Zusammenführen ohne Verweis, want 0", n)
	}
	for titelName, want := range map[string][]string{
		"Tintenherz":  {"Abenteuer", "Fantasy"},
		"Woodwalkers": {"Fantasy"},
	} {
		if got := woerterAm(t, pool, titelName); !slices.Equal(got, want) {
			t.Errorf("%s: %q, want %q", titelName, got, want)
		}
	}
	ziel := pflegeZeile(t, pool, "Fantasy")
	if ziel.Titel != 2 || !ziel.IstFilter || !slices.Equal(ziel.Verweise, []string{"Tierfantasie"}) {
		t.Errorf("„Fantasy“: %+v — erwartet 2 Titel, Filter übernommen, Verweis Tierfantasie umgehängt", ziel)
	}
}
