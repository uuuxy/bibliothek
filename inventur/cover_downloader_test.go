package inventur

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/color"
	"image/png"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"bibliothek/pkg/coverquelle"
)

func TestLadeCoverBytes(t *testing.T) {
	ctx := context.Background()

	t.Run("empty url", func(t *testing.T) {
		client := &http.Client{}
		ergebnis := ladeCoverBytes(ctx, client, "", coverquelle.IstErsatzbild)
		if ergebnis != nil {
			t.Errorf("ladeCoverBytes(…) = %v, erwartet nil", ergebnis)
		}
	})

	t.Run("openLibrary placeholder url", func(t *testing.T) {
		client := &http.Client{}
		ergebnis := ladeCoverBytes(ctx, client, openLibraryLeeresCover, coverquelle.IstErsatzbild)
		if ergebnis != nil {
			t.Errorf("ladeCoverBytes(…) = %v, erwartet nil", ergebnis)
		}
	})

	t.Run("invalid url format", func(t *testing.T) {
		client := &http.Client{}
		ergebnis := ladeCoverBytes(ctx, client, "http://%42", coverquelle.IstErsatzbild)
		if ergebnis != nil {
			t.Errorf("ladeCoverBytes(…) = %v, erwartet nil", ergebnis)
		}
	})

	t.Run("disallowed host", func(t *testing.T) {
		client := &http.Client{}
		ergebnis := ladeCoverBytes(ctx, client, "https://evil.com/img.png", coverquelle.IstErsatzbild)
		if ergebnis != nil {
			t.Errorf("ladeCoverBytes(…) = %v, erwartet nil", ergebnis)
		}
	})

	t.Run("client error", func(t *testing.T) {
		client := &http.Client{
			Transport: &mockTransport{
				roundTripFunc: func(anfrage *http.Request) (*http.Response, error) {
					return nil, errors.New("network error")
				},
			},
		}
		ergebnis := ladeCoverBytes(ctx, client, "https://covers.openlibrary.org/b/id/1-L.jpg", coverquelle.IstErsatzbild)
		if ergebnis != nil {
			t.Errorf("ladeCoverBytes(…) = %v, erwartet nil", ergebnis)
		}
	})

	t.Run("non 200 status code", func(t *testing.T) {
		client := &http.Client{
			Transport: &mockTransport{
				roundTripFunc: func(anfrage *http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: http.StatusNotFound,
						Body:       io.NopCloser(bytes.NewBufferString("not found")),
					}, nil
				},
			},
		}
		ergebnis := ladeCoverBytes(ctx, client, "https://covers.openlibrary.org/b/id/1-L.jpg", coverquelle.IstErsatzbild)
		if ergebnis != nil {
			t.Errorf("ladeCoverBytes(…) = %v, erwartet nil", ergebnis)
		}
	})

	t.Run("bot protection html response", func(t *testing.T) {
		client := &http.Client{
			Transport: &mockTransport{
				roundTripFunc: func(anfrage *http.Request) (*http.Response, error) {
					antwort := &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(bytes.NewBufferString("<html><head>bot check</head></html>")),
						Header:     make(http.Header),
					}
					antwort.Header.Set("Content-Type", "text/html; charset=utf-8")
					return antwort, nil
				},
			},
		}
		ergebnis := ladeCoverBytes(ctx, client, "https://covers.openlibrary.org/b/id/1-L.jpg", coverquelle.IstErsatzbild)
		if ergebnis != nil {
			t.Errorf("ladeCoverBytes(…) = %v, erwartet nil", ergebnis)
		}
	})

	t.Run("valid image response", func(t *testing.T) {
		expectedBytes := []byte("fake_image_data")
		client := &http.Client{
			Transport: &mockTransport{
				roundTripFunc: func(anfrage *http.Request) (*http.Response, error) {
					antwort := &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(bytes.NewReader(expectedBytes)),
						Header:     make(http.Header),
					}
					antwort.Header.Set("Content-Type", "image/jpeg")
					return antwort, nil
				},
			},
		}
		ergebnis := ladeCoverBytes(ctx, client, "https://covers.openlibrary.org/b/id/1-L.jpg", coverquelle.IstErsatzbild)
		if !bytes.Equal(ergebnis, expectedBytes) {
			t.Errorf("ladeCoverBytes(…) = %v, erwartet %v", ergebnis, expectedBytes)
		}
	})
}

