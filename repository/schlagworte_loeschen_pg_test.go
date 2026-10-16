package repository

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

// Mehrere Schlagworte auf einmal löschen (wie Littera „Datenbearbeitung"):
// alle oder keins, jeder Titel einmal gezählt, ein gewählter Verweis samt Ziel ist kein
// Fehler.
func TestSchlagwortPflege_MehrereLoeschen(t *testing.T) {
	pool := pgTestPool(t)
	resetSchlagworte(t, pool)
	ctx := context.Background()
	ids := pflegeStand(t, pool, map[string][]string{
		"Krabat":    {"Magie", "Mühle", "Freundschaft"},
		"Momo":      {"Zeit", "Freundschaft"},
		"Die Welle": {"Schule"},
	})
	if _, err := SetzeSchlagwortVerweis(ctx, pool, "Zauberei", ids["Magie"]); err != nil {
		t.Fatal(err)
	}
	if _, err := SetzeSchlagwortVerweis(ctx, pool, "Hexerei", ids["Magie"]); err != nil {
		t.Fatal(err)
	}
	zauberei := pflegeZeile(t, pool, "Zauberei").ID
	anzahl := func() int {
		t.Helper()
		var n int
		if err := pool.QueryRow(ctx, `SELECT count(*)::int FROM schlagworte`).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	vorher := anzahl() // 6 Wörter, 2 Verweise

	// Eine unbekannte Kennung in der Auswahl: nichts fällt, auch nicht die bekannten.
	unbekannt := "00000000-0000-0000-0000-000000000000"
	if _, err := LoescheSchlagworte(ctx, pool, []string{ids["Schule"], unbekannt}); !errors.Is(err, ErrSchlagwortNichtGefunden) {
		t.Errorf("mit unbekannter Kennung: %v, erwartet ErrSchlagwortNichtGefunden", err)
	}
	if got := anzahl(); got != vorher {
		t.Errorf("nach dem abgelehnten Löschen %d Zeilen, erwartet %d — alle oder keins", got, vorher)
	}
	if _, err := LoescheSchlagworte(ctx, pool, nil); !errors.Is(err, ErrSchlagwortUngueltig) {
		t.Errorf("leere Auswahl: %v, erwartet ErrSchlagwortUngueltig", err)
	}

	// Magie mit einem seiner Verweise gewählt, dazu Mühle (auch an Krabat) und Zeit (an Momo),
	// Mühle doppelt, einmal in Großbuchstaben — dieselbe Kennung, die Tür nimmt beide
	// Schreibweisen an: 4 Wörter; Krabat und Momo je einmal gezählt; Hexerei fällt mit,
	// Zauberei ist gewählt und zählt nicht als mitgefallen.
	geloescht, err := LoescheSchlagworte(ctx, pool, []string{
		ids["Magie"], zauberei, ids["Mühle"], ids["Zeit"], strings.ToUpper(ids["Mühle"]),
	})
	if err != nil {
		t.Fatalf("mehrere löschen: %v", err)
	}
	want := SchlagwortLoeschung{Woerter: 4, Titel: 2, Verweise: 1, Namen: []string{"Magie", "Mühle", "Zauberei", "Zeit"}}
	if !cmp.Equal(geloescht, want) {
		t.Errorf("mehrere löschen: %+v, erwartet %+v", geloescht, want)
	}
	if got := anzahl(); got != vorher-5 {
		t.Errorf("nach dem Löschen %d Zeilen, erwartet %d (4 gewählt, Hexerei mit)", got, vorher-5)
	}
	if got := woerterAm(t, pool, "Krabat"); len(got) != 1 || got[0] != "Freundschaft" {
		t.Errorf("Krabat nach dem Löschen: %q, erwartet [Freundschaft]", got)
	}
	if got := woerterAm(t, pool, "Die Welle"); len(got) != 1 || got[0] != "Schule" {
		t.Errorf("Die Welle: %q, erwartet [Schule] — nicht gewählt", got)
	}
}
