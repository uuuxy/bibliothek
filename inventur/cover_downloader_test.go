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
		res := ladeCoverBytes(ctx, client, "")
		if res != nil {
			t.Errorf("expected nil, got %v", res)
		}
	})

	t.Run("openLibrary placeholder url", func(t *testing.T) {
		client := &http.Client{}
		res := ladeCoverBytes(ctx, client, openLibraryLeeresCover)
		if res != nil {
			t.Errorf("expected nil, got %v", res)
		}
	})

	t.Run("invalid url format", func(t *testing.T) {
		client := &http.Client{}
		res := ladeCoverBytes(ctx, client, "http://%42")
		if res != nil {
			t.Errorf("expected nil, got %v", res)
		}
	})

	t.Run("disallowed host", func(t *testing.T) {
		client := &http.Client{}
		res := ladeCoverBytes(ctx, client, "https://evil.com/img.png")
		if res != nil {
			t.Errorf("expected nil, got %v", res)
		}
	})

	t.Run("client error", func(t *testing.T) {
		client := &http.Client{
			Transport: &mockTransport{
				roundTripFunc: func(req *http.Request) (*http.Response, error) {
					return nil, errors.New("network error")
				},
			},
		}
		res := ladeCoverBytes(ctx, client, "https://covers.openlibrary.org/b/id/1-L.jpg")
		if res != nil {
			t.Errorf("expected nil, got %v", res)
		}
	})

	t.Run("non 200 status code", func(t *testing.T) {
		client := &http.Client{
			Transport: &mockTransport{
				roundTripFunc: func(req *http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: http.StatusNotFound,
						Body:       io.NopCloser(bytes.NewBufferString("not found")),
					}, nil
				},
			},
		}
		res := ladeCoverBytes(ctx, client, "https://covers.openlibrary.org/b/id/1-L.jpg")
		if res != nil {
			t.Errorf("expected nil, got %v", res)
		}
	})

	t.Run("bot protection html response", func(t *testing.T) {
		client := &http.Client{
			Transport: &mockTransport{
				roundTripFunc: func(req *http.Request) (*http.Response, error) {
					resp := &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(bytes.NewBufferString("<html><head>bot check</head></html>")),
						Header:     make(http.Header),
					}
					resp.Header.Set("Content-Type", "text/html; charset=utf-8")
					return resp, nil
				},
			},
		}
		res := ladeCoverBytes(ctx, client, "https://covers.openlibrary.org/b/id/1-L.jpg")
		if res != nil {
			t.Errorf("expected nil, got %v", res)
		}
	})

	t.Run("valid image response", func(t *testing.T) {
		expectedBytes := []byte("fake_image_data")
		client := &http.Client{
			Transport: &mockTransport{
				roundTripFunc: func(req *http.Request) (*http.Response, error) {
					resp := &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(bytes.NewReader(expectedBytes)),
						Header:     make(http.Header),
					}
					resp.Header.Set("Content-Type", "image/jpeg")
					return resp, nil
				},
			},
		}
		res := ladeCoverBytes(ctx, client, "https://covers.openlibrary.org/b/id/1-L.jpg")
		if !bytes.Equal(res, expectedBytes) {
			t.Errorf("expected %v, got %v", expectedBytes, res)
		}
	})
}

// Ein Ersatzbild der Quelle („image not available") ist kein Cover: Die Abfrage geht zur
// nächsten Quelle, statt es am Titel abzulegen.
func TestLadeCoverBytes_ErsatzbildIstKeinCover(t *testing.T) {
	bild := []byte("das ersatzbild der quelle")
	client := &http.Client{Transport: &mockTransport{
		roundTripFunc: func(*http.Request) (*http.Response, error) {
			resp := &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(bytes.NewReader(bild)), Header: make(http.Header)}
			resp.Header.Set("Content-Type", "image/png")
			return resp, nil
		},
	}}
	const adresse = "https://books.google.com/books/content?vid=ISBN:9783551551672&printsec=frontcover&img=1&zoom=1"

	if got := ladeCoverBytes(context.Background(), client, adresse); !bytes.Equal(got, bild) {
		t.Fatalf("ein gewöhnliches Bild kommt nicht an: %q", got)
	}
	t.Cleanup(coverquelle.MerkeErsatzbildFuerTest(bild))
	if got := ladeCoverBytes(context.Background(), client, adresse); got != nil {
		t.Errorf("das Ersatzbild kommt als Cover zurück (%d Bytes)", len(got))
	}
}

