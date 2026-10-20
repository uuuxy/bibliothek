package theke

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"bibliothek/api/tuer"
	"bibliothek/internal/pgtest"
	"bibliothek/internal/testdaten"
	"bibliothek/repository"
)

// GET /api/search erkennt Scans exakt (Treffer), ohne zu buchen: Exemplar-Barcode,
// Littera-Etikett (EAN-13 → Mediennummer) und Schülerausweis, dieselbe Reihenfolge wie die
// Theke (verarbeiteOhneVorsilbe), nur als Auskunft. Damit springt die globale Suchleiste
// direkt zur Akte: Buch → Buchakte, Ausweis → Schülerakte.
func TestSuche_ErkenntScansOhneZuBuchen(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	ctx := context.Background()
	titelID := testdaten.LegeTitelAn(t, pool, "Sprungbuch", 1)
	exID := testdaten.LegeExemplarAn(t, pool, titelID, "B-SPRUNG-1", true, "")
	schuelerID := testdaten.LegeSchuelerMitNamenAn(t, pool, "S-SPRUNG-1", "Greta", "Sprung", "07A", 2031)
	if _, err := pool.Exec(ctx, `DELETE FROM leser WHERE barcode_id = 'A-KOLL-77'`); err != nil {
		t.Fatal(err)
	}
	kollegeID := testdaten.LegeLeserAn(t, pool, "A-KOLL-77", "Kim", "Kollegin", "lehrkraft")
	s := &Tueren{Basis: &tuer.Basis{}}
	h := s.Suche(repository.NewLeserAbfragen(pool), repository.NewKatalogAbfragen(pool))
	suche := func(q string) suchErgebnis {
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
		return res
	}

	if r := suche("B-SPRUNG-1"); r.Treffer == nil || r.Treffer.Typ != "exemplar" || r.Treffer.ID != exID || r.Treffer.TitelID != titelID {
		t.Errorf("Exemplar-Barcode: %+v", r.Treffer)
	}
	if r := suche("S-SPRUNG-1"); r.Treffer == nil || r.Treffer.Typ != "schueler" || r.Treffer.ID != schuelerID {
		t.Errorf("Ausweis: %+v", r.Treffer)
	}
	// Der Ausweis eines Kollegen ist derselbe Sprung: Die Akte ist eine Maske für jeden, und
	// die Theke löst ihn über FindeZuBarcode auf. Fragte die Scan-Erkennung die Sicht
	// `schueler`, wäre Treffer nil, und die Suchleiste fiele auf die unscharfe Namenssuche
	// zurück.
	if r := suche("A-KOLL-77"); r.Treffer == nil || r.Treffer.Typ != "schueler" || r.Treffer.ID != kollegeID {
		t.Errorf("Ausweis eines Kollegen: %+v — erwartet Treffer auf %s", r.Treffer, kollegeID)
	}
	if r := suche("Sprung"); r.Treffer != nil || len(r.Leser) != 1 || len(r.Titel) != 1 {
		t.Errorf("Freitext: treffer=%+v schueler=%d buecher=%d", r.Treffer, len(r.Leser), len(r.Titel))
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
