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
		"X-Content-Type-Options":    "nosniff",
		"X-Frame-Options":           "DENY",
		"X-XSS-Protection":          "1; mode=block",
		"Content-Security-Policy":   "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; font-src 'self'; img-src 'self' data: blob: https:; connect-src 'self'; frame-ancestors 'none'; form-action 'self';",
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
