package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"testing"

	"bibliothek/db"
	"bibliothek/repository"
)

// Der öffentliche Katalog und „Mein Portal" suchen über dieselbe Tür,
// GET /api/public/opac/suche — seit dem 23.09.2026 auch über Schlagwort und Verweis
// (docs/OFFEN.md 4.20). Zwei Regeln der Tür müssen dabei halten: Was nicht öffentlich ist
// (hier ein Lernmittel), bleibt es, auch wenn sein Schlagwort passt. Und die LIKE-Joker
// bleiben maskiert — ein „%" darf nicht über die Schlagworte den ganzen Bestand liefern.
func TestOpacSuche_FindetUeberSchlagwortUndVerweis(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `TRUNCATE schlagworte CASCADE`); err != nil {
		t.Fatal(err)
	}

	mondflug := titelMitSignatur(t, pool, "Mondflug", "", 0)
	exemplar(t, pool, mondflug, "B-OPAC-1", true, "")
	physik := titelMitSignatur(t, pool, "LMF-Physik 9", "", 0) // Lernmittel: nie im öffentlichen Katalog
	exemplar(t, pool, physik, "B-OPAC-2", true, "")
	for _, id := range []string{mondflug, physik} {
		if _, err := repository.SetzeSchlagworte(ctx, pool, id, []string{"Weltraum"}); err != nil {
			t.Fatal(err)
		}
	}
	var weltraum string
	if err := pool.QueryRow(ctx, `SELECT id FROM schlagworte WHERE wort = 'Weltraum'`).Scan(&weltraum); err != nil {
		t.Fatal(err)
	}
	if err := repository.SetzeSchlagwortVerweis(ctx, pool, "Raumfahrt", weltraum); err != nil {
		t.Fatal(err)
	}

	s := &Server{DB: &db.Database{Pool: pool}}
	suche := func(q string) []string {
		t.Helper()
		rec := httptest.NewRecorder()
		s.PublicCatalogSearchHandler()(rec, httptest.NewRequest(http.MethodGet, "/api/public/opac/suche?q="+url.QueryEscape(q), nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("q=%q: %d %s", q, rec.Code, rec.Body.String())
		}
		var treffer []OpacTitel
		if err := json.Unmarshal(rec.Body.Bytes(), &treffer); err != nil {
			t.Fatal(err)
		}
		var titel []string
		for _, b := range treffer {
			titel = append(titel, b.Titel)
		}
		return titel
	}

	faelle := []struct {
		q     string
		titel []string
	}{
		{"Weltraum", []string{"Mondflug"}},
		{"raumf", []string{"Mondflug"}}, // der Verweis „Raumfahrt" findet sein Ziel, auch angefangen
		{"%", nil},                      // Joker maskiert: kein Treffer über die Schlagworte
	}
	for _, f := range faelle {
		if got := suche(f.q); !slices.Equal(got, f.titel) {
			t.Errorf("q=%q: %q, erwartet %q", f.q, got, f.titel)
		}
	}
}

