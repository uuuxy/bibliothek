package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"testing"

	"bibliothek/db"
	"bibliothek/internal/pgtest"
	"bibliothek/internal/testdaten"
	"bibliothek/repository"
)

// Der öffentliche Katalog findet einen Titel auch über sein Schlagwort und über einen
// Verweis darauf. Zwei Regeln der Tür müssen dabei halten: Was nicht öffentlich ist (hier
// ein Lernmittel), bleibt es, auch wenn sein Schlagwort passt. Und die LIKE-Joker bleiben
// maskiert — ein „%" darf nicht über die Schlagworte den ganzen Bestand liefern.
func TestOpacSuche_FindetUeberSchlagwortUndVerweis(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.ResetBestandsdaten(t, pool)
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `TRUNCATE schlagworte CASCADE`); err != nil {
		t.Fatal(err)
	}

	mondflug := testdaten.TitelMitSignatur(t, pool, "Mondflug", "", 0)
	testdaten.Exemplar(t, pool, mondflug, "B-OPAC-1", true, "")
	physik := testdaten.TitelMitSignatur(t, pool, "LMF-Physik 9", "", 0) // Lernmittel: nie im öffentlichen Katalog
	testdaten.Exemplar(t, pool, physik, "B-OPAC-2", true, "")
	for _, id := range []string{mondflug, physik} {
		if _, err := repository.SetzeSchlagworte(ctx, pool, id, []string{"Weltraum"}); err != nil {
			t.Fatal(err)
		}
	}
	var weltraum string
	if err := pool.QueryRow(ctx, `SELECT id FROM schlagworte WHERE wort = 'Weltraum'`).Scan(&weltraum); err != nil {
		t.Fatal(err)
	}
	if _, err := repository.SetzeSchlagwortVerweis(ctx, pool, "Raumfahrt", weltraum); err != nil {
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
