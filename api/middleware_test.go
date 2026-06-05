package api

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestCORSMiddleware(t *testing.T) {
	tests := []struct {
		name           string
		allowedOrigin  string
		originHeader   string
		expectedOrigin string
		expectedCreds  string
	}{
		{
			name:           "Empty allowedOrigin",
			allowedOrigin:  "",
			originHeader:   "http://example.com",
			expectedOrigin: "",
			expectedCreds:  "",
		},
		{
			name:           "Matching allowedOrigin",
			allowedOrigin:  "http://example.com",
			originHeader:   "http://example.com",
			expectedOrigin: "http://example.com",
			expectedCreds:  "true",
		},
		{
			name:           "Mismatching allowedOrigin",
			allowedOrigin:  "http://example.com",
			originHeader:   "http://other.com",
			expectedOrigin: "",
			expectedCreds:  "",
		},
		{
			name:           "Wildcard allowedOrigin",
			allowedOrigin:  "*",
			originHeader:   "http://example.com",
			expectedOrigin: "*",
			expectedCreds:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("ALLOWED_ORIGIN", tt.allowedOrigin)
			defer os.Unsetenv("ALLOWED_ORIGIN")

			handler := CORSMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set("Origin", tt.originHeader)

			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			if got := rr.Header().Get("Access-Control-Allow-Origin"); got != tt.expectedOrigin {
				t.Errorf("expected ACAO %q, got %q", tt.expectedOrigin, got)
			}
			if got := rr.Header().Get("Access-Control-Allow-Credentials"); got != tt.expectedCreds {
				t.Errorf("expected ACAC %q, got %q", tt.expectedCreds, got)
			}
		})
	}
}
