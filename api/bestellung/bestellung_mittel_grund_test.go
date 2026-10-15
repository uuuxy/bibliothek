package bestellung

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bibliothek/api/tuer"
	"bibliothek/internal/tuertest"
)

// Die Korrektur des Topfs verlangt einen Grund, bevor sie die Datenbank fragt; ohne ihn sagt die
// Antwort, was fehlt.
func TestBestellungMittelKorrigieren_OhneGrund(t *testing.T) {
	srv := &Tueren{Basis: &tuer.Basis{}}
	req := httptest.NewRequest(http.MethodPut, "/api/bestellungen/b-1/mittel", strings.NewReader(`{"mittel":"land","grund":"  "}`))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", "b-1")
	rec := httptest.NewRecorder()
	srv.BestellungMittelKorrigieren()(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("BestellungMittelKorrigieren: Status %d, erwartet 400: %s", rec.Code, rec.Body.String())
	}
	const erwartet = "Bitte einen Grund für die Korrektur angeben"
	if got := tuertest.Fehlermeldung(t, rec); got != erwartet {
		t.Errorf("BestellungMittelKorrigieren: Meldung %q, erwartet %q", got, erwartet)
	}
}
