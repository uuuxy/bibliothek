package api

// middleware.go — HTTP middleware chain for the library server.
// All middleware is side-effect free and does not depend on application state
// beyond the Server struct. Middleware is registered in router.go.

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"bibliothek/apierrors"
	"regexp"
)

// PanicRecoveryMiddleware catches panics during HTTP request handling, logs them, and returns a 500 error.
func PanicRecoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("PANIC RECOVERED in request %s %s: %v", r.Method, r.URL.Path, err)
				apierrors.SendHTTPError(w, http.StatusInternalServerError, fmt.Errorf("interner server fehler: %v", err))
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// HTTPSRedirectMiddleware automatically redirects unencrypted HTTP requests to HTTPS.
func (s *Server) HTTPSRedirectMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Bypass HTTPS redirection in local/development mode when CookieSecure is disabled
		if !s.CookieSecure {
			next.ServeHTTP(w, r)
			return
		}
		isHTTPS := r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https"
		if !isHTTPS {
			target := "https://" + r.Host + r.URL.RequestURI()
			http.Redirect(w, r, target, http.StatusMovedPermanently)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// MaxBodySizeMiddleware limits the request body size to prevent DoS.
func MaxBodySizeMiddleware(limit int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Body = http.MaxBytesReader(w, r.Body, limit)
			next.ServeHTTP(w, r)
		})
	}
}

// RBACBlockMiddleware checks roles and enforces path access rules for LEHRER and HELFER roles.
func (s *Server) RBACBlockMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if path == "/health" || path == "/login/barcode" || path == "/api/auth/status" {
			next.ServeHTTP(w, r)
			return
		}

		cookie, err := r.Cookie("session_token")
		if err == nil && cookie.Value != "" {
			claims, err := s.Auth.VerifyToken(cookie.Value)
			if err == nil {
				role := strings.ToUpper(string(claims.Rolle))
				switch role {
				case "LEHRER":
					isAllowed := (r.Method == http.MethodGet && (path == "/api/search" || strings.HasPrefix(path, "/api/buecher/titel/") && strings.Contains(path, "/exemplare"))) ||
						(r.Method == http.MethodPost && path == "/api/auth/logout")

					if !isAllowed {
						apierrors.SendHTTPError(w, http.StatusForbidden, errors.New("zugriff verweigert für Lehrer"))
						return
					}
				case "HELFER":
					isAllowed := (r.Method == http.MethodPost && (path == "/api/action" || path == "/api/auth/logout")) ||
						(r.Method == http.MethodGet && path == "/events")

					if !isAllowed {
						apierrors.SendHTTPError(w, http.StatusForbidden, errors.New("zugriff verweigert für Helfer"))
						return
					}
				}
			}
		}

		next.ServeHTTP(w, r)
	})
}

// SecurityHeadersMiddleware sets HSTS, X-Frame-Options, X-Content-Type-Options and
// Referrer-Policy on every response. HSTS uses a 1-year max-age with includeSubDomains
// and preload to harden the school domain against protocol-downgrade attacks.
func SecurityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// HSTS: 1 year, include subdomains, eligible for preload list
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")
		// Prevent clickjacking
		w.Header().Set("X-Frame-Options", "DENY")
		// Prevent MIME-type sniffing
		w.Header().Set("X-Content-Type-Options", "nosniff")
		// Restrict referrer information
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		// Basic CSP: allow same-origin resources only (adjust if CDN is added)
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; font-src 'self' https://fonts.gstatic.com; img-src 'self' data: https:; connect-src 'self'")
		next.ServeHTTP(w, r)
	})
}

// CORSMiddleware restricts cross-origin requests to the configured school domain.
// Set ALLOWED_ORIGIN env var to the school's frontend URL (e.g. https://bibliothek.schule.de).
// Falls back to same-origin only if not configured.
func CORSMiddleware(next http.Handler) http.Handler {
	allowedOrigin := os.Getenv("ALLOWED_ORIGIN")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin == "" {
			// Same-origin request (no Origin header) – always allowed
			next.ServeHTTP(w, r)
			return
		}
		// Only allow explicitly configured origin; reject everything else
		if allowedOrigin != "" && origin == allowedOrigin {
			w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Vary", "Origin")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// requireAuth is an internal helper that extracts valid JWT claims from the request cookie.
// Returns claims and true on success; writes 401 and returns false on failure.
/*
func (s *Server) requireAuth(w http.ResponseWriter, r *http.Request) (*auth.Claims, bool) {
	claims, ok := auth.GetClaims(r.Context())
	if !ok {
		apierrors.SendHTTPError(w, http.StatusUnauthorized, errors.New("authentication required"))
	}
	return claims, ok
}
*/

var uuidRegex = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// ValidateUUIDParamsMiddleware intercepts requests and validates {id} path parameters
// against a standard UUID format before they hit the database.
func ValidateUUIDParamsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if id != "" && !uuidRegex.MatchString(id) {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("ungültiges UUID Format im Pfadparameter"))
			return
		}
		next.ServeHTTP(w, r)
	})
}

