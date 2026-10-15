package portal

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bibliothek/api/tuer"
	"bibliothek/internal/tuertest"
)

// Ohne Sitzung nehmen die Türen der Meldungen nichts an und zeigen nichts; die Antwort sagt,
// woran es liegt.
func TestAnliegen_OhneSitzung(t *testing.T) {
	srv := &Tueren{Basis: &tuer.Basis{}}
	for name, aufruf := range map[string]http.HandlerFunc{
		"AnliegenAnlegen": srv.AnliegenAnlegen(),
		"EigeneAnliegen":  srv.EigeneAnliegen(),
	} {
		rec := httptest.NewRecorder()
		aufruf(rec, httptest.NewRequest(http.MethodPost, "/api/anliegen", strings.NewReader(`{}`)))
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("%s: Status %d, erwartet 401: %s", name, rec.Code, rec.Body.String())
			continue
		}
		if got := tuertest.Fehlermeldung(t, rec); got != "Sitzungs-Information fehlt" {
			t.Errorf("%s: Meldung %q, erwartet %q", name, got, "Sitzungs-Information fehlt")
		}
	}
}
