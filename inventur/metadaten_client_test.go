package inventur

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

type mockRoundTripper struct {
	roundTripFunc func(req *http.Request) (*http.Response, error)
}

func (m *mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return m.roundTripFunc(req)
}

func TestSucheNachISBN(t *testing.T) {
	// Setup mock responses
	dnbResponse := `<?xml version="1.0" encoding="UTF-8"?>
<searchRetrieveResponse xmlns="http://www.loc.gov/zing/srw/">
  <records>
    <record>
      <recordData>
        <record xmlns="http://www.loc.gov/MARC21/slim">
          <datafield tag="245" ind1="1" ind2="0">
            <subfield code="a">DNB Title</subfield>
          </datafield>
          <datafield tag="100" ind1="1" ind2=" ">
            <subfield code="a">DNB Author</subfield>
          </datafield>
        </record>
      </recordData>
    </record>
  </records>
</searchRetrieveResponse>`

	googleResponse := `{
  "items": [
    {
      "volumeInfo": {
        "title": "Google Title",
        "authors": ["Google Author"]
      }
    }
  ]
}`

	openLibraryResponse := `{
  "ISBN:9781234567897": {
    "title": "OpenLibrary Title",
    "authors": [{"name": "OpenLibrary Author"}]
  }
}`

	tests := []struct {
		name          string
		isbn          string
		mockResponses map[string]string // URL substring -> response body
		expectedError string
		expectedTitle string
		expectedAuth  string
	}{
		{
			name:          "Invalid ISBN",
			isbn:          "invalid-isbn",
			expectedError: "ungültiges ISBN format: sicherheitsabbruch",
		},
		{
			name: "DNB Success",
			isbn: "9781234567897",
			mockResponses: map[string]string{
				"services.dnb.de": dnbResponse,
				// Head request for DNB cover
				"portal.dnb.de": "",
			},
			expectedTitle: "DNB Title",
			expectedAuth:  "DNB Author",
		},
		{
			name: "Google Books Success (DNB fails)",
			isbn: "9781234567897",
			mockResponses: map[string]string{
				"services.dnb.de": `<?xml version="1.0"?><searchRetrieveResponse><records></records></searchRetrieveResponse>`,
				"googleapis.com":  googleResponse,
				"portal.dnb.de": "",
			},
			expectedTitle: "Google Title",
			expectedAuth:  "Google Author",
		},
		{
			name: "OpenLibrary Success (DNB and Google fail)",
			isbn: "9781234567897",
			mockResponses: map[string]string{
				"services.dnb.de": `<?xml version="1.0"?><searchRetrieveResponse><records></records></searchRetrieveResponse>`,
				"googleapis.com":  `{"items": []}`,
				"openlibrary.org": openLibraryResponse,
				"portal.dnb.de": "",
			},
			expectedTitle: "OpenLibrary Title",
			expectedAuth:  "OpenLibrary Author",
		},
		{
			name: "All APIs Fail",
			isbn: "9781234567897",
			mockResponses: map[string]string{
				"services.dnb.de": `<?xml version="1.0"?><searchRetrieveResponse><records></records></searchRetrieveResponse>`,
				"googleapis.com":  `{"items": []}`,
				"openlibrary.org": `{}`,
			},
			expectedError: "keine metadaten für ISBN gefunden",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NeuerMetadatenClient()

			// Inject mock HTTP client
			client.httpClient = &http.Client{
				Transport: &mockRoundTripper{
					roundTripFunc: func(req *http.Request) (*http.Response, error) {
						for urlSubstr, respBody := range tt.mockResponses {
							if strings.Contains(req.URL.String(), urlSubstr) {
								if req.Method == http.MethodHead {
									return &http.Response{
										StatusCode: http.StatusNotFound,
										Body:       io.NopCloser(bytes.NewBufferString("")),
									}, nil
								}
								return &http.Response{
									StatusCode: http.StatusOK,
									Body:       io.NopCloser(bytes.NewBufferString(respBody)),
								}, nil
							}
						}
						return &http.Response{
							StatusCode: http.StatusNotFound,
							Body:       io.NopCloser(bytes.NewBufferString("Not Found")),
						}, nil
					},
				},
			}

			result, err := client.SucheNachISBN(context.Background(), tt.isbn)

			if tt.expectedError != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.expectedError)
				}
				if !strings.Contains(err.Error(), tt.expectedError) {
					t.Errorf("expected error containing %q, got %q", tt.expectedError, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result.Titel != tt.expectedTitle {
				t.Errorf("expected title %q, got %q", tt.expectedTitle, result.Titel)
			}

			if result.Autor != tt.expectedAuth {
				t.Errorf("expected author %q, got %q", tt.expectedAuth, result.Autor)
			}
		})
	}
}
