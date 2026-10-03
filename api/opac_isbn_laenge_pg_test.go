package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"bibliothek/db"
)

// Der Katalog für Leser — und über dieselbe Tür „Mein Portal" — vergleicht eine getippte ISBN
// als Teilstring. Die zehnstellige vom Titelblatt und die dreizehnstellige, unter der die
// Datenbank den Titel führt (Migration 157), enden auf verschiedene Prüfzeichen: Ohne den
// Vergleich mit der Normalform des Suchtexts fände die getippte zehnstellige das Buch nicht.
func TestOpacSuche_FindetDenTitelUeberDieZehnstelligeISBN(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)

	const titel = "Advanced Organic Chemistry"
	id := titelMitSignatur(t, pool, titel, "", 0)
	exemplar(t, pool, id, "B-ISBN-OPAC-1", true, "")
	if _, err := pool.Exec(t.Context(), `UPDATE buecher_titel SET isbn = '0306406152' WHERE id = $1`, id); err != nil {
		t.Fatal(err)
	}

	s := &Server{DB: &db.Database{Pool: pool}}
	suche := func(q string) []OpacTitel {
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
		return treffer
	}

	for _, q := range []string{"0306406152", "0-306-40615-2", "9780306406157", "978-0-306-40615-7"} {
		if treffer := suche(q); len(treffer) != 1 || treffer[0].Titel != titel {
			t.Errorf("q=%q: %d Treffer, erwartet den Titel %q", q, len(treffer), titel)
		}
	}
	// Gegenprobe: Dieselben Ziffern mit falschem Prüfzeichen sind eine andere Nummer.
	if treffer := suche("0306406153"); len(treffer) != 0 {
		t.Errorf("falsches Prüfzeichen: %d Treffer, erwartet keinen", len(treffer))
	}
}
