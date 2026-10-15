package inventur

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// lookupAnfrage baut den GET für eine ISBN. Den Platzhalter {isbn} füllt im Betrieb der
// Mux (api_routen.go); die Route selbst prüft er auch.
func lookupAnfrage(isbn string) *http.Request {
	anfrage := httptest.NewRequest(http.MethodGet, "/api/lookup/x", nil)
	anfrage.SetPathValue("isbn", isbn)
	return anfrage
}

func TestISBNSuche_LehntUngueltigeISBNAb(t *testing.T) {
	tueren := &Tueren{}
	anfrage := lookupAnfrage("123&foo=bar")
	rr := httptest.NewRecorder()

	tueren.ISBNSuche(rr, anfrage)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}
}

type mockTransportForLookup struct {
	RoundTripFunc func(anfrage *http.Request) (*http.Response, error)
}

func (m *mockTransportForLookup) RoundTrip(anfrage *http.Request) (*http.Response, error) {
	return m.RoundTripFunc(anfrage)
}

func TestISBNSuche_SucheScheitert(t *testing.T) {
	mockClient := &http.Client{
		Transport: &mockTransportForLookup{
			RoundTripFunc: func(anfrage *http.Request) (*http.Response, error) {
				return nil, errors.New("simulated network error")
			},
		},
	}

	metadaten := NewMetadatenClient()
	metadaten.SetzeHTTPClientFuerTest(mockClient)

	tueren := &Tueren{
		metadaten: metadaten,
	}

	// 9783161484100 ist eine gültige ISBN.
	anfrage := lookupAnfrage("9783161484100")
	rr := httptest.NewRecorder()

	tueren.ISBNSuche(rr, anfrage)

	// Ein Ausfall des Netzes ist kein fehlender Treffer: keine Quelle erreichbar → 502
	// (lookup_ausfall_test.go), nur „erreichbar ohne Treffer" → 404.
	if rr.Code != http.StatusBadGateway {
		t.Fatalf("expected status %d, got %d. Body: %s", http.StatusBadGateway, rr.Code, rr.Body.String())
	}
}

// lookupMitDNB fragt die Tür nach einer ISBN, zu der die DNB die genannten Datenfelder meldet;
// die übrigen Katalogdienste kennen sie nicht.
func lookupMitDNB(t *testing.T, datenfelder string) *httptest.ResponseRecorder {
	t.Helper()
	dnbXML := `<?xml version="1.0" encoding="UTF-8"?>
<searchRetrieveResponse xmlns="http://www.loc.gov/zing/srw/">
  <records>
    <record>
      <recordData>
        <record xmlns="http://www.loc.gov/MARC21/slim">` + datenfelder + `</record>
      </recordData>
    </record>
  </records>
</searchRetrieveResponse>`
	mockTr := &mockTransportForLookup{
		RoundTripFunc: func(anfrage *http.Request) (*http.Response, error) {
			status, koerper := http.StatusNotFound, ""
			if strings.Contains(anfrage.URL.String(), "services.dnb.de") {
				status, koerper = http.StatusOK, dnbXML
			}
			return &http.Response{
				StatusCode: status,
				Body:       io.NopCloser(strings.NewReader(koerper)),
				Header:     make(http.Header),
			}, nil
		},
	}
	metadaten := NewMetadatenClient()
	metadaten.SetzeHTTPClientFuerTest(&http.Client{Transport: mockTr})
	tueren := &Tueren{metadaten: metadaten}

	rr := httptest.NewRecorder()
	tueren.ISBNSuche(rr, lookupAnfrage("9783161484100"))

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d. Body: %s", http.StatusOK, rr.Code, rr.Body.String())
	}
	return rr
}

func TestISBNSuche_Erfolg(t *testing.T) {
	rr := lookupMitDNB(t, `
          <datafield tag="245" ind1="1" ind2="0">
            <subfield code="a">Mocked Title</subfield>
          </datafield>
          <datafield tag="100" ind1="1" ind2=" ">
            <subfield code="a">Mocked Author</subfield>
          </datafield>`)

	erwartetesJSON := `"title":"Mocked Title"`
	if !strings.Contains(rr.Body.String(), erwartetesJSON) {
		t.Fatalf("expected response to contain %s, got %s", erwartetesJSON, rr.Body.String())
	}
}

// Nennt der Titel zwei Jahrgänge, trägt die Antwort die Spanne in „jahrgangVon" und
// „jahrgangBis". Das Titelfeld ist das der DNB zur ISBN 9783141096835.
func TestISBNSuche_SpanneAusDemTitel(t *testing.T) {
	rr := lookupMitDNB(t, `
          <datafield tag="245" ind1="1" ind2="0">
            <subfield code="a">EinFach Deutsch Unterrichtsmodelle</subfield>
            <subfield code="b">John Green: Eine wie Alaska Klassen 8 - 10</subfield>
          </datafield>`)

	var antwort struct {
		Data struct {
			JahrgangVon int `json:"jahrgangVon"`
			JahrgangBis int `json:"jahrgangBis"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &antwort); err != nil {
		t.Fatalf("Antwort nicht lesbar: %v — %s", err, rr.Body.String())
	}
	if antwort.Data.JahrgangVon != 8 || antwort.Data.JahrgangBis != 10 {
		t.Fatalf("jahrgangVon = %d, jahrgangBis = %d, erwartet 8 und 10 — %s", antwort.Data.JahrgangVon, antwort.Data.JahrgangBis, rr.Body.String())
	}
}

func TestISBNSuche_LehntFehlendeISBNAb(t *testing.T) {
	tueren := &Tueren{}
	// Nur Leerzeichen im Platzhalter: nach dem Trimmen ist die ISBN leer
	anfrage := lookupAnfrage("   ")
	rr := httptest.NewRecorder()

	tueren.ISBNSuche(rr, anfrage)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "isbn fehlt") {
		t.Fatalf("expected response to contain isbn fehlt, got %s", rr.Body.String())
	}
}
