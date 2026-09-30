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
)

// Die DNB liefert Umlaute zerlegt, als Grundbuchstabe mit Pünktchen dahinter (Migration 154).
// Am Testserver fand der öffentliche Katalog — und über dieselbe Tür „Mein Portal" —
// „Der Herr der Ringe - Anhänge und Register" mit „Anhänge" nicht, weder über ILIKE noch
// über den Volltext; mit „Register" schon. Die Titel entstehen hier zerlegt, wie die
// Bestellung sie aus der DNB schrieb; gesucht wird, wie man tippt: zusammengesetzt.
func TestOpacSuche_FindetTitelMitZerlegtenUmlauten(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()

	const zusammengesetzt = "Der Herr der Ringe - Anhänge und Register"
	ringe := titelMitSignatur(t, pool, "Der Herr der Ringe - Anha\u0308nge und Register", "", 0)
	exemplar(t, pool, ringe, "B-NFC-1", true, "")
	vogel := titelMitSignatur(t, pool, "Vogelwelt", "", 0)
	exemplar(t, pool, vogel, "B-NFC-2", true, "")
	if _, err := pool.Exec(ctx, `UPDATE buecher_titel SET autor = $2 WHERE id = $1`, vogel, "Mu\u0308ller, Gu\u0308nter"); err != nil {
		t.Fatal(err)
	}

	var gespeichert string
	if err := pool.QueryRow(ctx, `SELECT titel FROM buecher_titel WHERE id = $1`, ringe).Scan(&gespeichert); err != nil {
		t.Fatal(err)
	}
	if gespeichert != zusammengesetzt {
		t.Errorf("gespeichert %+q, erwartet %+q", gespeichert, zusammengesetzt)
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
		{"Anhänge", []string{zusammengesetzt}},     // Volltext
		{"Anhänge und", []string{zusammengesetzt}}, // Teilstring über ILIKE
		{"Günter", []string{"Vogelwelt"}},          // Autor, über ILIKE
		{"Register", []string{zusammengesetzt}},    // ohne Umlaut ging es schon vorher
	}
	for _, f := range faelle {
		if got := suche(f.q); !slices.Equal(got, f.titel) {
			t.Errorf("q=%+q: %+q, erwartet %+q", f.q, got, f.titel)
		}
	}
}
