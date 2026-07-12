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

// authClaimsAusRequest liest und verifiziert das Session-Cookie. ok=false bedeutet:
// die Fehlerantwort (401 ohne Cookie / bei ungültigem Token, 400 bei Cookie-Fehler)
// wurde bereits geschrieben.
func (a *Authenticator) authClaimsAusRequest(w http.ResponseWriter, r *http.Request) (*Claims, bool) {
	cookie, err := r.Cookie("session_token")
	if err != nil {
		if errors.Is(err, http.ErrNoCookie) {
			apierrors.SendHTTPError(w, http.StatusUnauthorized, err)
			return nil, false
		}
		apierrors.SendHTTPError(w, http.StatusBadRequest, err)
		return nil, false
	}

	claims, err := a.VerifyToken(cookie.Value)
	if err != nil {
		apierrors.SendHTTPError(w, http.StatusUnauthorized, err)
		return nil, false
	}
	return claims, true
}

// enthaeltRolle prüft, ob rolle in der Liste erlaubter Rollen enthalten ist (case-insensitive).
func enthaeltRolle(allowedRoles []Role, rolle Role) bool {
	for _, role := range allowedRoles {
		if strings.EqualFold(string(rolle), string(role)) {
			return true
		}
	}
	return false
}

// RequireRoles gibt eine Middleware zurück, die das Session-Cookie validiert und prüft,
// ob der authentifizierte Benutzer eine der erlaubten Rollen besitzt.
func (a *Authenticator) RequireRoles(allowedRoles ...Role) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := a.authClaimsAusRequest(w, r)
			if !ok {
				return
			}

			// Role-Based Access Control (RBAC) Validierung
			if !enthaeltRolle(allowedRoles, claims.Rolle) {
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
