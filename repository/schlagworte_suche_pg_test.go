package repository

import (
	"context"
	"slices"
	"strings"
	"testing"
)

// Suche über Schlagworte (docs/OFFEN.md 4.20). Zwei Formen derselben Regel: die Bedingung
// am Server (SQLTitelUeberSchlagwort — Titel-Verwaltung, öffentlicher Katalog, Portal) und
// die Wörter je Titel für die Suche im Browser (SuchwoerterDerTitel — Medienkatalog „Suche
// & Filter"). Beide müssen dieselben Titel finden, sonst fände der eine Reiter des
// Medienkatalogs etwas anderes als der andere. Und beide müssen den Verweis auflösen:
// „Tierfantasy" findet, was „Fantasy" trägt.
func TestSchlagwortSuche_ServerUndBrowserFindenDieselbenTitel(t *testing.T) {
	pool := pgTestPool(t)
	resetSchlagworte(t, pool)
	ctx := context.Background()

	drachen := seedSchlagwortTitel(t, pool, "Drachenreiter")
	krabat := seedSchlagwortTitel(t, pool, "Krabat")
	mathe := seedSchlagwortTitel(t, pool, "Mathematik heute 7")
	for id, woerter := range map[string][]string{drachen: {"Fantasy", "Drachen"}, krabat: {"Sage"}} {
		if _, err := SetzeSchlagworte(ctx, pool, id, woerter); err != nil {
			t.Fatalf("Schlagworte setzen: %v", err)
		}
	}
	var fantasy string
	if err := pool.QueryRow(ctx, `SELECT id FROM schlagworte WHERE wort = 'Fantasy'`).Scan(&fantasy); err != nil {
		t.Fatal(err)
	}
	if err := SetzeSchlagwortVerweis(ctx, pool, "Tierfantasy", fantasy); err != nil {
		t.Fatalf("Verweis anlegen: %v", err)
	}

	woerter, err := SuchwoerterDerTitel(ctx, pool, []string{drachen, krabat, mathe})
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"Drachen", "Fantasy", "Tierfantasy"}; !slices.Equal(woerter[drachen], want) {
		t.Errorf("Suchwörter Drachenreiter = %q, erwartet %q — der Verweis gehört zu den Wörtern seines Ziels", woerter[drachen], want)
	}
	if _, da := woerter[mathe]; da {
		t.Errorf("ein Titel ohne Schlagworte steht in der Antwort: %q", woerter[mathe])
	}

	amServer := func(muster string) []string {
		t.Helper()
		rows, err := pool.Query(ctx, `SELECT bt.titel FROM buecher_titel bt WHERE `+
			SQLTitelUeberSchlagwort("bt", "$1")+` ORDER BY bt.titel`, muster)
		if err != nil {
			t.Fatalf("Bedingung %q: %v", muster, err)
		}
		defer rows.Close()
		var titel []string
		for rows.Next() {
			var name string
			if err := rows.Scan(&name); err != nil {
				t.Fatal(err)
			}
			titel = append(titel, name)
		}
		if err := rows.Err(); err != nil {
			t.Fatal(err)
		}
		return titel
	}
	// Die Browser-Regel aus buecherSuchen (startseiten_api.js): ein Wort enthält den
	// Suchbegriff, ohne Rücksicht auf Groß- und Kleinschreibung.
	imBrowser := func(muster string) []string {
		var titel []string
		for _, zeile := range []struct{ id, name string }{ // in der Reihenfolge der Titel
			{drachen, "Drachenreiter"}, {krabat, "Krabat"}, {mathe, "Mathematik heute 7"},
		} {
			if slices.ContainsFunc(woerter[zeile.id], func(w string) bool {
				return strings.Contains(strings.ToLower(w), strings.ToLower(muster))
			}) {
				titel = append(titel, zeile.name)
			}
		}
		return titel
	}

	faelle := []struct {
		muster string
		titel  []string
	}{
		{"Fantasy", []string{"Drachenreiter"}},
		{"fanta", []string{"Drachenreiter"}},       // Teilstring, wie bei Titel und Autor
		{"TIERFANTASY", []string{"Drachenreiter"}}, // der Verweis findet sein Ziel
		{"sage", []string{"Krabat"}},
		{"a", []string{"Drachenreiter", "Krabat"}},
		{"Mathematik", nil}, // der Titel trägt kein Schlagwort; die Titelsuche ist nicht Sache dieser Regel
	}
	for _, f := range faelle {
		if got := amServer(f.muster); !slices.Equal(got, f.titel) {
			t.Errorf("am Server %q: %q, erwartet %q", f.muster, got, f.titel)
		}
		if got := imBrowser(f.muster); !slices.Equal(got, f.titel) {
			t.Errorf("im Browser %q: %q, erwartet %q", f.muster, got, f.titel)
		}
	}
}