func TestSpeichereCoverDatei(t *testing.T) {
	// Im temporären Verzeichnis, nicht im Paketverzeichnis: Bis zum 21.09.2026 legte
	// der Test inventur/uploads im Repo an und räumte es per RemoveAll wieder weg —
	// während die Ratschen im Wurzelpaket (WalkDir) parallel den Baum lasen und über
	// das verschwundene Verzeichnis stolperten („open inventur/uploads: no such file").
	imTestVerzeichnis(t)

	t.Run("successful save", func(t *testing.T) {
		res := speichereCoverDatei([]byte("dummy data"), "1234567890", ".webp")
		if !strings.HasPrefix(res, "/uploads/cover_auto_1234567890_") {
			t.Errorf("expected path to start with /uploads/cover_auto_1234567890_, got %s", res)
		}
		if !strings.HasSuffix(res, ".webp") {
			t.Errorf("expected path to end with .webp, got %s", res)
		}

		// Verify file exists
		localPath := filepath.Join(".", res)
		if _, err := os.Stat(localPath); os.IsNotExist(err) {
			t.Errorf("expected file %s to exist, but it does not", localPath)
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
		res := speichereCoverDatei([]byte("dummy data"), "../../../etc/passwd", ".webp")
		if !strings.HasPrefix(res, "/uploads/cover_auto_passwd_") {
			t.Errorf("expected path traversal to be sanitized to base name, got %s", res)
		}
	})
}

func TestDownloadAndSaveCoverLocally(t *testing.T) {
	ctx := context.Background()
	imTestVerzeichnis(t) // siehe TestSpeichereCoverDatei

	t.Run("invalid image data fails decode", func(t *testing.T) {
		client := &http.Client{
			Transport: &mockTransport{
				roundTripFunc: func(req *http.Request) (*http.Response, error) {
					resp := &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(bytes.NewReader([]byte("not an image"))),
						Header:     make(http.Header),
					}
					resp.Header.Set("Content-Type", "image/jpeg")
					return resp, nil
				},
			},
		}
		res := downloadAndSaveCoverLocally(ctx, client, "https://covers.openlibrary.org/b/id/1-L.jpg", "123")
		if res != "" {
			t.Errorf("expected empty string, got %s", res)
		}
	})

	t.Run("small image is ignored", func(t *testing.T) {
		img := image.NewRGBA(image.Rect(0, 0, 5, 5))
		var buf bytes.Buffer
		if err := png.Encode(&buf, img); err != nil {
			t.Fatalf("png encode: %v", err)
		}

		client := &http.Client{
			Transport: &mockTransport{
				roundTripFunc: func(req *http.Request) (*http.Response, error) {
					resp := &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(bytes.NewReader(buf.Bytes())),
						Header:     make(http.Header),
					}
					resp.Header.Set("Content-Type", "image/png")
					return resp, nil
				},
			},
		}
		res := downloadAndSaveCoverLocally(ctx, client, "https://covers.openlibrary.org/b/id/1-L.jpg", "123")
		if res != "" {
			t.Errorf("expected empty string for small image, got %s", res)
		}
	})

	t.Run("valid large image is saved", func(t *testing.T) {
		img := image.NewRGBA(image.Rect(0, 0, 15, 15))
		for x := 0; x < 15; x++ {
			for y := 0; y < 15; y++ {
				img.Set(x, y, color.RGBA{255, 0, 0, 255})
			}
		}
		var buf bytes.Buffer
		if err := png.Encode(&buf, img); err != nil {
			t.Fatalf("png encode: %v", err)
		}

		client := &http.Client{
			Transport: &mockTransport{
				roundTripFunc: func(req *http.Request) (*http.Response, error) {
					resp := &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(bytes.NewReader(buf.Bytes())),
						Header:     make(http.Header),
					}
					resp.Header.Set("Content-Type", "image/png")
					return resp, nil
				},
			},
		}
		res := downloadAndSaveCoverLocally(ctx, client, "https://covers.openlibrary.org/b/id/1-L.jpg", "123")
		if !strings.HasPrefix(res, "/uploads/cover_auto_123_") {
			t.Errorf("expected path to start with /uploads/cover_auto_123_, got %s", res)
		}
	})
}
