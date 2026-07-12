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
	"os"
	"strings"

	"bibliothek/apierrors"
)

const csrfCookieName = "csrf_token"

// generateGlobalCSRFToken creates a 32-byte cryptographically random token.
func generateGlobalCSRFToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// newCSRFCookie baut das (nicht-HttpOnly) Double-Submit-Cookie. Das Secure-Flag wird
// dynamisch aus APP_ENV abgeleitet, damit lokale HTTP-Entwicklung funktioniert.
func newCSRFCookie(token string) *http.Cookie {
	// #nosec G124 - Secure flag is dynamically configured
	return &http.Cookie{
		Name:     csrfCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: false, // Must be readable by frontend JS
		Secure:   os.Getenv("APP_ENV") != "local",
		SameSite: http.SameSiteStrictMode,
		MaxAge:   86400, // 24 hours
	}
}

// CSRFTokenHandler is an idempotent bootstrap endpoint (GET /api/csrf-token) that
// guarantees a csrf_token cookie is set and returns the token in the body. It lets
// non-browser API clients obtain a token deterministically — without first triggering
// a 403 on a mutating request that has no prior cookie. Browsers get the cookie via the
// CSRFMiddleware on any GET, but a direct POST without a preceding GET would otherwise fail.
func (s *Server) CSRFTokenHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := ""
		if cookie, err := r.Cookie(csrfCookieName); err == nil {
			token = strings.TrimSpace(cookie.Value)
		}
		if token == "" {
			generated, err := generateGlobalCSRFToken()
			if err != nil {
				apierrors.SendHTTPError(w, http.StatusInternalServerError, errors.New("CSRF-Token konnte nicht erzeugt werden"))
				return
			}
			token = generated
			http.SetCookie(w, newCSRFCookie(token))
		}
		RespondJSON(w, http.StatusOK, map[string]string{"csrf_token": token})
	}
}

// classifyCSRFPath ordnet einen Request-Pfad für die globale CSRF-Prüfung ein.
// isInventurPath markiert Pfade, die das Inventur-Modul mit eigenem CSRF absichert.
func classifyCSRFPath(path string) (isAPIPath, isInventurPath bool) {
	// Skip non-API paths (static frontend assets, swagger, etc.)
	isAPIPath = strings.HasPrefix(path, "/api/") ||
		path == "/login/barcode"

	// Diese Admin-Mutationen werden NICHT vom Inventur-Modul behandelt, sondern direkt
	// im globalen Router registriert (router.go). Sie müssen daher der globalen
	// CSRF-Prüfung unterliegen und dürfen nicht über den /api/admin-Präfix ausgenommen
	// werden — sonst gäbe es für sie gar keinen CSRF-Schutz. Das Frontend sendet für
	// beide bereits den X-CSRF-Token-Header.
	globalAdminMutation := path == "/api/admin/sync-covers" || path == "/api/admin/import-bestand"

	// Skip paths handled by the inventur module's own CSRF system
	isInventurPath = !globalAdminMutation && (strings.HasPrefix(path, "/api/books") ||
		strings.HasPrefix(path, "/api/class-books") ||
		strings.HasPrefix(path, "/api/lookup/") ||
		strings.HasPrefix(path, "/api/subjects") ||
		strings.HasPrefix(path, "/api/admin") ||
		strings.HasPrefix(path, "/api/auth/status") ||
		strings.HasPrefix(path, "/uploads/"))

	return isAPIPath, isInventurPath
}

// refreshCSRFCookie setzt das csrf_token-Cookie, falls noch keines (bzw. ein leeres)
// existiert, damit das Frontend es auslesen kann. Der Bootstrap-Endpunkt verwaltet
// sein Cookie selbst und wird hier übersprungen, um doppelte Set-Cookie-Header zu
// vermeiden.
func refreshCSRFCookie(w http.ResponseWriter, r *http.Request, isAPIPath, isInventurPath bool, path string) {
	if !isAPIPath || isInventurPath || path == "/api/csrf-token" {
		return
	}

	existingToken := ""
	if cookie, err := r.Cookie(csrfCookieName); err == nil {
		existingToken = cookie.Value
	}
	// Generate a new token only if one doesn't exist yet
	if existingToken != "" {
		return
	}
	token, err := generateGlobalCSRFToken()
	if err == nil {
		http.SetCookie(w, newCSRFCookie(token))
	}
}

// validateCSRFDoubleSubmit vergleicht das csrf_token-Cookie mit dem X-CSRF-Token-
// Header in konstanter Laufzeit und liefert bei Fehlschlag einen sprechenden Fehler.
func validateCSRFDoubleSubmit(r *http.Request) error {
	csrfCookie, err := r.Cookie(csrfCookieName)
	if err != nil {
		return errors.New("CSRF-Validierung fehlgeschlagen: Cookie fehlt")
	}
	cookieVal := strings.TrimSpace(csrfCookie.Value)
	headerVal := strings.TrimSpace(r.Header.Get("X-CSRF-Token"))

	if cookieVal == "" || headerVal == "" {
		return errors.New("CSRF-Validierung fehlgeschlagen: Token fehlt")
	}
	if subtle.ConstantTimeCompare([]byte(cookieVal), []byte(headerVal)) != 1 {
		return errors.New("CSRF-Validierung fehlgeschlagen: Token stimmt nicht überein")
	}
	return nil
}

// CSRFMiddleware returns an HTTP middleware that enforces the Double-Submit
// Cookie CSRF pattern on all mutating API requests.
//
// Exempt paths: /login/barcode, /health, paths starting with /api/books
// (handled by inventur's own CSRF), and non-API paths (static assets).
func (s *Server) CSRFMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		isAPIPath, isInventurPath := classifyCSRFPath(path)

		// Always set/refresh the CSRF cookie so the frontend can read it.
		refreshCSRFCookie(w, r, isAPIPath, isInventurPath, path)

		// Only validate on mutating methods for API paths (not inventur)
		isMutation := r.Method == http.MethodPost ||
			r.Method == http.MethodPut ||
			r.Method == http.MethodPatch ||
			r.Method == http.MethodDelete

		// Exempt: login endpoint (no cookie yet), logout, refresh, and inventur paths
		isExempt := path == "/login/barcode" ||
			path == "/api/auth/logout" ||
			path == "/api/auth/refresh" ||
			isInventurPath ||
			!isAPIPath

		if isMutation && !isExempt {
			if err := validateCSRFDoubleSubmit(r); err != nil {
				apierrors.SendHTTPError(w, http.StatusForbidden, err)
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}