// Der Filter in „Mein Portal" (docs/OFFEN.md 4.20): Die Liste nennt nur markierte Wörter,
// zu denen der öffentliche Katalog einen Titel zeigt — ein Filter, der nichts findet, wäre
// eine Sackgasse. Die gefilterte Suche geht ohne und mit Suchtext, und der Kopf
// X-Treffer-Gesamt sagt, wie viele Titel es sind, wenn die Antwort bei 50 abschneidet:
// Beim Stöbern über ein Thema sind mehr als 50 Titel der Normalfall.
func TestOpacFilter_ListeUndGefilterteSuche(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `TRUNCATE schlagworte CASCADE`); err != nil {
		t.Fatal(err)
	}

	titel := func(name string, woerter ...string) string {
		t.Helper()
		id := titelMitSignatur(t, pool, name, "", 0)
		exemplar(t, pool, id, "B-FILTER-"+name, true, "")
		if _, err := repository.SetzeSchlagworte(ctx, pool, id, woerter); err != nil {
			t.Fatal(err)
		}
		return id
	}
	titel("Mondflug", "Weltraum", "Abenteuer")
	titel("Krabat", "Sage")         // öffentlich, aber ohne „Weltraum": zählt beim Filter nicht mit
	titel("LMF-Physik 9", "Physik") // Lernmittel: im öffentlichen Katalog unsichtbar
	for i := range 51 {
		titel(fmt.Sprintf("Sternfahrt %02d", i), "Weltraum")
	}
	wortID := func(wort string) string {
		t.Helper()
		var id string
		if err := pool.QueryRow(ctx, `SELECT id FROM schlagworte WHERE wort = $1`, wort).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	for _, wort := range []string{"Weltraum", "Physik"} { // „Abenteuer" bleibt unmarkiert
		if err := repository.SetzeSchlagwortFilter(ctx, pool, wortID(wort), true); err != nil {
			t.Fatal(err)
		}
	}

	s := &Server{DB: &db.Database{Pool: pool}}

	// Die Liste: markiert UND mit öffentlichem Titel.
	rec := httptest.NewRecorder()
	s.PublicCatalogFilterHandler()(rec, httptest.NewRequest(http.MethodGet, "/api/public/opac/filter", nil))
	var filter []repository.SchlagwortFilter
	if err := json.Unmarshal(rec.Body.Bytes(), &filter); err != nil || rec.Code != http.StatusOK {
		t.Fatalf("Filterliste: %d %s (%v)", rec.Code, rec.Body.String(), err)
	}
	if len(filter) != 1 || filter[0].Wort != "Weltraum" || filter[0].ID != wortID("Weltraum") {
		t.Errorf("Filterliste %+v, erwartet nur „Weltraum“ — „Physik“ trägt nur ein Lernmittel, „Abenteuer“ ist nicht markiert", filter)
	}

	suche := func(query string) (int, []string, string) {
		t.Helper()
		rec := httptest.NewRecorder()
		s.PublicCatalogSearchHandler()(rec, httptest.NewRequest(http.MethodGet, "/api/public/opac/suche?"+query, nil))
		var treffer []OpacTitel
		if rec.Code == http.StatusOK {
			if err := json.Unmarshal(rec.Body.Bytes(), &treffer); err != nil {
				t.Fatal(err)
			}
		}
		var namen []string
		for _, b := range treffer {
			namen = append(namen, b.Titel)
		}
		return rec.Code, namen, rec.Header().Get("X-Treffer-Gesamt")
	}

	weltraum := url.QueryEscape(wortID("Weltraum"))
	if code, namen, gesamt := suche("schlagwort_id=" + weltraum); code != http.StatusOK || len(namen) != 50 || gesamt != "52" {
		t.Errorf("nur Filter: HTTP %d, %d Titel, X-Treffer-Gesamt %q — erwartet 50 gezeigt von 52", code, len(namen), gesamt)
	}
	if code, namen, gesamt := suche("q=mond&schlagwort_id=" + weltraum); code != http.StatusOK || !slices.Equal(namen, []string{"Mondflug"}) || gesamt != "1" {
		t.Errorf("Filter und Suchtext: HTTP %d, %q, X-Treffer-Gesamt %q", code, namen, gesamt)
	}
	if code, namen, _ := suche("q=mond&schlagwort_id=" + url.QueryEscape(wortID("Abenteuer"))); code != http.StatusOK || !slices.Equal(namen, []string{"Mondflug"}) {
		t.Errorf("Filter über ein unmarkiertes Wort: HTTP %d, %q", code, namen)
	}
	if code, namen, _ := suche("q=Sternfahrt&schlagwort_id=" + url.QueryEscape(wortID("Physik"))); code != http.StatusOK || len(namen) != 0 {
		t.Errorf("Filter ohne gemeinsamen Titel: HTTP %d, %q — erwartet nichts", code, namen)
	}
	if code, _, _ := suche("schlagwort_id=kein-wort"); code != http.StatusBadRequest {
		t.Errorf("kaputte Kennung: HTTP %d statt 400", code)
	}
}
