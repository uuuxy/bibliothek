package theke

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bibliothek/api/tuer"
	"bibliothek/internal/pgtest"
	"bibliothek/internal/testdaten"
	"bibliothek/repository"
)

// GET /api/search sucht unscharf nach Lesern und Titeln und bucht nie, auch nicht, wenn der
// Suchtext die Nummer eines Exemplars ist. Gebucht wird an der Theke nur über POST /api/action.
// Ein Exemplar zu einer Nummer findet der Medienkatalog über GET /api/buecher/exemplare/erkennen;
// diese Suche rechnet dafür nichts mehr aus.
func TestSuche_SuchtUnscharfUndBuchtNicht(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	ctx := context.Background()
	titelID := testdaten.LegeTitelAn(t, pool, "Sprungbuch", 1)
	exID := testdaten.LegeExemplarAn(t, pool, titelID, "B-SPRUNG-1", true, "")
	testdaten.LegeSchuelerMitNamenAn(t, pool, "S-SPRUNG-1", "Greta", "Sprung", "07A", 2031)
	s := &Tueren{Basis: &tuer.Basis{}}
	h := s.Suche(repository.NewLeserAbfragen(pool), repository.NewKatalogAbfragen(pool))
	suche := func(q string) (suchErgebnis, string) {
		t.Helper()
		w := httptest.NewRecorder()
		h(w, httptest.NewRequest(http.MethodGet, "/api/search?q="+q, nil))
		if w.Code != http.StatusOK {
			t.Fatalf("%s: HTTP %d %s", q, w.Code, w.Body.String())
		}
		var res suchErgebnis
		if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
			t.Fatal(err)
		}
		return res, w.Body.String()
	}

	if r, _ := suche("Sprung"); len(r.Leser) != 1 || len(r.Titel) != 1 {
		t.Errorf("Freitext: schueler=%d buecher=%d, erwartet je einen", len(r.Leser), len(r.Titel))
	}
	if _, body := suche("B-SPRUNG-1"); strings.Contains(body, `"treffer"`) {
		t.Errorf("die Antwort trägt noch einen Scan-Treffer, den niemand liest: %s", body)
	}
	var offen int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ausleihen WHERE exemplar_id = $1 AND rueckgabe_am IS NULL`, exID).Scan(&offen); err != nil {
		t.Fatal(err)
	}
	if offen != 0 {
		t.Error("die Suche darf nicht buchen")
	}
}
