package inventur

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"testing"
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


func TestSpeichereCoverDatei(t *testing.T) {
	// Clean up uploads dir if it exists
	if err := os.RemoveAll("uploads"); err != nil {
		t.Fatalf("cleanup uploads: %v", err)
	}
	defer func() {
		if err := os.RemoveAll("uploads"); err != nil {
			t.Logf("cleanup uploads: %v", err)
		}
	}()

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

	t.Run("path traversal attempt in isbn is sanitized", func(t *testing.T) {
		res := speichereCoverDatei([]byte("dummy data"), "../../../etc/passwd", ".webp")
		if !strings.HasPrefix(res, "/uploads/cover_auto_passwd_") {
			t.Errorf("expected path traversal to be sanitized to base name, got %s", res)
		}
	})
}

func TestDownloadAndSaveCoverLocally(t *testing.T) {
	ctx := context.Background()
	if err := os.RemoveAll("uploads"); err != nil {
		t.Fatalf("cleanup uploads: %v", err)
	}
	defer func() {
		if err := os.RemoveAll("uploads"); err != nil {
			t.Logf("cleanup uploads: %v", err)
		}
	}()

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
