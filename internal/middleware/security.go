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
		// img-src: self and data: (removed wildcard https:)
		// script-src/style-src: self (removed unsafe-inline)
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self'; font-src 'self'; img-src 'self' data:; connect-src 'self'; frame-ancestors 'none'; form-action 'self';")
		
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		
		// Feature-Policy is deprecated, using Permissions-Policy
		w.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")

		next.ServeHTTP(w, r)
	})
}
