package inventur

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

type mockTransport struct {
	roundTripFunc func(anfrage *http.Request) (*http.Response, error)
}

func (m *mockTransport) RoundTrip(anfrage *http.Request) (*http.Response, error) {
	return m.roundTripFunc(anfrage)
}

func TestSucheNachISBN(t *testing.T) {
	// Jeder Subtest steckt in einer eigenen Top-Level-Funktion, um die doppelte
	// Closure-Verschachtelung (t.Run → roundTripFunc) und damit die kognitive
	// Komplexität gering zu halten. Namen/Hierarchie bleiben unverändert.
	t.Run("Invalid ISBN", testSucheISBNInvalid)
	t.Run("Valid ISBN, Found in DNB", testSucheISBNDNB)
	t.Run("Valid ISBN, Found in Google Books", testSucheISBNGoogle)
	t.Run("Valid ISBN, Found in OpenLibrary", testSucheISBNOpenLibrary)
	t.Run("Valid ISBN, Not Found Anywhere", testSucheISBNNotFound)
}

func testSucheISBNInvalid(t *testing.T) {
	client := &MetadatenClient{}
	_, err := client.SucheNachISBN(context.Background(), "invalid")
	if err == nil || !strings.Contains(err.Error(), "ungültiges ISBN format") {
		t.Errorf("SucheNachISBN(…, \"invalid\") = %v, erwartet den Fehler „ungültiges ISBN format“", err)
	}
}

func testSucheISBNDNB(t *testing.T) {
	mockTr := &mockTransport{
		roundTripFunc: func(anfrage *http.Request) (*http.Response, error) {
			if strings.Contains(anfrage.URL.String(), "services.dnb.de") {
				dnbXML := `<?xml version="1.0" encoding="UTF-8"?>
<searchRetrieveResponse xmlns="http://www.loc.gov/zing/srw/">
  <records>
    <record>
      <recordData>
        <record xmlns="http://www.loc.gov/MARC21/slim">
          <datafield tag="245" ind1="1" ind2="0">
            <subfield code="a">DNB Book</subfield>
          </datafield>
          <datafield tag="100" ind1="1" ind2=" ">
            <subfield code="a">DNB Author</subfield>
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

	client := &MetadatenClient{
		httpClient: &http.Client{Transport: mockTr},
	}

	ergebnis, err := client.SucheNachISBN(context.Background(), "9783161484100")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if ergebnis.Titel != "DNB Book" {
		t.Errorf("SucheNachISBN(…).Titel = '%s', erwartet 'DNB Book'", ergebnis.Titel)
	}
	if ergebnis.Autor != "DNB Author" {
		t.Errorf("SucheNachISBN(…).Autor = '%s', erwartet 'DNB Author'", ergebnis.Autor)
	}
}

func testSucheISBNGoogle(t *testing.T) {
	mockTr := &mockTransport{
		roundTripFunc: func(anfrage *http.Request) (*http.Response, error) {
			if strings.Contains(anfrage.URL.String(), "services.dnb.de") {
				return &http.Response{
					StatusCode: http.StatusNotFound,
					Body:       io.NopCloser(bytes.NewBufferString("")),
					Header:     make(http.Header),
				}, nil
			}
			if strings.Contains(anfrage.URL.String(), "googleapis.com") {
				googleJSON := `{
					"items": [{
						"volumeInfo": {
							"title": "Google Book",
							"authors": ["Google Author"]
						}
					}]
				}`
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString(googleJSON)),
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

	client := &MetadatenClient{
		httpClient: &http.Client{Transport: mockTr},
	}

	ergebnis, err := client.SucheNachISBN(context.Background(), "9783161484100")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if ergebnis.Titel != "Google Book" {
		t.Errorf("SucheNachISBN(…).Titel = '%s', erwartet 'Google Book'", ergebnis.Titel)
	}
	if ergebnis.Autor != "Google Author" {
		t.Errorf("SucheNachISBN(…).Autor = '%s', erwartet 'Google Author'", ergebnis.Autor)
	}
}

func testSucheISBNOpenLibrary(t *testing.T) {
	mockTr := &mockTransport{
		roundTripFunc: func(anfrage *http.Request) (*http.Response, error) {
			if strings.Contains(anfrage.URL.String(), "services.dnb.de") || strings.Contains(anfrage.URL.String(), "googleapis.com") {
				return &http.Response{
					StatusCode: http.StatusNotFound,
					Body:       io.NopCloser(bytes.NewBufferString("")),
					Header:     make(http.Header),
				}, nil
			}
			if strings.Contains(anfrage.URL.String(), "openlibrary.org") {
				openLibraryJSON := `{
					"ISBN:9783161484100": {
						"title": "OpenLibrary Book",
						"authors": [{"name": "OpenLibrary Author"}]
					}
				}`
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString(openLibraryJSON)),
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

	client := &MetadatenClient{
		httpClient: &http.Client{Transport: mockTr},
	}

	ergebnis, err := client.SucheNachISBN(context.Background(), "9783161484100")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if ergebnis.Titel != "OpenLibrary Book" {
		t.Errorf("SucheNachISBN(…).Titel = '%s', erwartet 'OpenLibrary Book'", ergebnis.Titel)
	}
	if ergebnis.Autor != "OpenLibrary Author" {
		t.Errorf("SucheNachISBN(…).Autor = '%s', erwartet 'OpenLibrary Author'", ergebnis.Autor)
	}
}

func testSucheISBNNotFound(t *testing.T) {
	mockTr := &mockTransport{
		roundTripFunc: func(anfrage *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Body:       io.NopCloser(bytes.NewBufferString("")),
				Header:     make(http.Header),
			}, nil
		},
	}

	client := &MetadatenClient{
		httpClient: &http.Client{Transport: mockTr},
	}

	_, err := client.SucheNachISBN(context.Background(), "9783161484100")
	if err == nil || !strings.Contains(err.Error(), "keine metadaten für ISBN gefunden") {
		t.Errorf("SucheNachISBN(…) = %v, erwartet den Fehler „keine metadaten für ISBN gefunden“", err)
	}
}

func TestNeuerMetadatenClient(t *testing.T) {
	client := NewMetadatenClient()
	if client == nil {
		t.Fatal("Erwartete einen initialisierten MetadatenClient, aber bekam nil")
	}
	if client.httpClient == nil {
		t.Fatal("Erwartete einen initialisierten httpClient im MetadatenClient, aber bekam nil")
	}
}

func TestSetzeHTTPClientFuerTest(t *testing.T) {
	client := NewMetadatenClient()
	customHTTPClient := &http.Client{}

	client.SetzeHTTPClientFuerTest(customHTTPClient)

	if client.httpClient != customHTTPClient {
		t.Errorf("Erwartete, dass der httpClient ueberschrieben wurde")
	}
}
