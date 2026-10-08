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
	req := httptest.NewRequest(http.MethodGet, "/api/lookup/x", nil)
	req.SetPathValue("isbn", isbn)
	return req
}

func TestHandleLookupRejectsInvalidISBN(t *testing.T) {
	handler := &APIHandler{}
	req := lookupAnfrage("123&foo=bar")
	rr := httptest.NewRecorder()

	handler.handleLookup(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}
}

type mockTransportForLookup struct {
	RoundTripFunc func(req *http.Request) (*http.Response, error)
}

func (m *mockTransportForLookup) RoundTrip(req *http.Request) (*http.Response, error) {
	return m.RoundTripFunc(req)
}

func TestHandleLookupSucheFehlschlag(t *testing.T) {
	mockClient := &http.Client{
		Transport: &mockTransportForLookup{
			RoundTripFunc: func(req *http.Request) (*http.Response, error) {
				return nil, errors.New("simulated network error")
			},
		},
	}

	metadaten := NeuerMetadatenClient()
	metadaten.SetzeHTTPClientFuerTest(mockClient)

	handler := &APIHandler{
		metadaten: metadaten,
	}

	// 9783161484100 is a valid ISBN
	req := lookupAnfrage("9783161484100")
	rr := httptest.NewRecorder()

	handler.handleLookup(rr, req)

	// Seit dem 31.08.2026 ist ein Netzausfall KEIN Nicht-Treffer mehr: keine Quelle
	// erreichbar → 502 (siehe lookup_ausfall_test.go), nur „erreichbar ohne Treffer" → 404.
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
		RoundTripFunc: func(req *http.Request) (*http.Response, error) {
			status, koerper := http.StatusNotFound, ""
			if strings.Contains(req.URL.String(), "services.dnb.de") {
				status, koerper = http.StatusOK, dnbXML
			}
			return &http.Response{
				StatusCode: status,
				Body:       io.NopCloser(strings.NewReader(koerper)),
				Header:     make(http.Header),
			}, nil
		},
	}
	metadaten := NeuerMetadatenClient()
	metadaten.SetzeHTTPClientFuerTest(&http.Client{Transport: mockTr})
	handler := &APIHandler{metadaten: metadaten}

	rr := httptest.NewRecorder()
	handler.handleLookup(rr, lookupAnfrage("9783161484100"))

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d. Body: %s", http.StatusOK, rr.Code, rr.Body.String())
	}
	return rr
}

func TestHandleLookupHappyPath(t *testing.T) {
	rr := lookupMitDNB(t, `
          <datafield tag="245" ind1="1" ind2="0">
            <subfield code="a">Mocked Title</subfield>
          </datafield>
          <datafield tag="100" ind1="1" ind2=" ">
            <subfield code="a">Mocked Author</subfield>
          </datafield>`)

	expectedJSONSnippet := `"title":"Mocked Title"`
	if !strings.Contains(rr.Body.String(), expectedJSONSnippet) {
		t.Fatalf("expected response to contain %s, got %s", expectedJSONSnippet, rr.Body.String())
	}
}

// Nennt der Titel zwei Jahrgänge, trägt die Antwort die Spanne in „jahrgangVon" und
// „jahrgangBis". Das Titelfeld ist das der DNB zur ISBN 9783141096835.
func TestHandleLookup_SpanneAusDemTitel(t *testing.T) {
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

func TestHandleLookupRejectsMissingISBN(t *testing.T) {
	handler := &APIHandler{}
	// Nur Leerzeichen im Platzhalter: nach dem Trimmen ist die ISBN leer
	req := lookupAnfrage("   ")
	rr := httptest.NewRecorder()

	handler.handleLookup(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "isbn fehlt") {
		t.Fatalf("expected response to contain isbn fehlt, got %s", rr.Body.String())
	}
}
