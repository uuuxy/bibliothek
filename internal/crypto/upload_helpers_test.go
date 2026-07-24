package crypto

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

type dummyFile struct {
	*bytes.Reader
	readErr  error
	closeErr error
}

func (f *dummyFile) Read(p []byte) (n int, err error) {
	if f.readErr != nil {
		return 0, f.readErr
	}
	return f.Reader.Read(p)
}

func (f *dummyFile) Close() error {
	return f.closeErr
}

func TestEncryptUpload(t *testing.T) {
	t.Setenv("ENCRYPTION_KEY", "12345678901234567890123456789012")

	t.Run("nil file", func(t *testing.T) {
		_, err := EncryptUpload(nil)
		if err == nil || err.Error() != "leere Datei übergeben" {
			t.Errorf("expected error 'leere Datei übergeben', got %v", err)
		}
	})

	t.Run("read error", func(t *testing.T) {
		f := &dummyFile{
			Reader:  bytes.NewReader(nil),
			readErr: errors.New("read error"),
		}
		_, err := EncryptUpload(f)
		if err == nil {
			t.Errorf("expected read error, got nil")
		}
	})

	t.Run("success", func(t *testing.T) {
		content := []byte("hello world")
		f := &dummyFile{
			Reader: bytes.NewReader(content),
		}
		enc, err := EncryptUpload(f)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		dec, err := Decrypt(enc)
		if err != nil {
			t.Fatalf("unexpected decryption error: %v", err)
		}
		if !bytes.Equal(dec, content) {
			t.Errorf("expected %q, got %q", content, dec)
		}
	})
}

func TestDecryptAndServe(t *testing.T) {
	t.Setenv("ENCRYPTION_KEY", "12345678901234567890123456789012")

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
