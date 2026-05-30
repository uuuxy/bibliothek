package auth

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"bibliothek/apierrors"
)

type contextKey string

// ClaimsContextKey is the key used to store and retrieve authentication claims from the request context.
const ClaimsContextKey contextKey = "auth_claims"

// RequireRoles returns a middleware that validates the session cookie and verifies
// if the authenticated user has one of the allowed roles.
func (a *Authenticator) RequireRoles(allowedRoles ...Role) func(http.Handler) http.Handler {
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

			claims, err := a.VerifyToken(cookie.Value)
			if err != nil {
				apierrors.SendHTTPError(w, http.StatusUnauthorized, err)
				return
			}

			// Role-Based Access Control (RBAC) validation
			roleAllowed := false
			for _, role := range allowedRoles {
				if strings.EqualFold(string(claims.Rolle), string(role)) {
					roleAllowed = true
					break
				}
			}

			if !roleAllowed {
				apierrors.SendHTTPError(w, http.StatusForbidden, errors.New("insufficient permissions"))
				return
			}

			// Inject user claims into request context for downstream handlers
			ctx := context.WithValue(r.Context(), ClaimsContextKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetClaims retrieves auth claims from the request context.
// Returns the claims and a boolean indicating whether the claims were present.
func GetClaims(ctx context.Context) (*Claims, bool) {
	claims, ok := ctx.Value(ClaimsContextKey).(*Claims)
	return claims, ok
}
