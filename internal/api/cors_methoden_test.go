package api

import (
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
)

// Der Preflight eines erlaubten anderen Ursprungs nennt jede Methode, unter der eine Route
// angemeldet ist; sonst scheiterte jede Anfrage dieser Methode schon an ihm.
func TestCORS_PreflightNenntJedeMethodeDerRouten(t *testing.T) {
	const fremd = "https://bibliothek.schule.invalid"
	t.Setenv("ALLOWED_ORIGIN", fremd)
	h := corsMiddleware(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))

	methoden := map[string]bool{}
	for route := range registrierteRouten(t) {
		methode, _, ok := strings.Cut(route, " ")
		if ok && methode == strings.ToUpper(methode) && !strings.HasPrefix(methode, "/") {
			methoden[methode] = true
		}
	}
	if len(methoden) < 5 {
		t.Fatalf("nur %d Methoden an den Routen erkannt: %v", len(methoden), methoden)
	}
	for methode := range methoden {
		r := httptest.NewRequest(http.MethodOptions, "/api/x", nil)
		r.Header.Set("Origin", fremd)
		r.Header.Set("Access-Control-Request-Method", methode)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		erlaubt := w.Header().Get("Access-Control-Allow-Methods")
		if !slices.Contains(strings.Split(erlaubt, ", "), methode) {
			t.Errorf("Preflight für %s erlaubt %q, erwartet die Methode dabei", methode, erlaubt)
		}
	}
}
