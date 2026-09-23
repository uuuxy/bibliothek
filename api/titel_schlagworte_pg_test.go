package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"bibliothek/db"
	"bibliothek/repository"
)

// Die Tür des Bestellkorbs für Schlagworte (Migration 138): GET liest die vorhandenen,
// PUT ersetzt die Menge. Die Regeln selbst prüft repository/schlagworte_pg_test.go; hier
// geht es um das, was nur an der Tür entstehen kann — ein fehlendes Feld, eine falsche
// Kennung, die Übersetzung der fachlichen Fehler in 400 und 404.
func TestTitelSchlagworte_TuerDesBestellkorbs(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `TRUNCATE schlagworte CASCADE`); err != nil {
		t.Fatal(err)
	}
	srv := &Server{DB: &db.Database{Pool: pool}}
	var id string
	if err := pool.QueryRow(ctx, `INSERT INTO buecher_titel (titel) VALUES ('Woodwalkers') RETURNING id`).Scan(&id); err != nil {
		t.Fatal(err)
	}

	aufruf := func(methode, titelID, koerper string) (int, TitelSchlagworte, string) {
		t.Helper()
		req := httptest.NewRequest(methode, "/api/buecher/titel/"+titelID+"/schlagworte", strings.NewReader(koerper))
		req.SetPathValue("id", titelID)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		if methode == http.MethodGet {
			srv.GetTitelSchlagworteHandler()(rec, req)
		} else {
			srv.PutTitelSchlagworteHandler()(rec, req)
		}
		var antwort TitelSchlagworte
		if rec.Code == http.StatusOK {
			if err := json.Unmarshal(rec.Body.Bytes(), &antwort); err != nil {
				t.Fatalf("Antwort unlesbar: %v — %s", err, rec.Body.String())
			}
		}
		return rec.Code, antwort, rec.Body.String()
	}
	gelesen := func() []string {
		t.Helper()
		code, antwort, text := aufruf(http.MethodGet, id, "")
		if code != http.StatusOK {
			t.Fatalf("GET: %d %s", code, text)
		}
		return antwort.Schlagworte
	}

	if got := gelesen(); got == nil || len(got) != 0 {
		t.Errorf("ohne Schlagworte: %q, erwartet [] (nicht null)", got)
	}

	code, antwort, text := aufruf(http.MethodPut, id, `{"schlagworte":["Tierfantasy"," Magische  Schule "]}`)
	if code != http.StatusOK {
		t.Fatalf("PUT: %d %s", code, text)
	}
	if want := []string{"Magische Schule", "Tierfantasy"}; !slices.Equal(antwort.Schlagworte, want) || !slices.Equal(gelesen(), want) {
		t.Errorf("nach PUT: Antwort %q, gelesen %q, erwartet %q", antwort.Schlagworte, gelesen(), want)
	}

	// Ein Körper ohne das Feld ist ein Fehler — er darf nicht still alle entfernen.
	for _, koerper := range []string{`{}`, `{"schlagworte":null}`} {
		if code, _, text := aufruf(http.MethodPut, id, koerper); code != http.StatusBadRequest {
			t.Errorf("PUT %s: %d, erwartet 400: %s", koerper, code, text)
		}
	}
	if got := gelesen(); len(got) != 2 {
		t.Errorf("nach dem abgewiesenen PUT: %q — die Schlagworte müssen stehen bleiben", got)
	}

	zuViele := make([]string, repository.SchlagworteJeTitelMax+1)
	for i := range zuViele {
		zuViele[i] = fmt.Sprintf("%q", fmt.Sprintf("Wort %d", i))
	}
	code, _, text = aufruf(http.MethodPut, id, `{"schlagworte":[`+strings.Join(zuViele, ",")+`]}`)
	if code != http.StatusBadRequest || !strings.Contains(text, "höchstens") {
		t.Errorf("zu viele: %d %s, erwartet 400 mit der Grenze im Text", code, text)
	}

	if code, antwort, text := aufruf(http.MethodPut, id, `{"schlagworte":[]}`); code != http.StatusOK || len(antwort.Schlagworte) != 0 {
		t.Errorf("leere Liste: %d %q %s, erwartet 200 und []", code, antwort.Schlagworte, text)
	}

	const unbekannt = "00000000-0000-0000-0000-000000000138"
	for _, methode := range []string{http.MethodGet, http.MethodPut} {
		if code, _, text := aufruf(methode, unbekannt, `{"schlagworte":["Fantasy"]}`); code != http.StatusNotFound {
			t.Errorf("%s unbekannter Titel: %d, erwartet 404: %s", methode, code, text)
		}
		if code, _, text := aufruf(methode, "keine-uuid", `{"schlagworte":["Fantasy"]}`); code != http.StatusBadRequest {
			t.Errorf("%s ohne UUID: %d, erwartet 400: %s", methode, code, text)
		}
	}
}

func TestSchlagwortVorschlaege_AusDemBestand(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `TRUNCATE schlagworte CASCADE`); err != nil {
		t.Fatal(err)
	}
	srv := &Server{DB: &db.Database{Pool: pool}}
	// Eine Liste, keine Map: Die zuerst angelegte Schreibweise gewinnt, die Reihenfolge
	// muss also feststehen — über eine Map war sie zufällig und der Test mal rot.
	for _, titel := range []struct {
		name    string
		woerter []string
	}{
		{"Tintenherz", []string{"Fantasy", "Bücher"}},
		{"Woodwalkers", []string{"fantasy", "Tiere"}},
	} {
		var id string
		if err := pool.QueryRow(ctx, `INSERT INTO buecher_titel (titel) VALUES ($1) RETURNING id`, titel.name).Scan(&id); err != nil {
			t.Fatal(err)
		}
		if _, err := repository.SetzeSchlagworte(ctx, pool, id, titel.woerter); err != nil {
			t.Fatal(err)
		}
	}

	rec := httptest.NewRecorder()
	srv.GetSchlagwortVorschlaegeHandler()(rec, httptest.NewRequest(http.MethodGet, "/api/schlagworte", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("Status %d: %s", rec.Code, rec.Body.String())
	}
	var vorschlaege []repository.SchlagwortZahl
	if err := json.Unmarshal(rec.Body.Bytes(), &vorschlaege); err != nil {
		t.Fatal(err)
	}
	want := []repository.SchlagwortZahl{{Wort: "Fantasy", Titel: 2}, {Wort: "Bücher", Titel: 1}, {Wort: "Tiere", Titel: 1}}
	if !slices.Equal(vorschlaege, want) {
		t.Errorf("Vorschläge %+v, erwartet %+v", vorschlaege, want)
	}
}
