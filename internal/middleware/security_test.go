package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSecurityHeadersMiddleware(t *testing.T) {
	// Create a mock next handler that simply sets a dummy status and body
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("OK"))
		if err != nil {
			t.Fatalf("Failed to write response: %v", err)
		}
	})

	// Wrap the next handler with the middleware being tested
	middleware := SecurityHeadersMiddleware(nextHandler)

	// Create a mock request and response recorder
	req := httptest.NewRequest(http.MethodGet, "http://example.com/foo", nil)
	rr := httptest.NewRecorder()

	// Execute the middleware
	middleware.ServeHTTP(rr, req)

	// Define expected headers and their expected values
	expectedHeaders := map[string]string{
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":        "DENY",
		"X-XSS-Protection":       "1; mode=block",
		// base-uri und object-src sind seit dem 04.08.2026 dabei. base-uri fällt NICHT auf
		// default-src zurück — ohne die Direktive kann ein eingeschleustes <base href>
		// jede relative URL der SPA umbiegen.
		// img-src ohne https: seit dem 06.08.2026: Bilder von beliebigen fremden Hosts
		// waren der Abflusskanal für eingeschleustes Markup (gemessen im Druckfenster).
		// Cover laufen seitdem ausnahmslos über den eigenen Proxy.
		"Content-Security-Policy":   "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; font-src 'self'; img-src 'self' data: blob:; connect-src 'self'; frame-ancestors 'none'; form-action 'self'; base-uri 'self'; object-src 'none';",
		"Strict-Transport-Security": "max-age=31536000; includeSubDomains",
		"Referrer-Policy":           "strict-origin-when-cross-origin",
		"Permissions-Policy":        "geolocation=(), microphone=(), camera=(self)",
	}

	// Verify that each expected header is present and has the correct value
	for header, expectedValue := range expectedHeaders {
		t.Run(header, func(t *testing.T) {
			actualValue := rr.Header().Get(header)
			if actualValue == "" {
				t.Errorf("Expected header %s to be set, but it was missing", header)
			} else if actualValue != expectedValue {
				t.Errorf("Expected header %s to have value %q, but got %q", header, expectedValue, actualValue)
			}
		})
	}

	// Verify that the next handler was executed correctly
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Expected HTTP status %v, got %v", http.StatusOK, status)
	}
	if body := rr.Body.String(); body != "OK" {
		t.Errorf("Expected response body %q, got %q", "OK", body)
	}
}

// Antworten unter /api/ legt der Browser nicht ab (seit dem 30.09.2026, OFFEN.md 5.29).
// Die Programmdateien der Oberfläche bleiben unberührt, und ein Handler, der ablegen lassen
// will, behält seinen eigenen Kopf — die Vorgabe steht vor ihm, nicht nach ihm.
func TestSecurityHeadersMiddleware_SchnittstelleWirdNichtAbgelegt(t *testing.T) {
	faelle := []struct {
		name, pfad, eigenerKopf, erwartet string
	}{
		{"Liste", "/api/schueler", "", "no-store"},
		{"Auskunft als PDF", "/api/schueler/5f0c9a1e-2b7d-4c1a-9d3e-7a1b2c3d4e5f/dsgvo-auskunft/pdf", "", "no-store"},
		{"Oberfläche", "/", "", ""},
		{"Programmdatei", "/assets/index-4f2a.js", "", ""},
		{"Pfad beginnt nur mit api", "/apidoc", "", ""},
		{"eigener Kopf des Handlers", "/api/images/cover", "public, max-age=86400", "public, max-age=86400"},
	}
	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			handler := SecurityHeadersMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if f.eigenerKopf != "" {
					w.Header().Set("Cache-Control", f.eigenerKopf)
				}
				w.WriteHeader(http.StatusOK)
			}))
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "http://example.com"+f.pfad, nil))
			if got := rr.Header().Get("Cache-Control"); got != f.erwartet {
				t.Errorf("%s: Cache-Control %q, erwartet %q", f.pfad, got, f.erwartet)
			}
		})
	}
}
