package inventur

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bibliothek/internal/pgtest"
)

// Einen Titel zu ändern, den es nicht gibt, ist ein 404 mit Auskunft und kein 500.
func TestBuchAendern_UnbekannterTitelIst404(t *testing.T) {
	pool := pgtest.Pool(t)
	handler := &APIHandler{repo: &BookRepository{db: pool}, metadaten: offlineMetadatenClient()}

	const unbekannt = "00000000-0000-0000-0000-00000000dead"
	req := httptest.NewRequest(http.MethodPut, "/api/books/"+unbekannt,
		strings.NewReader(`{"title":"Nie da","author":"Niemand"}`))
	req.SetPathValue("id", unbekannt)
	rec := httptest.NewRecorder()
	handler.BearbeiteBuchAktualisieren(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("Status %d, erwartet 404: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "Buch nicht gefunden") {
		t.Errorf("die Antwort nennt den Grund nicht: %s", rec.Body.String())
	}
}
