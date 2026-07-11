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
	roundTripFunc func(req *http.Request) (*http.Response, error)
}

func (m *mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return m.roundTripFunc(req)
}

func TestSucheNachISBN(t *testing.T) {
	t.Run("Invalid ISBN", func(t *testing.T) {
		client := &MetadatenClient{}
		_, err := client.SucheNachISBN(context.Background(), "invalid")
		if err == nil || !strings.Contains(err.Error(), "ungültiges ISBN format") {
			t.Errorf("Expected invalid ISBN error, got: %v", err)
		}
	})

	t.Run("Valid ISBN, Found in DNB", func(t *testing.T) {
		mockTr := &mockTransport{
			roundTripFunc: func(req *http.Request) (*http.Response, error) {
				if strings.Contains(req.URL.String(), "services.dnb.de") {
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

		res, err := client.SucheNachISBN(context.Background(), "9783161484100")
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if res.Titel != "DNB Book" {
			t.Errorf("Expected title 'DNB Book', got '%s'", res.Titel)
		}
		if res.Autor != "DNB Author" {
			t.Errorf("Expected author 'DNB Author', got '%s'", res.Autor)
		}
	})

	t.Run("Valid ISBN, Found in Google Books", func(t *testing.T) {
		mockTr := &mockTransport{
			roundTripFunc: func(req *http.Request) (*http.Response, error) {
				if strings.Contains(req.URL.String(), "services.dnb.de") {
					return &http.Response{
						StatusCode: http.StatusNotFound,
						Body:       io.NopCloser(bytes.NewBufferString("")),
						Header:     make(http.Header),
					}, nil
				}
				if strings.Contains(req.URL.String(), "googleapis.com") {
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

		res, err := client.SucheNachISBN(context.Background(), "9783161484100")
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if res.Titel != "Google Book" {
			t.Errorf("Expected title 'Google Book', got '%s'", res.Titel)
		}
		if res.Autor != "Google Author" {
			t.Errorf("Expected author 'Google Author', got '%s'", res.Autor)
		}
	})

	t.Run("Valid ISBN, Found in OpenLibrary", func(t *testing.T) {
		mockTr := &mockTransport{
			roundTripFunc: func(req *http.Request) (*http.Response, error) {
				if strings.Contains(req.URL.String(), "services.dnb.de") || strings.Contains(req.URL.String(), "googleapis.com") {
					return &http.Response{
						StatusCode: http.StatusNotFound,
						Body:       io.NopCloser(bytes.NewBufferString("")),
						Header:     make(http.Header),
					}, nil
				}
				if strings.Contains(req.URL.String(), "openlibrary.org") {
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

		res, err := client.SucheNachISBN(context.Background(), "9783161484100")
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if res.Titel != "OpenLibrary Book" {
			t.Errorf("Expected title 'OpenLibrary Book', got '%s'", res.Titel)
		}
		if res.Autor != "OpenLibrary Author" {
			t.Errorf("Expected author 'OpenLibrary Author', got '%s'", res.Autor)
		}
	})

	t.Run("Valid ISBN, Not Found Anywhere", func(t *testing.T) {
		mockTr := &mockTransport{
			roundTripFunc: func(req *http.Request) (*http.Response, error) {
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
			t.Errorf("Expected not found error, got: %v", err)
		}
	})
}
