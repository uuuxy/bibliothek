package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"bibliothek/db"
	"bibliothek/sse"
)

// Ohne Anmeldung antwortet eine geschützte Tür mit dem einen Satz der Anmeldung, nicht mit dem
// Text der Prüfung („http: named cookie not present").
func TestGeschuetzteTuer_OhneAnmeldungMeldetEinenSatz(t *testing.T) {
	t.Chdir(t.TempDir())
	router := NewServer(&db.Verbindung{}, nil, sse.NewBroker(), ServerKonfig{}).Routes()
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/admin/system/betriebsbereitschaft", nil))
	var antwort struct {
		Fehler string `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &antwort); err != nil {
		t.Fatalf("Antwort ist kein JSON-Fehler (%s): %v", rec.Body.String(), err)
	}
	if want := "Nicht angemeldet. Bitte neu anmelden."; rec.Code != http.StatusUnauthorized || antwort.Fehler != want {
		t.Errorf("ohne Cookie: Status %d, Meldung %q, erwartet 401 und %q", rec.Code, antwort.Fehler, want)
	}
}
