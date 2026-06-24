package api

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strings"

	"bibliothek/apierrors"
	"bibliothek/auth"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
)

var (
	permCache   = make(map[string]cacheEntry)
	permCacheMu sync.RWMutex
)

type cacheEntry struct {
	Allowed   bool
	ExpiresAt time.Time
}

// InvalidatePermissionCache clears all cached permission entries.
// Call this whenever roles or permissions are changed (e.g. UpdateUser, UpdatePermissions)
// to prevent stale cache entries from granting/denying access incorrectly.
func InvalidatePermissionCache() {
	permCacheMu.Lock()
	permCache = make(map[string]cacheEntry)
	permCacheMu.Unlock()
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
					log.Printf("Permission denied for role '%s' permission '%s' (FROM CACHE).", claims.Rolle, permission)
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

			// Transiente DB-Fehler (Timeout, Pool erschöpft, Verbindungsabriss) dürfen NICHT
			// als Entscheidung gecacht werden: sonst würde ein kurzer DB-Aussetzer für berechtigte
			// Nutzer 60 s lang ein 403 zementieren. Ein echter Fehler ist zudem ein Server- (500),
			// kein Berechtigungsproblem (403). pgx.ErrNoRows hingegen bedeutet "Recht nicht gewährt"
			// und ist eine stabile, cachebare Negativ-Entscheidung.
			if err != nil && !errors.Is(err, pgx.ErrNoRows) {
				log.Printf("Permission check DB error for role '%s' permission '%s': %v", claims.Rolle, permission, err)
				apierrors.SendHTTPError(w, http.StatusInternalServerError, errors.New("berechtigung konnte nicht geprüft werden"))
				return
			}

			notFound := errors.Is(err, pgx.ErrNoRows)
			finalAllowed := err == nil && allowed

			// Update Cache (nur stabile Entscheidungen: gewährt, oder explizit nicht gewährt)
			permCacheMu.Lock()
			// Abgelaufene Einträge beim Miss-Write opportunistisch entfernen, damit der Cache
			// nicht unbegrenzt wächst. Der Keyspace (Rolle × Permission) ist klein, daher ist
			// ein vollständiger Sweep hier günstig und kommt ohne Hintergrund-Goroutine aus.
			now := time.Now()
			for k, v := range permCache {
				if now.After(v.ExpiresAt) {
					delete(permCache, k)
				}
			}
			permCache[cacheKey] = cacheEntry{
				Allowed:   finalAllowed,
				ExpiresAt: now.Add(60 * time.Second),
			}
			permCacheMu.Unlock()

			if notFound || !finalAllowed {
				log.Printf("Permission denied for role '%s' permission '%s'. allowed: %v", claims.Rolle, permission, finalAllowed)
				apierrors.SendHTTPError(w, http.StatusForbidden, errors.New("keine Berechtigung für diese Aktion"))
				return
			}

			// Inject user claims into request context
			ctx := context.WithValue(r.Context(), auth.ClaimsContextKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
