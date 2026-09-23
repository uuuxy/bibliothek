package inventur

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	"bibliothek/internal/pgtest"
	"bibliothek/repository"
)

// GET /api/books findet über Schlagwort und Verweis (docs/OFFEN.md 4.20) — in beiden
// Reitern des Medienkatalogs. Die Titel-Verwaltung schickt den Suchtext mit (?q=) und
// bekommt die Treffer vom Server; „Suche & Filter" lädt die ganze Liste und sucht im
// Browser, braucht also die Wörter je Titel in der Antwort. Fehlt eins von beiden, findet
// ein Reiter über „Raumfahrt" etwas und der andere nicht.
func TestKatalogliste_FindetUeberSchlagwortUndVerweis(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	handler := &APIHandler{repo: &BookRepository{db: pool}}

	titel := func(name, barcode string) string {
		t.Helper()
		var id string
		if err := pool.QueryRow(ctx, `INSERT INTO buecher_titel (titel) VALUES ($1) RETURNING id`, name).Scan(&id); err != nil {
			t.Fatalf("Titel %q: %v", name, err)
		}
		// Ohne Exemplar steht ein Titel in keinem Katalog (repository.SQLTitelHatExemplar).
		if _, err := pool.Exec(ctx, `INSERT INTO buecher_exemplare (titel_id, barcode_id) VALUES ($1, $2)`, id, barcode); err != nil {
			t.Fatalf("Exemplar %q: %v", barcode, err)
		}
		return id
	}
	mondflug := titel("Suchtest Mondflug", "SUCHTEST-1")
	ohneWort := titel("Suchtest ohne Wort", "SUCHTEST-2")
	t.Cleanup(func() {
		for _, sql := range []string{
			`DELETE FROM buecher_titel WHERE titel LIKE 'Suchtest %'`,
			`DELETE FROM schlagworte WHERE lower(wort) IN ('suchtest-weltraum', 'suchtest-raumfahrt')`,
		} {
			if _, err := pool.Exec(context.Background(), sql); err != nil {
				t.Errorf("Aufräumen: %v", err)
			}
		}
	})
	if _, err := repository.SetzeSchlagworte(ctx, pool, mondflug, []string{"Suchtest-Weltraum"}); err != nil {
		t.Fatal(err)
	}
	var weltraum string
	if err := pool.QueryRow(ctx, `SELECT id FROM schlagworte WHERE wort = 'Suchtest-Weltraum'`).Scan(&weltraum); err != nil {
		t.Fatal(err)
	}
	if err := repository.SetzeSchlagwortVerweis(ctx, pool, "Suchtest-Raumfahrt", weltraum); err != nil {
		t.Fatal(err)
	}

	type zeile struct {
		ID          string    `json:"id"`
		Suchwoerter *[]string `json:"suchwoerter"`
	}
	liste := func(q string) []zeile {
		t.Helper()
		rec := httptest.NewRecorder()
		handler.BearbeiteBuecherListe(rec, httptest.NewRequest(http.MethodGet, "/api/books?q="+q, nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("GET /api/books?q=%s: %d %s", q, rec.Code, rec.Body.String())
		}
		var antwort struct {
			Data []zeile `json:"data"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &antwort); err != nil {
			t.Fatal(err)
		}
		return antwort.Data
	}
	ids := func(zeilen []zeile) []string {
		var out []string
		for _, z := range zeilen {
			if z.ID == mondflug || z.ID == ohneWort {
				out = append(out, z.ID)
			}
		}
		return out
	}

	// Titel-Verwaltung: Suche am Server, über das Wort und über den Verweis darauf.
	for _, q := range []string{"suchtest-weltraum", "suchtest-raumfahrt", "raumfahrt"} {
		if got := ids(liste(q)); !slices.Equal(got, []string{mondflug}) {
			t.Errorf("?q=%s: %q, erwartet nur den Titel mit dem Schlagwort", q, got)
		}
	}

	// „Suche & Filter": die ganze Liste, jeder Titel mit seinen Suchwörtern.
	for _, z := range liste("") {
		switch z.ID {
		case mondflug:
			if want := []string{"Suchtest-Raumfahrt", "Suchtest-Weltraum"}; z.Suchwoerter == nil || !slices.Equal(*z.Suchwoerter, want) {
				t.Errorf("Suchwörter %v, erwartet %q — ohne sie findet die Suche im Browser nichts", z.Suchwoerter, want)
			}
		case ohneWort:
			if z.Suchwoerter != nil {
				t.Errorf("ein Titel ohne Schlagworte trägt suchwoerter = %q", *z.Suchwoerter)
			}
		}
	}
}