// Ein Ersatzbild der Quelle („image not available") ist kein Cover: Die Abfrage geht zur
// nächsten Quelle, statt es am Titel abzulegen.
func TestLadeCoverBytes_ErsatzbildIstKeinCover(t *testing.T) {
	bild := []byte("das ersatzbild der quelle")
	client := &http.Client{Transport: &mockTransport{
		roundTripFunc: func(*http.Request) (*http.Response, error) {
			antwort := &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(bytes.NewReader(bild)), Header: make(http.Header)}
			antwort.Header.Set("Content-Type", "image/png")
			return antwort, nil
		},
	}}
	const adresse = "https://books.google.com/books/content?vid=ISBN:9783551551672&printsec=frontcover&img=1&zoom=1"

	if got := ladeCoverBytes(context.Background(), client, adresse, coverquelle.IstErsatzbild); !bytes.Equal(got, bild) {
		t.Fatalf("ein gewöhnliches Bild kommt nicht an: %q", got)
	}
	istErsatzbild := func(antwort []byte) bool { return bytes.Equal(antwort, bild) }
	if got := ladeCoverBytes(context.Background(), client, adresse, istErsatzbild); got != nil {
		t.Errorf("das Ersatzbild kommt als Cover zurück (%d Bytes)", len(got))
	}
}

func TestSpeichereCoverDatei(t *testing.T) {
	// Im vorübergehenden Verzeichnis, nicht im Verzeichnis des Pakets: Ein Test, der
	// inventur/uploads im Repository anlegt und wieder löscht, stört die Ratschen im Wurzelpaket,
	// die parallel den Baum lesen (WalkDir) und über das verschwundene Verzeichnis stolpern
	// („open inventur/uploads: no such file").
	imTestVerzeichnis(t)

	t.Run("successful save", func(t *testing.T) {
		ergebnis := speichereCoverDatei([]byte("dummy data"), "1234567890", ".webp")
		if !strings.HasPrefix(ergebnis, "/uploads/cover_auto_1234567890_") {
			t.Errorf("speichereCoverDatei(…) = %s, erwartet den Anfang /uploads/cover_auto_1234567890_", ergebnis)
		}
		if !strings.HasSuffix(ergebnis, ".webp") {
			t.Errorf("speichereCoverDatei(…) = %s, erwartet die Endung .webp", ergebnis)
		}

		// Die Datei muss bestehen.
		lokalerPfad := filepath.Join(".", ergebnis)
		if _, err := os.Stat(lokalerPfad); os.IsNotExist(err) {
			t.Errorf("Datei %s fehlt nach speichereCoverDatei(…)", lokalerPfad)
		}
	})

	// Der Name kommt aus dem Inhalt: Dieselbe Abfrage legt keine zweite Datei ab, ein
	// anderes Bild bekommt einen anderen Namen.
	t.Run("derselbe Inhalt, dieselbe Datei", func(t *testing.T) {
		erste := speichereCoverDatei([]byte("bild eins"), "9783551551672", ".webp")
		wieder := speichereCoverDatei([]byte("bild eins"), "9783551551672", ".webp")
		anderes := speichereCoverDatei([]byte("bild zwei"), "9783551551672", ".webp")
		if erste == "" || erste != wieder {
			t.Errorf("zweimal derselbe Inhalt: %q und %q", erste, wieder)
		}
		if anderes == erste {
			t.Errorf("ein anderes Bild trägt denselben Namen %q", anderes)
		}
		dateien, err := filepath.Glob(filepath.Join("uploads", "cover_auto_9783551551672_*"))
		if err != nil || len(dateien) != 2 {
			t.Errorf("Dateien zur ISBN: %v (%v), erwartet zwei", dateien, err)
		}
	})

	t.Run("path traversal attempt in isbn is sanitized", func(t *testing.T) {
		ergebnis := speichereCoverDatei([]byte("dummy data"), "../../../etc/passwd", ".webp")
		if !strings.HasPrefix(ergebnis, "/uploads/cover_auto_passwd_") {
			t.Errorf("speichereCoverDatei(…, \"../../../etc/passwd\", …) = %s, erwartet den bereinigten Namen /uploads/cover_auto_passwd_…", ergebnis)
		}
	})
}

