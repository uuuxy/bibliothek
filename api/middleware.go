package api

// middleware.go — HTTP middleware chain for the library server.
// All middleware is side-effect free and does not depend on application state
// beyond the Server struct. Middleware is registered in router.go.

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"runtime/debug"
	"strings"
	"time"

	"bibliothek/apierrors"
)

// TimeoutMiddleware wraps the request with a context timeout.
func TimeoutMiddleware(timeout time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.Contains(r.URL.Path, "/events") {
				next.ServeHTTP(w, r)
				return
			}
			ctx, cancel := context.WithTimeout(r.Context(), timeout)
			defer cancel()
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

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
			allowedOrigin := os.Getenv("ALLOWED_ORIGIN")
			var targetHost string

			if allowedOrigin != "" {
				// Use trusted configuration to prevent Open Redirect / Host Header Injection
				targetHost = strings.TrimPrefix(allowedOrigin, "https://")
				targetHost = strings.TrimPrefix(targetHost, "http://")
			} else {
				// Fallback to r.Host for multi-domain support if not strictly configured
				targetHost = r.Host
				if strings.ContainsAny(targetHost, "\r\n\t") {
					http.Error(w, "Bad Request", http.StatusBadRequest)
					return
				}
			}

			target := "https://" + targetHost + r.URL.RequestURI()
			http.Redirect(w, r, target, http.StatusMovedPermanently) // #nosec G710
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

// Hinweis: Die frühere RBACBlockMiddleware wurde entfernt. Sie erzwang eine hartkodierte
// Pfad-Allowlist für LEHRER/HELFER, die das konfigurierbare role_permissions-System überstimmte
// (ein LEHRER konnte seine im PermissionManager gewährten Rechte nicht nutzen). Autorisierung
// erfolgt jetzt einheitlich pro Route über RequirePermission bzw. RequireRoles.

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

// statusRecorder is a custom ResponseWriter that tracks the HTTP status code
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

// Unwrap returns the underlying ResponseWriter so that http.NewResponseController
// can traverse the middleware chain (e.g. to call SetWriteDeadline for SSE).
func (r *statusRecorder) Unwrap() http.ResponseWriter {
	return r.ResponseWriter
}

// Flush implements http.Flusher to support Server-Sent Events (SSE)
func (r *statusRecorder) Flush() {
	if f, ok := r.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// LoggingMiddleware records the HTTP status and prints the exact stack trace for 500 errors.
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		recorder := &statusRecorder{
			ResponseWriter: w,
			status:         http.StatusOK,
		}

		next.ServeHTTP(recorder, r)

		if recorder.status >= 500 {
			log.Printf("HTTP 500 ERROR on %s %s - STACKTRACE:\n%s", r.Method, r.URL.Path, string(debug.Stack()))
		}
	})
}
