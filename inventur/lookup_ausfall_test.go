package inventur

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Netzausfall ist kein Nicht-Treffer.
//
// Bis zum 31.08.2026 antworteten alle drei Lookup-Wege (api/isbn_handler.go,
// inventur/isbn_suche.go, inventur/cover_aktualisierung.go) bei ausgefallenem Netz mit
// 404 „nicht gefunden" — für die Theke derselbe Bildschirm wie ein Buch, das die DNB
// wirklich nicht kennt. Bei einer WLAN-Störung katalogisiert dann jemand Bücher von
// Hand, die längst in der DNB stehen (Produktentscheidung 31.08.2026: 502 mit klarer
// Meldung; Sweep „Fehler-Kollaps", docs/sweeps.md).

// kaputtesNetz lässt jede Verbindung scheitern — wie ein gezogener Uplink.
type kaputtesNetz struct{}

func (kaputtesNetz) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, errors.New("dial tcp: no route to host")
}

// leereQuellen antwortet überall 200 mit leerem Inhalt — erreichbar, aber ohne Treffer.
type leereQuellen struct{}

func (leereQuellen) RoundTrip(r *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       http.NoBody,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Request:    r,
	}, nil
}

func lookupMit(t *testing.T, transport http.RoundTripper) *httptest.ResponseRecorder {
	t.Helper()
	client := NeuerMetadatenClient()
	client.SetzeHTTPClientFuerTest(&http.Client{Transport: transport})
	handler := &APIHandler{metadaten: client}

	req := httptest.NewRequest(http.MethodGet, "/api/lookup/9783551551672", nil)
	rec := httptest.NewRecorder()
	handler.handleLookup(rec, req)
	return rec
}

func TestLookup_NetzausfallIst502(t *testing.T) {
	rec := lookupMit(t, kaputtesNetz{})
	if rec.Code != http.StatusBadGateway {
		t.Fatalf("Netzausfall: erwartet 502, bekam %d (%s) — Ausfall sieht aus wie „Buch existiert nicht»",
			rec.Code, strings.TrimSpace(rec.Body.String()))
	}
}

func TestLookup_ErreichbarOhneTrefferBleibt404(t *testing.T) {
	rec := lookupMit(t, leereQuellen{})
	if rec.Code != http.StatusNotFound {
		t.Fatalf("Kein Treffer bei erreichbaren Quellen: erwartet 404, bekam %d (%s)",
			rec.Code, strings.TrimSpace(rec.Body.String()))
	}
}

// Die Unterscheidung lebt im Client: Transportfehler tragen das Sentinel, ein leeres
// Ergebnis erreichbarer Quellen nicht.
func TestSucheNachISBN_KlassifiziertAusfallUndTreffer(t *testing.T) {
	client := NeuerMetadatenClient()
	client.SetzeHTTPClientFuerTest(&http.Client{Transport: kaputtesNetz{}})
	_, err := client.SucheNachISBN(t.Context(), "9783551551672")
	if !errors.Is(err, ErrKatalogdiensteNichtErreichbar) {
		t.Fatalf("Netzausfall: erwartet ErrKatalogdiensteNichtErreichbar, bekam %v", err)
	}

	client.SetzeHTTPClientFuerTest(&http.Client{Transport: leereQuellen{}})
	_, err = client.SucheNachISBN(t.Context(), "9783551551672")
	if err == nil || errors.Is(err, ErrKatalogdiensteNichtErreichbar) {
		t.Fatalf("erreichbar ohne Treffer: erwartet schlichten Nicht-Treffer, bekam %v", err)
	}
}
