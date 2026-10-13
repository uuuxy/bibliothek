package crypto

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSchreibeEntschluesselt(t *testing.T) {
	t.Setenv(SchluesselVariable, "12345678901234567890123456789012")

	t.Run("leer", func(t *testing.T) {
		rec := httptest.NewRecorder()
		err := SchreibeEntschluesselt(rec, nil, "image/png")
		if err == nil {
			t.Error("leere Daten: erwartet einen Fehler")
		}
		if rec.Code != http.StatusNotFound {
			t.Errorf("Status %d, erwartet 404", rec.Code)
		}
	})

	t.Run("nicht zu entschlüsseln", func(t *testing.T) {
		rec := httptest.NewRecorder()
		err := SchreibeEntschluesselt(rec, []byte("invalid"), "image/png")
		if err == nil {
			t.Error("unlesbare Daten: erwartet einen Fehler")
		}
		if rec.Code != http.StatusInternalServerError {
			t.Errorf("Status %d, erwartet 500", rec.Code)
		}
	})

	t.Run("gültig", func(t *testing.T) {
		inhalt := []byte("hello world")
		geheimtext, err := Verschluessele(inhalt)
		if err != nil {
			t.Fatalf("Verschluessele: %v", err)
		}

		rec := httptest.NewRecorder()
		err = SchreibeEntschluesselt(rec, geheimtext, "text/plain")
		if err != nil {
			t.Fatalf("SchreibeEntschluesselt: %v", err)
		}

		if rec.Code != http.StatusOK {
			t.Errorf("Status %d, erwartet 200", rec.Code)
		}
		if rec.Header().Get("Content-Type") != "text/plain" {
			t.Errorf("Content-Type %s, erwartet text/plain", rec.Header().Get("Content-Type"))
		}
		if rec.Header().Get("Cache-Control") != "private, no-cache, no-store, must-revalidate" {
			t.Errorf("Cache-Control weicht ab")
		}
		if rec.Body.String() != "hello world" {
			t.Errorf("Inhalt %s, erwartet 'hello world'", rec.Body.String())
		}
	})
}
