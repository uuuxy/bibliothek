package auth

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"bibliothek/apierrors"
)

type contextKey string

// ClaimsContextKey ist der Schlüssel, der verwendet wird, um Authentifizierungs-Claims im Request-Kontext zu speichern und abzurufen.
const ClaimsContextKey contextKey = "auth_claims"

// RequireRoles gibt eine Middleware zurück, die das Session-Cookie validiert und prüft,
// ob der authentifizierte Benutzer eine der erlaubten Rollen besitzt.
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

			// Role-Based Access Control (RBAC) Validierung
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

			// Benutzer-Claims für nachgelagerte Handler in den Request-Kontext injizieren
			ctx := context.WithValue(r.Context(), ClaimsContextKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetClaims ruft Authentifizierungs-Claims aus dem Request-Kontext ab.
// Gibt die Claims und einen booleschen Wert zurück, der angibt, ob die Claims vorhanden waren.
func GetClaims(ctx context.Context) (*Claims, bool) {
	claims, ok := ctx.Value(ClaimsContextKey).(*Claims)
	return claims, ok
}
