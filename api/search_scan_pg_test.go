package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"bibliothek/repository"
)

// GET /api/search erkennt Scans exakt (Treffer), ohne zu buchen: Exemplar-Barcode,
// Littera-Etikett (EAN-13 → Mediennummer) und Schülerausweis — dieselbe Reihenfolge wie
// die Theke (resolveOhnePraefix), nur als Auskunft. Grundlage der globalen Suchleiste
// (03.09.2026): Buch → Buchakte, Ausweis → Schülerakte.
func TestSearch_ErkenntScansOhneZuBuchen(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()
	titelID := titelMitMeldebestand(t, pool, "Sprungbuch", 1)
	exID := exemplar(t, pool, titelID, "B-SPRUNG-1", true, "")
	var schuelerID string
	if err := pool.QueryRow(ctx, `INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr)
		VALUES ('S-SPRUNG-1', 'Greta', 'Sprung', '07A', 2031) RETURNING id`).Scan(&schuelerID); err != nil {
		t.Fatal(err)
	}
	s := &Server{}
	h := s.SearchHandler(repository.NewStudentRepository(pool), repository.NewBookRepository(pool))
	suche := func(q string) UnifiedSearchResult {
		t.Helper()
		w := httptest.NewRecorder()
		h(w, httptest.NewRequest(http.MethodGet, "/api/search?q="+q, nil))
		if w.Code != http.StatusOK {
			t.Fatalf("%s: HTTP %d %s", q, w.Code, w.Body.String())
		}
		var res UnifiedSearchResult
		if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
			t.Fatal(err)
		}
		return res
	}

	if r := suche("B-SPRUNG-1"); r.Treffer == nil || r.Treffer.Typ != "exemplar" || r.Treffer.ID != exID || r.Treffer.TitelID != titelID {
		t.Errorf("Exemplar-Barcode: %+v", r.Treffer)
	}
	if r := suche("S-SPRUNG-1"); r.Treffer == nil || r.Treffer.Typ != "schueler" || r.Treffer.ID != schuelerID {
		t.Errorf("Ausweis: %+v", r.Treffer)
	}
	if r := suche("Sprung"); r.Treffer != nil || len(r.Students) != 1 || len(r.Books) != 1 {
		t.Errorf("Freitext: treffer=%+v schueler=%d buecher=%d", r.Treffer, len(r.Students), len(r.Books))
	}
	// Nichts gebucht: das Exemplar hat keine offene Ausleihe.
	var offen int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ausleihen WHERE exemplar_id = $1 AND rueckgabe_am IS NULL`, exID).Scan(&offen); err != nil {
		t.Fatal(err)
	}
	if offen != 0 {
		t.Error("die Suche darf nicht buchen")
	}
}
