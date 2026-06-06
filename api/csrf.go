package api

// csrf.go — Global CSRF protection middleware using the Double-Submit Cookie pattern.
//
// How it works:
//   1. Every response sets a non-HttpOnly cookie "csrf_token" containing a
//      cryptographically random token. The frontend JS reads this cookie.
//   2. On mutating requests (POST/PUT/PATCH/DELETE), the middleware compares
//      the cookie value against the X-CSRF-Token header sent by the frontend.
//   3. If they don't match or are missing, the request is rejected with 403.
//
// This complements SameSite=Strict cookies as a defense-in-depth measure.
// The inventur sub-module has its own CSRF system ("inventur_csrf") which
// remains untouched — this middleware skips paths handled by inventur.

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"net/http"
	"strings"

	"bibliothek/apierrors"
)

// generateGlobalCSRFToken creates a 32-byte cryptographically random token.
func generateGlobalCSRFToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// CSRFMiddleware returns an HTTP middleware that enforces the Double-Submit
// Cookie CSRF pattern on all mutating API requests.
//
// Exempt paths: /login/barcode, /health, paths starting with /api/books
// (handled by inventur's own CSRF), and non-API paths (static assets).
func (s *Server) CSRFMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		// Skip non-API paths (static frontend assets, swagger, etc.)
		isAPIPath := strings.HasPrefix(path, "/api/") ||
			path == "/login/barcode"

		// Skip paths handled by the inventur module's own CSRF system
		isInventurPath := strings.HasPrefix(path, "/api/books") ||
			strings.HasPrefix(path, "/api/class-books") ||
			strings.HasPrefix(path, "/api/lookup/") ||
			strings.HasPrefix(path, "/api/subjects") ||
			strings.HasPrefix(path, "/api/admin") ||
			strings.HasPrefix(path, "/api/auth/status") ||
			strings.HasPrefix(path, "/uploads/")

		// Always set/refresh the CSRF cookie so the frontend can read it
		if isAPIPath && !isInventurPath {
			existingToken := ""
			if cookie, err := r.Cookie("csrf_token"); err == nil {
				existingToken = cookie.Value
			}
			// Generate a new token only if one doesn't exist yet
			if existingToken == "" {
				token, err := generateGlobalCSRFToken()
				if err == nil {
					// #nosec G124 - HttpOnly must be false for CSRF double submit, Secure is dynamic
					http.SetCookie(w, &http.Cookie{
						Name:     "csrf_token",
						Value:    token,
						Path:     "/",
						HttpOnly: false, // Must be readable by frontend JS
						Secure:   s.CookieSecure,
						SameSite: http.SameSiteLaxMode,
						MaxAge:   86400, // 24 hours
					})
				}
			}
		}

		// Only validate on mutating methods for API paths (not inventur)
		isMutation := r.Method == http.MethodPost ||
			r.Method == http.MethodPut ||
			r.Method == http.MethodPatch ||
			r.Method == http.MethodDelete

		// Exempt: login endpoint (no cookie yet), logout, and inventur paths
		isExempt := path == "/login/barcode" ||
			path == "/api/auth/logout" ||
			isInventurPath ||
			!isAPIPath

		if isMutation && !isExempt {
			csrfCookie, err := r.Cookie("csrf_token")
			if err != nil {
				apierrors.SendHTTPError(w, http.StatusForbidden,
					errors.New("CSRF-Validierung fehlgeschlagen: Cookie fehlt"))
				return
			}
			cookieVal := strings.TrimSpace(csrfCookie.Value)
			headerVal := strings.TrimSpace(r.Header.Get("X-CSRF-Token"))

			if cookieVal == "" || headerVal == "" {
				apierrors.SendHTTPError(w, http.StatusForbidden,
					errors.New("CSRF-Validierung fehlgeschlagen: Token fehlt"))
				return
			}

			if subtle.ConstantTimeCompare([]byte(cookieVal), []byte(headerVal)) != 1 {
				apierrors.SendHTTPError(w, http.StatusForbidden,
					errors.New("CSRF-Validierung fehlgeschlagen: Token stimmt nicht überein"))
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}
