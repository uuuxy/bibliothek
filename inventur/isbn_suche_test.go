package inventur

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandleLookupRejectsInvalidISBN(t *testing.T) {
	handler := &APIHandler{}
	req := httptest.NewRequest(http.MethodGet, "/api/lookup/123&foo=bar", nil)
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
	req := httptest.NewRequest(http.MethodGet, "/api/lookup/9783161484100", nil)
	rr := httptest.NewRecorder()

	handler.handleLookup(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d. Body: %s", http.StatusNotFound, rr.Code, rr.Body.String())
	}
}

func TestHandleLookupHappyPath(t *testing.T) {
	mockTr := &mockTransportForLookup{
		RoundTripFunc: func(req *http.Request) (*http.Response, error) {
			if strings.Contains(req.URL.String(), "services.dnb.de") {
				dnbXML := `<?xml version="1.0" encoding="UTF-8"?>
<searchRetrieveResponse xmlns="http://www.loc.gov/zing/srw/">
  <records>
    <record>
      <recordData>
        <record xmlns="http://www.loc.gov/MARC21/slim">
          <datafield tag="245" ind1="1" ind2="0">
            <subfield code="a">Mocked Title</subfield>
          </datafield>
          <datafield tag="100" ind1="1" ind2=" ">
            <subfield code="a">Mocked Author</subfield>
          </datafield>
        </record>
      </recordData>
    </record>
  </records>
</searchRetrieveResponse>`
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString(dnbXML)),
					Header:     make(http.Header),
				}, nil
			}
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Body:       io.NopCloser(bytes.NewBufferString("")),
				Header:     make(http.Header),
			}, nil
		},
	}

	mockClient := &http.Client{Transport: mockTr}
	metadaten := NeuerMetadatenClient()
	metadaten.SetzeHTTPClientFuerTest(mockClient)

	handler := &APIHandler{
		metadaten: metadaten,
	}

	req := httptest.NewRequest(http.MethodGet, "/api/lookup/9783161484100", nil)
	rr := httptest.NewRecorder()

	handler.handleLookup(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d. Body: %s", http.StatusOK, rr.Code, rr.Body.String())
	}

	expectedJSONSnippet := `"title":"Mocked Title"`
	if !strings.Contains(rr.Body.String(), expectedJSONSnippet) {
		t.Fatalf("expected response to contain %s, got %s", expectedJSONSnippet, rr.Body.String())
	}
}

func TestHandleLookupRejectsMissingISBN(t *testing.T) {
	handler := &APIHandler{}
	// Test the case where parts are extracted but isbn is empty string
	req := httptest.NewRequest(http.MethodGet, "/api/lookup/%20%20%20", nil)
	rr := httptest.NewRecorder()

	handler.handleLookup(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "isbn fehlt") {
		t.Fatalf("expected response to contain isbn fehlt, got %s", rr.Body.String())
	}
}

func TestHandleLookupRejectsInvalidRoute(t *testing.T) {
	handler := &APIHandler{}
	// Wrong route
	req := httptest.NewRequest(http.MethodGet, "/api/something_else/123", nil)
	rr := httptest.NewRecorder()

	handler.handleLookup(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}
}
