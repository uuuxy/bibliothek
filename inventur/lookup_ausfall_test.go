package inventur

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bibliothek/internal/katalogdienste"
)

// Ein Ausfall des Netzes ist kein fehlender Treffer.
//
// Antworteten die drei Wege des Nachschlagens (api/katalog/isbn_zu_titel.go,
// inventur/isbn_suche.go, inventur/cover_aktualisierung.go) bei ausgefallenem Netz mit 404 „nicht
// gefunden", zeigte die Theke denselben Bildschirm wie bei einem Buch, das die DNB wirklich nicht
// kennt, und bei einer Störung des WLANs katalogisierte jemand Bücher von Hand, die längst in der
// DNB stehen. Deshalb 502 mit klarer Meldung.

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

func findeISBNMit(t *testing.T, transport http.RoundTripper) *httptest.ResponseRecorder {
	t.Helper()
	client := katalogdienste.NewMetadatenClient()
	client.SetzeHTTPClientFuerTest(&http.Client{Transport: transport})
	tueren := &Tueren{metadaten: client}

	anfrage := httptest.NewRequest(http.MethodGet, "/api/lookup/9783551551672", nil)
	anfrage.SetPathValue("isbn", "9783551551672")
	rec := httptest.NewRecorder()
	tueren.ISBNSuche(rec, anfrage)
	return rec
}

func TestLookup_NetzausfallIst502(t *testing.T) {
	rec := findeISBNMit(t, kaputtesNetz{})
	if rec.Code != http.StatusBadGateway {
		t.Fatalf("Netzausfall: Status = %d (%s), erwartet 502 — der Ausfall sieht aus wie „Buch existiert nicht“", rec.Code, strings.TrimSpace(rec.Body.String()))
	}
}

func TestLookup_ErreichbarOhneTrefferBleibt404(t *testing.T) {
	rec := findeISBNMit(t, leereQuellen{})
	if rec.Code != http.StatusNotFound {
		t.Fatalf("Kein Treffer bei erreichbaren Quellen: Status = %d (%s), erwartet 404", rec.Code, strings.TrimSpace(rec.Body.String()))
	}
}

// Die Unterscheidung lebt im Client: Transportfehler tragen das Sentinel, ein leeres
// Ergebnis erreichbarer Quellen nicht.
func TestSucheNachISBN_KlassifiziertAusfallUndTreffer(t *testing.T) {
	client := katalogdienste.NewMetadatenClient()
	client.SetzeHTTPClientFuerTest(&http.Client{Transport: kaputtesNetz{}})
	_, err := client.SucheNachISBN(t.Context(), "9783551551672")
	if !errors.Is(err, katalogdienste.ErrKatalogdiensteNichtErreichbar) {
		t.Fatalf("Netzausfall = %v, erwartet ErrKatalogdiensteNichtErreichbar", err)
	}

	client.SetzeHTTPClientFuerTest(&http.Client{Transport: leereQuellen{}})
	_, err = client.SucheNachISBN(t.Context(), "9783551551672")
	if err == nil || errors.Is(err, katalogdienste.ErrKatalogdiensteNichtErreichbar) {
		t.Fatalf("erreichbar ohne Treffer = %v, erwartet schlichten Nicht-Treffer", err)
	}
}
