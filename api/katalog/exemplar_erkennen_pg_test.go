package katalog

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"bibliothek/api/tuer"
	"bibliothek/internal/pgtest"
	"bibliothek/internal/testdaten"
	"bibliothek/repository"
)

// GET /api/buecher/exemplare/erkennen findet das Exemplar zu einer Nummer, damit die Suche im
// Medienkatalog seinen Titel zeigt: über den Barcode, über das Etikett von Littera (der Scanner
// liest bei 101634 den EAN-13 1016340039562) und über eine lange LMF-Nummer. Eine Nummer ohne
// Exemplar ist ein leerer Treffer und kein Fehler; ein Ausweis mit dieser Nummer kommt nicht
// heraus, die Tür hängt an view_books.
func TestExemplarErkennen_FindetExemplarUeberBarcodeUndEtikett(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)

	litteraTitel := testdaten.LegeTitelMitSignaturAn(t, pool, "Erkennen Littera-Band", "ERK 1", 1)
	litteraExemplar := testdaten.LegeExemplarAn(t, pool, litteraTitel, "101634", true, "")
	lmfTitel := testdaten.LegeTitelMitSignaturAn(t, pool, "Erkennen Lernmittel", "ERK 2", 1)
	lmfExemplar := testdaten.LegeExemplarAn(t, pool, lmfTitel, "LMF-9783060158423-001", true, "")
	testdaten.LegeSchuelerMitNamenAn(t, pool, "S-ERK-1", "Erna", "Erkennen", "07A", 2031)
	// Ein ausgesondertes Exemplar wird gefunden und als solches genannt: Sein Titel steht ohne
	// ein Exemplar im Bestand nicht in der Katalogliste.
	weg := testdaten.LegeExemplarAn(t, pool, lmfTitel, "B-ERK-AUS", false, "")
	if _, err := pool.Exec(t.Context(), `
		UPDATE buecher_exemplare SET ist_ausgesondert = true, aussonderung_grund = 'AUSSORTIERT'
		WHERE id = $1`, weg); err != nil {
		t.Fatalf("Exemplar aussondern: %v", err)
	}

	h := (&Tueren{Basis: &tuer.Basis{}}).ExemplarErkennen(repository.NewKatalogAbfragen(pool))
	rufe := func(code string) (int, string, exemplarErkennenAntwort) {
		t.Helper()
		w := httptest.NewRecorder()
		h(w, httptest.NewRequest(http.MethodGet, "/api/buecher/exemplare/erkennen?code="+url.QueryEscape(code), nil))
		var antwort exemplarErkennenAntwort
		if w.Code == http.StatusOK {
			if err := json.Unmarshal(w.Body.Bytes(), &antwort); err != nil {
				t.Fatal(err)
			}
		}
		return w.Code, w.Body.String(), antwort
	}

	faelle := []struct {
		name, code, exemplar, titel string
	}{
		{"Barcode, wie er gespeichert ist", "101634", litteraExemplar, litteraTitel},
		{"Etikett von Littera, wie der Scanner es liest", "1016340039562", litteraExemplar, litteraTitel},
		{"lange LMF-Nummer", "LMF-9783060158423-001", lmfExemplar, lmfTitel},
		{"Leerzeichen um die Nummer", " 101634 ", litteraExemplar, litteraTitel},
	}
	for _, f := range faelle {
		code, body, antwort := rufe(f.code)
		if code != http.StatusOK || antwort.Treffer == nil {
			t.Errorf("%s: HTTP %d, Antwort %s", f.name, code, body)
			continue
		}
		if antwort.Treffer.ID != f.exemplar || antwort.Treffer.TitelID != f.titel {
			t.Errorf("%s: Treffer %+v, erwartet Exemplar %s am Titel %s", f.name, *antwort.Treffer, f.exemplar, f.titel)
		}
	}

	if _, body, antwort := rufe("B-ERK-AUS"); antwort.Treffer == nil || antwort.Treffer.ID != weg || !antwort.Treffer.Ausgesondert {
		t.Errorf("ausgesondertes Exemplar: Antwort %s, erwartet Exemplar %s mit ausgesondert", body, weg)
	}
	if _, body, antwort := rufe("101634"); antwort.Treffer == nil || antwort.Treffer.Ausgesondert {
		t.Errorf("Exemplar im Bestand: Antwort %s, erwartet ohne ausgesondert", body)
	}

	for _, ohne := range []string{"999999", "S-ERK-1"} {
		if code, body, antwort := rufe(ohne); code != http.StatusOK || antwort.Treffer != nil {
			t.Errorf("%q: HTTP %d, Antwort %s; erwartet 200 mit leerem Treffer", ohne, code, body)
		}
	}

	if code, body, _ := rufe(""); code != http.StatusBadRequest {
		t.Errorf("leere Nummer: HTTP %d statt 400 (%s)", code, body)
	}
}
