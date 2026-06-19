package auth

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"bibliothek/apierrors"
)

// RefreshTokenHandler returns a handler that silently refreshes an active, valid session.
// If the existing JWT is still valid and has not been revoked, a new JWT is issued with
// a fresh expiry window (sliding window). The old token is NOT blacklisted to avoid race
// conditions with concurrent requests that are still using the old token.
//
// This prevents forced re-login during active library use (e.g. a Mitarbeiter working
// a 6-hour shift with a 12h token window).
func RefreshTokenHandler(authenticator *Authenticator, cookieSecure bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("session_token")
		if err != nil {
			if errors.Is(err, http.ErrNoCookie) {
				apierrors.SendHTTPError(w, http.StatusUnauthorized, errors.New("keine aktive Sitzung"))
				return
			}
			apierrors.SendHTTPError(w, http.StatusBadRequest, err)
			return
		}

		// Verify the existing token is still valid and not revoked
		claims, err := authenticator.VerifyToken(cookie.Value)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusUnauthorized, errors.New("sitzung abgelaufen oder ungültig"))
			return
		}

		// Only refresh if the token has less than 50% of its lifetime remaining.
		// This prevents unnecessary token churn from frequent polling/requests.
		if claims.ExpiresAt != nil {
			remaining := time.Until(claims.ExpiresAt.Time)
			if remaining > authenticator.tokenDuration/2 {
				// Token is still fresh enough, return current session info
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok", "refresh": "skipped"})
				return
			}
		}

		// Generate a fresh token with the same claims but a new expiry
		newToken, err := authenticator.GenerateToken(claims.UserID, claims.BarcodeID, claims.Rolle)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		// Set the new session cookie
		// #nosec G124 - Secure flag is dynamically configured via cookieSecure
		http.SetCookie(w, &http.Cookie{
			Name:     "session_token",
			Value:    newToken,
			Path:     "/",
			Expires:  time.Now().Add(authenticator.tokenDuration),
			HttpOnly: true,
			Secure:   cookieSecure,
			SameSite: http.SameSiteStrictMode,
		})

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok", "refresh": "renewed"})
	}
}