func TestSpeichereCoverLokal(t *testing.T) {
	ctx := context.Background()
	imTestVerzeichnis(t) // siehe TestSpeichereCoverDatei

	t.Run("invalid image data fails decode", func(t *testing.T) {
		client := &http.Client{
			Transport: &mockTransport{
				roundTripFunc: func(anfrage *http.Request) (*http.Response, error) {
					antwort := &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(bytes.NewReader([]byte("not an image"))),
						Header:     make(http.Header),
					}
					antwort.Header.Set("Content-Type", "image/jpeg")
					return antwort, nil
				},
			},
		}
		ergebnis := speichereCoverLokal(ctx, client, "https://covers.openlibrary.org/b/id/1-L.jpg", "123")
		if ergebnis != "" {
			t.Errorf("speichereCoverLokal(…) = %s, erwartet leer", ergebnis)
		}
	})

	t.Run("small image is ignored", func(t *testing.T) {
		bild := image.NewRGBA(image.Rect(0, 0, 5, 5))
		var puffer bytes.Buffer
		if err := png.Encode(&puffer, bild); err != nil {
			t.Fatalf("png encode: %v", err)
		}

		client := &http.Client{
			Transport: &mockTransport{
				roundTripFunc: func(anfrage *http.Request) (*http.Response, error) {
					antwort := &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(bytes.NewReader(puffer.Bytes())),
						Header:     make(http.Header),
					}
					antwort.Header.Set("Content-Type", "image/png")
					return antwort, nil
				},
			},
		}
		ergebnis := speichereCoverLokal(ctx, client, "https://covers.openlibrary.org/b/id/1-L.jpg", "123")
		if ergebnis != "" {
			t.Errorf("speichereCoverLokal(…) bei kleinem Bild = %s, erwartet leer", ergebnis)
		}
	})

	t.Run("valid large image is saved", func(t *testing.T) {
		bild := image.NewRGBA(image.Rect(0, 0, 15, 15))
		for x := 0; x < 15; x++ {
			for y := 0; y < 15; y++ {
				bild.Set(x, y, color.RGBA{255, 0, 0, 255})
			}
		}
		var puffer bytes.Buffer
		if err := png.Encode(&puffer, bild); err != nil {
			t.Fatalf("png encode: %v", err)
		}

		client := &http.Client{
			Transport: &mockTransport{
				roundTripFunc: func(anfrage *http.Request) (*http.Response, error) {
					antwort := &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(bytes.NewReader(puffer.Bytes())),
						Header:     make(http.Header),
					}
					antwort.Header.Set("Content-Type", "image/png")
					return antwort, nil
				},
			},
		}
		ergebnis := speichereCoverLokal(ctx, client, "https://covers.openlibrary.org/b/id/1-L.jpg", "123")
		if !strings.HasPrefix(ergebnis, "/uploads/cover_auto_123_") {
			t.Errorf("speichereCoverLokal(…) = %s, erwartet den Anfang /uploads/cover_auto_123_", ergebnis)
		}
	})
}
