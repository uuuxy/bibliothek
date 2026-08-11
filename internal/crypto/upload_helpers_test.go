package crypto

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDecryptAndServe(t *testing.T) {
	t.Setenv(SchluesselVariable, "12345678901234567890123456789012")

	t.Run("empty ciphertext", func(t *testing.T) {
		rec := httptest.NewRecorder()
		err := DecryptAndServe(rec, nil, "image/png")
		if err == nil {
			t.Error("expected error for empty ciphertext")
		}
		if rec.Code != http.StatusNotFound {
			t.Errorf("expected 404, got %d", rec.Code)
		}
	})

	t.Run("invalid ciphertext", func(t *testing.T) {
		rec := httptest.NewRecorder()
		err := DecryptAndServe(rec, []byte("invalid"), "image/png")
		if err == nil {
			t.Error("expected error for invalid ciphertext")
		}
		if rec.Code != http.StatusInternalServerError {
			t.Errorf("expected 500, got %d", rec.Code)
		}
	})

	t.Run("success", func(t *testing.T) {
		content := []byte("hello world")
		enc, err := Encrypt(content)
		if err != nil {
			t.Fatalf("unexpected encryption error: %v", err)
		}

		rec := httptest.NewRecorder()
		err = DecryptAndServe(rec, enc, "text/plain")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if rec.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rec.Code)
		}
		if rec.Header().Get("Content-Type") != "text/plain" {
			t.Errorf("expected content type text/plain, got %s", rec.Header().Get("Content-Type"))
		}
		if rec.Header().Get("Cache-Control") != "private, no-cache, no-store, must-revalidate" {
			t.Errorf("unexpected Cache-Control")
		}
		if rec.Body.String() != "hello world" {
			t.Errorf("expected body 'hello world', got %s", rec.Body.String())
		}
	})
}
