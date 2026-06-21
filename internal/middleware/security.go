package middleware

import (
	"net/http"
)

// SecurityHeadersMiddleware adds strict security headers to all responses.
// This includes Content-Security-Policy (CSP) restricted to 'self', 
// X-Content-Type-Options, X-Frame-Options, X-XSS-Protection, 
// Referrer-Policy, and Permissions-Policy.
func SecurityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		
		// Set CSP strictly to 'self' where possible
		// font-src: now only self (Google Fonts removed)
		// img-src: self, data:, blob:, and https: (for external covers and blob URLs)
		// script-src: self
		// style-src: self and unsafe-inline (needed for Svelte inline style bindings)
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; font-src 'self'; img-src 'self' data: blob: https:; connect-src 'self'; frame-ancestors 'none'; form-action 'self';")
		
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		
		// Feature-Policy is deprecated, using Permissions-Policy
		// Allow camera for barcode scanning
		w.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=(self)")

		next.ServeHTTP(w, r)
	})
}
