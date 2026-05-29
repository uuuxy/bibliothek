package inventur

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"golang.org/x/time/rate"
)

func TestRateLimiterMiddleware(t *testing.T) {
	// Create a rate limiter with 1 request per second and burst of 2.
	limiter := NewIPBasedRateLimiter(rate.Limit(1), 2)

	// Dummy handler that returns 200 OK
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Wrap the dummy handler with the middleware
	handlerToTest := limiter.Middleware(nextHandler)

	// Helper function to make a request and return the status code
	makeRequest := func(ip string) int {
		req := httptest.NewRequest("GET", "/", nil)
		req.RemoteAddr = ip + ":1234" // Simulate RemoteAddr
		rr := httptest.NewRecorder()
		handlerToTest.ServeHTTP(rr, req)
		return rr.Code
	}

	// Test case 1: Allow first 2 requests for IP1 (matches burst size)
	ip1 := "192.168.1.1"
	if status := makeRequest(ip1); status != http.StatusOK {
		t.Errorf("Expected status %v for 1st request, got %v", http.StatusOK, status)
	}
	if status := makeRequest(ip1); status != http.StatusOK {
		t.Errorf("Expected status %v for 2nd request, got %v", http.StatusOK, status)
	}

	// Test case 2: Block 3rd request for IP1
	if status := makeRequest(ip1); status != http.StatusTooManyRequests {
		t.Errorf("Expected status %v for 3rd request, got %v", http.StatusTooManyRequests, status)
	}

	// Test case 3: Allow requests for a different IP (IP2) even if IP1 is blocked
	ip2 := "10.0.0.1"
	if status := makeRequest(ip2); status != http.StatusOK {
		t.Errorf("Expected status %v for 1st request from IP2, got %v", http.StatusOK, status)
	}
}

func TestExtrahiereClientIP(t *testing.T) {
	tests := []struct {
		name       string
		remoteAddr string
		xRealIP    string
		expected   string
	}{
		{
			name:       "Trusted proxy uses X-Real-IP",
			remoteAddr: "127.0.0.1:1234",
			xRealIP:    "10.0.0.1",
			expected:   "10.0.0.1",
		},
		{
			name:       "Untrusted peer ignores X-Real-IP",
			remoteAddr: "8.8.8.8:1234",
			xRealIP:    "10.0.0.1",
			expected:   "8.8.8.8",
		},
		{
			name:       "Trusted proxy with invalid X-Real-IP falls back to peer",
			remoteAddr: "127.0.0.1:1234",
			xRealIP:    "invalid-ip",
			expected:   "127.0.0.1",
		},
		{
			name:       "Fallback to RemoteAddr with port",
			remoteAddr: "192.168.1.1:1234",
			xRealIP:    "",
			expected:   "192.168.1.1",
		},
		{
			name:       "Fallback to RemoteAddr without port",
			remoteAddr: "192.168.1.1",
			xRealIP:    "",
			expected:   "192.168.1.1",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			req.RemoteAddr = tc.remoteAddr
			if tc.xRealIP != "" {
				req.Header.Set("X-Real-IP", tc.xRealIP)
			}

			actual := extrahiereClientIP(req)
			if actual != tc.expected {
				t.Errorf("Expected %v, got %v", tc.expected, actual)
			}
		})
	}
}
