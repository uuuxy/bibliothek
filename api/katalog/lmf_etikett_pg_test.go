package katalog

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"bibliothek/api/tuer"
	"bibliothek/db"
	"bibliothek/internal/pgtest"
	"bibliothek/internal/testdaten"
	"bibliothek/repository"
)

// Ein altes LMF-Etikett führt in der Suche des Medienkatalogs zu seinem Exemplar, und die
// Exemplarliste des Titels nennt es: Das Nummernfeld des Druck-Centers findet das Exemplar
// darüber.
func TestAltesLMFEtikett_SucheUndExemplarliste(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)

	titelID := testdaten.LegeTitelAn(t, pool, "LMF-Katalog-Titel", 0)
	exemplarID := testdaten.LegeExemplarAn(t, pool, titelID, "B-LMFK-1", true, "")
	testdaten.LegeExemplarAn(t, pool, titelID, "B-LMFK-2", true, "")
	if _, err := pool.Exec(context.Background(),
		`UPDATE buecher_exemplare SET lmf_barcode = 'LMF-9783060158423-901' WHERE id = $1`, exemplarID); err != nil {
		t.Fatalf("altes Etikett setzen: %v", err)
	}
	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}}}

	req := httptest.NewRequest(http.MethodGet, "/api/buecher/exemplare/erkennen?code=LMF-9783060158423-901", nil)
	rec := httptest.NewRecorder()
	srv.ExemplarErkennen(repository.NewKatalogAbfragen(pool))(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("Suche: Status %d: %s", rec.Code, rec.Body.String())
	}
	var antwort exemplarErkennenAntwort
	if err := json.Unmarshal(rec.Body.Bytes(), &antwort); err != nil {
		t.Fatalf("Antwort: %v", err)
	}
	if antwort.Treffer == nil || antwort.Treffer.ID != exemplarID || antwort.Treffer.Barcode != "B-LMFK-1" {
		t.Errorf("Treffer %+v, erwartet das Exemplar %s mit der Nummer B-LMFK-1", antwort.Treffer, exemplarID)
	}

	karten := exemplarKarten(t, srv, pool, titelID)
	if got := karten["B-LMFK-1"]["lmf_barcode"]; got != "LMF-9783060158423-901" {
		t.Errorf("lmf_barcode in der Exemplarliste = %v, erwartet LMF-9783060158423-901", got)
	}
	if got := karten["B-LMFK-2"]["lmf_barcode"]; got != "" {
		t.Errorf("lmf_barcode ohne altes Etikett = %v, erwartet leer", got)
	}
}
