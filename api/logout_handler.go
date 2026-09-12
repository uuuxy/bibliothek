package api

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"bibliothek/apierrors"
	"bibliothek/auth"
	"bibliothek/pkg/httpresp"
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
				httpresp.Write(w, []byte(`{"status":"ok"}`))
				return
			}
			apierrors.SendHTTPError(w, http.StatusBadRequest, err)
			return
		}

		// Parse the token to get expiration time
		claims, err := s.Auth.VerifyToken(cookie.Value)
		// widerrufFehler bleibt nil, solange es nichts zu widerrufen GIBT: Ein
		// ungültiges, abgelaufenes oder längst widerrufenes Token ist kein Aussetzer,
		// die Abmeldung ist dann vollständig. Gestört ist es nur, wenn die Datenbank
		// nicht antwortet — dann bleibt das Token gültig.
		var widerrufFehler error
		switch {
		case errors.Is(err, auth.ErrPruefungGestoert):
			// Der Widerruf wurde nie versucht: Schon die Prüfung fand keine Datenbank,
			// und ohne sie ist die Ablaufzeit nicht zu bekommen.
			widerrufFehler = err
		case err == nil && claims.ExpiresAt != nil:
			// Blacklist the token so it can't be reused until it naturally expires
			widerrufFehler = s.Auth.Blacklist.Add(cookie.Value, claims.ExpiresAt.Time)
		}

		// #nosec G124 - Secure flag is dynamically configured
		http.SetCookie(w, &http.Cookie{
			Name:     "session_token",
			Value:    "",
			Path:     "/",
			Expires:  time.Unix(0, 0),
			MaxAge:   -1,
			HttpOnly: true,
			Secure:   s.CookieSecure,
			SameSite: http.SameSiteStrictMode,
		})

		// Das Löschcookie steht oben und geht in BEIDEN Fällen hinaus — es ist die
		// Hälfte, die stattgefunden hat. Die Antwort darunter sagt, ob die andere
		// Hälfte auch stattgefunden hat: Ein „ok" nach gescheitertem Widerruf käme aus
		// der Eingabe und nicht aus der Wirkung.
		if widerrufFehler != nil {
			apierrors.SendHTTPErrorMitMeldung(w, http.StatusServiceUnavailable,
				auth.ErrWiderrufGestoert.Error(), fmt.Errorf("token-widerruf: %w", widerrufFehler))
			return
		}

		w.Header().Set("Content-Type", "application/json")
		httpresp.Write(w, []byte(`{"status":"ok"}`))
	}
}
