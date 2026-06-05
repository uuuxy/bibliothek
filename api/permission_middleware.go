package api

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"bibliothek/apierrors"
	"bibliothek/auth"
	"sync"
	"time"
)

var (
	permCache   = make(map[string]cacheEntry)
	permCacheMu sync.RWMutex
)

type cacheEntry struct {
	Allowed   bool
	ExpiresAt time.Time
}

// RequirePermission returns a middleware that validates if the authenticated user
// has the required permission dynamically defined in the database.
func (s *Server) RequirePermission(permission string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie("session_token")
			if err != nil {
				if errors.Is(err, http.ErrNoCookie) {
					apierrors.SendHTTPError(w, http.StatusUnauthorized, err)
					return
				}
				apierrors.SendHTTPError(w, http.StatusBadRequest, err)
				return
			}

			claims, err := s.Auth.VerifyToken(cookie.Value)
			if err != nil {
				apierrors.SendHTTPError(w, http.StatusUnauthorized, err)
				return
			}

			// Admin role always has all permissions allowed
			if strings.EqualFold(string(claims.Rolle), string(auth.RoleAdmin)) {
				ctx := context.WithValue(r.Context(), auth.ClaimsContextKey, claims)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			// Try Cache
			cacheKey := string(claims.Rolle) + ":" + permission
			permCacheMu.RLock()
			entry, found := permCache[cacheKey]
			permCacheMu.RUnlock()

			if found && time.Now().Before(entry.ExpiresAt) {
				if !entry.Allowed {
					apierrors.SendHTTPError(w, http.StatusForbidden, errors.New("keine Berechtigung für diese Aktion"))
					return
				}
				ctx := context.WithValue(r.Context(), auth.ClaimsContextKey, claims)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			// Check role permissions in DB (Cache miss)
			var allowed bool
			query := `
				SELECT allowed 
				FROM role_permissions 
				WHERE UPPER(role) = UPPER($1) AND permission = $2
			`
			err = s.DB.Pool.QueryRow(r.Context(), query, string(claims.Rolle), permission).Scan(&allowed)
			
			// Update Cache
			permCacheMu.Lock()
			permCache[cacheKey] = cacheEntry{
				Allowed:   err == nil && allowed,
				ExpiresAt: time.Now().Add(60 * time.Second),
			}
			permCacheMu.Unlock()

			if err != nil || !allowed {
				apierrors.SendHTTPError(w, http.StatusForbidden, errors.New("keine Berechtigung für diese Aktion"))
				return
			}

			// Inject user claims into request context
			ctx := context.WithValue(r.Context(), auth.ClaimsContextKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
