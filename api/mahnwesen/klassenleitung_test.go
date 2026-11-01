package mahnwesen

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Ohne Klasse oder ohne Adresse gibt es nichts zu speichern, auch wenn nur Leerzeichen kommen: Eine
// Klasse mit leerer Adresse stünde als versorgt da, und ihre Mahnlisten gingen nirgendwohin.
func TestKlassenleitungSetzen_VerlangtKlasseUndAdresse(t *testing.T) {
	faelle := []struct{ name, rumpf string }{
		{"ohne Adresse", `{"klasse":"5a","lehrer_email":""}`},
		{"Adresse nur Leerzeichen", `{"klasse":"5a","lehrer_email":"  "}`},
		{"ohne Klasse", `{"klasse":"","lehrer_email":"a@schule.invalid"}`},
		{"Klasse nur Leerzeichen", `{"klasse":"  ","lehrer_email":"a@schule.invalid"}`},
	}
	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/klassen-lehrer", strings.NewReader(f.rumpf))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			(&Tueren{}).KlassenleitungSetzen()(rec, req)
			if rec.Code != http.StatusBadRequest {
				t.Errorf("Status %d, erwartet 400: %s", rec.Code, rec.Body.String())
			}
		})
	}
}
