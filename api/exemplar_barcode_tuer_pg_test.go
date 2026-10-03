package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bibliothek/db"
	"bibliothek/repository"
)

// Den Barcode eines Exemplars ändern, über die Tür: Ein leerer oder schon vergebener Barcode
// und ein unbekanntes Exemplar werden abgelehnt und ändern nichts; danach ersetzt die Nummer
// vom Etikett den Platzhalter.
func TestExemplarBarcodeAendern_UeberDieTuer(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	srv := &Server{DB: &db.Database{Pool: pool}}

	titel := titelMitMeldebestand(t, pool, "Barcode-Tuer", 0)
	platzhalter := exemplar(t, pool, titel, "AUTO-TUER-1", true, "")
	exemplar(t, pool, titel, "TUER-BELEGT", true, "")
	barcodeVon := func(id string) string {
		t.Helper()
		var barcode string
		if err := pool.QueryRow(context.Background(),
			`SELECT barcode_id FROM buecher_exemplare WHERE id = $1`, id).Scan(&barcode); err != nil {
			t.Fatalf("Barcode lesen: %v", err)
		}
		return barcode
	}
	setze := func(id, rumpf string) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(http.MethodPut, "/api/buecher/exemplare/"+id+"/barcode", strings.NewReader(rumpf))
		req.Header.Set("Content-Type", "application/json")
		req.SetPathValue("id", id)
		rec := httptest.NewRecorder()
		srv.UpdateCopyBarcodeHandler(repository.NewBookRepository(pool))(rec, req)
		return rec
	}

	for _, f := range []struct {
		name, id, rumpf string
		status          int
		stueck          string
	}{
		{"leerer Barcode", platzhalter, `{"barcode":""}`, http.StatusBadRequest, "barcode cannot be empty"},
		{"vergebener Barcode", platzhalter, `{"barcode":"TUER-BELEGT"}`, http.StatusConflict, "bereits von einem anderen Exemplar"},
		{"unbekanntes Exemplar", "00000000-0000-0000-0000-00000000dead", `{"barcode":"TUER-NEU"}`, http.StatusNotFound, "error"},
	} {
		rec := setze(f.id, f.rumpf)
		if rec.Code != f.status {
			t.Errorf("%s: Status %d, erwartet %d: %s", f.name, rec.Code, f.status, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), f.stueck) {
			t.Errorf("%s: die Antwort nennt %q nicht: %s", f.name, f.stueck, rec.Body.String())
		}
	}
	if got := barcodeVon(platzhalter); got != "AUTO-TUER-1" {
		t.Fatalf("eine Ablehnung hat den Barcode geändert: %q", got)
	}

	if rec := setze(platzhalter, `{"barcode":"TUER-NEU"}`); rec.Code != http.StatusOK {
		t.Fatalf("ein freier Barcode: Status %d, erwartet 200: %s", rec.Code, rec.Body.String())
	}
	if got := barcodeVon(platzhalter); got != "TUER-NEU" {
		t.Errorf("Barcode nach dem Ändern %q, erwartet TUER-NEU", got)
	}
}
