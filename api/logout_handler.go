package api

import (
	"errors"
	"net/http"
	"os"
	"time"

	"bibliothek/apierrors"
)

// logoutHandler blacklists the current JWT and clears the session cookie.
// This was previously referenced in CSRF/RBAC exemptions but never actually registered.
func (s *Server) logoutHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("session_token")
		if err != nil {
			if errors.Is(err, http.ErrNoCookie) {
				// Already logged out — idempotent
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":"ok"}`))
				return
			}
			apierrors.SendHTTPError(w, http.StatusBadRequest, err)
			return
		}

		// Parse the token to get expiration time
		claims, err := s.Auth.VerifyToken(cookie.Value)
		if err == nil && claims.ExpiresAt != nil {
			// Blacklist the token so it can't be reused until it naturally expires
			s.Auth.Blacklist.Add(cookie.Value, claims.ExpiresAt.Time)
		}

		// #nosec G124 - Secure flag is dynamically configured
		http.SetCookie(w, &http.Cookie{
			Name:     "session_token",
			Value:    "",
			Path:     "/",
			Expires:  time.Unix(0, 0),
			MaxAge:   -1,
			HttpOnly: true,
			Secure:   os.Getenv("APP_ENV") != "local",
			SameSite: http.SameSiteStrictMode,
		})

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}
}
