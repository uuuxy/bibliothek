package inventur

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestSucheOpenLibrary(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockTr := &mockTransport{
			roundTripFunc: func(anfrage *http.Request) (*http.Response, error) {
				if strings.Contains(anfrage.URL.String(), "openlibrary.org") {
					openLibraryJSON := `{
						"ISBN:9783161484100": {
							"title": "OpenLibrary Book",
							"subtitle": "OpenLibrary Subtitle",
							"authors": [{"name": "OpenLibrary Author"}],
							"cover": {
								"medium": "http://medium.cover",
								"large": "http://large.cover"
							}
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

		ergebnis, err := client.sucheOpenLibrary(context.Background(), "9783161484100")
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if ergebnis.Titel != "OpenLibrary Book" {
			t.Errorf("sucheOpenLibrary(…).Titel = '%s', erwartet 'OpenLibrary Book'", ergebnis.Titel)
		}
		if ergebnis.Untertitel != "OpenLibrary Subtitle" {
			t.Errorf("sucheOpenLibrary(…).Untertitel = '%s', erwartet 'OpenLibrary Subtitle'", ergebnis.Untertitel)
		}
		if ergebnis.Autor != "OpenLibrary Author" {
			t.Errorf("sucheOpenLibrary(…).Autor = '%s', erwartet 'OpenLibrary Author'", ergebnis.Autor)
		}
		if ergebnis.CoverURL != "http://large.cover" {
			t.Errorf("ergebnis.CoverURL = '%s', erwartet 'http://large.cover'", ergebnis.CoverURL)
		}
	})

	t.Run("Medium Cover Fallback", func(t *testing.T) {
		mockTr := &mockTransport{
			roundTripFunc: func(anfrage *http.Request) (*http.Response, error) {
				openLibraryJSON := `{
					"ISBN:9783161484100": {
						"title": "OpenLibrary Book",
						"cover": {
							"medium": "http://medium.cover"
						}
					}
				}`
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString(openLibraryJSON)),
					Header:     make(http.Header),
				}, nil
			},
		}
		client := &MetadatenClient{httpClient: &http.Client{Transport: mockTr}}
		ergebnis, err := client.sucheOpenLibrary(context.Background(), "9783161484100")
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if ergebnis.CoverURL != "http://medium.cover" {
			t.Errorf("sucheOpenLibrary(…).CoverURL = '%s', erwartet das mittlere Cover", ergebnis.CoverURL)
		}
	})

	t.Run("Not Found", func(t *testing.T) {
		mockTr := &mockTransport{
			roundTripFunc: func(anfrage *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString(`{}`)),
					Header:     make(http.Header),
				}, nil
			},
		}

		client := &MetadatenClient{
			httpClient: &http.Client{Transport: mockTr},
		}

		_, err := client.sucheOpenLibrary(context.Background(), "9783161484100")
		if err == nil || !strings.Contains(err.Error(), "nicht gefunden") {
			t.Errorf("sucheOpenLibrary(…) = %v, erwartet den Fehler „nicht gefunden“", err)
		}
	})

	t.Run("HTTP Error", func(t *testing.T) {
		mockTr := &mockTransport{
			roundTripFunc: func(anfrage *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusInternalServerError,
					Body:       io.NopCloser(bytes.NewBufferString("Internal Server Error")),
					Header:     make(http.Header),
				}, nil
			},
		}

		client := &MetadatenClient{
			httpClient: &http.Client{Transport: mockTr},
		}

		_, err := client.sucheOpenLibrary(context.Background(), "9783161484100")
		if err == nil || !strings.Contains(err.Error(), "status 500") {
			t.Errorf("sucheOpenLibrary(…) = %v, erwartet den Fehler mit „status 500“", err)
		}
	})

	t.Run("Invalid JSON", func(t *testing.T) {
		mockTr := &mockTransport{
			roundTripFunc: func(anfrage *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString("{invalid json}")),
					Header:     make(http.Header),
				}, nil
			},
		}

		client := &MetadatenClient{
			httpClient: &http.Client{Transport: mockTr},
		}

		_, err := client.sucheOpenLibrary(context.Background(), "9783161484100")
		if err == nil {
			t.Errorf("sucheOpenLibrary(…) = nil, erwartet einen Fehler beim Lesen des JSON")
		}
	})
}

func TestSucheBeiGoogle(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockTr := &mockTransport{
			roundTripFunc: func(anfrage *http.Request) (*http.Response, error) {
				if strings.Contains(anfrage.URL.String(), "googleapis.com") {
					googleJSON := `{
						"items": [{
							"volumeInfo": {
								"title": "Google Book",
								"subtitle": "Google Subtitle",
								"authors": ["Google Author"],
								"imageLinks": {
									"thumbnail": "http://thumbnail.cover",
									"smallThumbnail": "http://small.thumbnail.cover"
								}
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

		ergebnis, err := client.sucheBeiGoogle(context.Background(), "9783161484100")
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if ergebnis.Titel != "Google Book" {
			t.Errorf("sucheBeiGoogle(…).Titel = '%s', erwartet 'Google Book'", ergebnis.Titel)
		}
		if ergebnis.Untertitel != "Google Subtitle" {
			t.Errorf("sucheBeiGoogle(…).Untertitel = '%s', erwartet 'Google Subtitle'", ergebnis.Untertitel)
		}
		if ergebnis.Autor != "Google Author" {
			t.Errorf("sucheBeiGoogle(…).Autor = '%s', erwartet 'Google Author'", ergebnis.Autor)
		}
		if ergebnis.CoverURL != "https://thumbnail.cover" {
			t.Errorf("ergebnis.CoverURL = '%s', erwartet 'https://thumbnail.cover'", ergebnis.CoverURL)
		}
	})

	t.Run("Small Thumbnail Fallback", func(t *testing.T) {
		mockTr := &mockTransport{
			roundTripFunc: func(anfrage *http.Request) (*http.Response, error) {
				googleJSON := `{
					"items": [{
						"volumeInfo": {
							"title": "Google Book",
							"imageLinks": {
								"smallThumbnail": "http://small.thumbnail.cover"
							}
						}
					}]
				}`
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString(googleJSON)),
					Header:     make(http.Header),
				}, nil
			},
		}
		client := &MetadatenClient{httpClient: &http.Client{Transport: mockTr}}
		ergebnis, err := client.sucheBeiGoogle(context.Background(), "9783161484100")
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if ergebnis.CoverURL != "https://small.thumbnail.cover" {
			t.Errorf("sucheBeiGoogle(…).CoverURL = '%s', erwartet das kleine Vorschaubild", ergebnis.CoverURL)
		}
	})

	t.Run("Not Found", func(t *testing.T) {
		mockTr := &mockTransport{
			roundTripFunc: func(anfrage *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString(`{"items": []}`)),
					Header:     make(http.Header),
				}, nil
			},
		}

		client := &MetadatenClient{
			httpClient: &http.Client{Transport: mockTr},
		}

		_, err := client.sucheBeiGoogle(context.Background(), "9783161484100")
		if err == nil || !strings.Contains(err.Error(), "nicht gefunden") {
			t.Errorf("sucheBeiGoogle(…) = %v, erwartet den Fehler „nicht gefunden“", err)
		}
	})

	t.Run("HTTP Error", func(t *testing.T) {
		mockTr := &mockTransport{
			roundTripFunc: func(anfrage *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusNotFound,
					Body:       io.NopCloser(bytes.NewBufferString("Not Found")),
					Header:     make(http.Header),
				}, nil
			},
		}

		client := &MetadatenClient{
			httpClient: &http.Client{Transport: mockTr},
		}

		_, err := client.sucheBeiGoogle(context.Background(), "9783161484100")
		if err == nil || !strings.Contains(err.Error(), "status 404") {
			t.Errorf("sucheBeiGoogle(…) = %v, erwartet den Fehler mit „status 404“", err)
		}
	})

	t.Run("Invalid JSON", func(t *testing.T) {
		mockTr := &mockTransport{
			roundTripFunc: func(anfrage *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString("{invalid json}")),
					Header:     make(http.Header),
				}, nil
			},
		}

		client := &MetadatenClient{
			httpClient: &http.Client{Transport: mockTr},
		}

		_, err := client.sucheBeiGoogle(context.Background(), "9783161484100")
		if err == nil {
			t.Errorf("sucheBeiGoogle(…) = nil, erwartet einen Fehler beim Lesen des JSON")
		}
	})
}
