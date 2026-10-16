package api

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"bibliothek/apierrors"
	"bibliothek/auth"
	"bibliothek/pkg/httpresp"
)

// abmelden widerruft das Token der Sitzung und löscht das Sitzungscookie.
func (s *Server) abmelden() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("session_token")
		if err != nil {
			if errors.Is(err, http.ErrNoCookie) {
				// Schon abgemeldet: Ein zweiter Aufruf antwortet wie der erste.
				w.Header().Set("Content-Type", "application/json")
				httpresp.Schreibe(w, []byte(`{"status":"ok"}`))
				return
			}
			apierrors.AntworteFehlerMitMeldung(w, http.StatusBadRequest, apierrors.MeldungAnfrageUnlesbar, err)
			return
		}

		// Die Ablaufzeit steht im Token. Abmelden geht auch aus der Sperre nach Inaktivität heraus
		// (Sperrbildschirm, „Abmelden und als andere Person anmelden").
		claims, err := s.Auth.PruefeTokenTrotzSperre(r.Context(), cookie.Value)
		// widerrufFehler bleibt nil, solange es nichts zu widerrufen gibt: Ein
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
			// Auf die Sperrliste: Bis zu seinem Ablauf gilt das Token nicht mehr.
			widerrufFehler = s.Auth.Sperrliste.Widerrufe(r.Context(), cookie.Value, claims.ExpiresAt.Time)
		}
		if err == nil {
			// Mit der Zeile geht der Prüfwert des Passworts. Scheitert das Löschen, räumt
			// der Server sie nach Ablauf ab; das Token ist schon widerrufen.
			if endeErr := s.Auth.Sitzungen.Beende(r.Context(), claims.SitzungID); endeErr != nil {
				slog.Warn("Abmelden: Zeile der Anmeldung ließ sich nicht löschen", "fehler", endeErr)
			}
		}

		// #nosec G124 - Secure folgt nurHTTPS
		http.SetCookie(w, &http.Cookie{
			Name:     "session_token",
			Value:    "",
			Path:     "/",
			Expires:  time.Unix(0, 0),
			MaxAge:   -1,
			HttpOnly: true,
			Secure:   s.nurHTTPS,
			SameSite: http.SameSiteStrictMode,
		})

		// Das Löschcookie steht oben und geht in beiden Fällen hinaus: Es ist die
		// Hälfte, die stattgefunden hat. Die Antwort darunter sagt, ob die andere
		// Hälfte auch stattgefunden hat: Ein „ok" nach gescheitertem Widerruf käme aus
		// der Eingabe und nicht aus der Wirkung.
		if widerrufFehler != nil {
			apierrors.AntworteFehlerMitMeldung(w, http.StatusServiceUnavailable,
				auth.ErrWiderrufGestoert.Error(), fmt.Errorf("token-widerruf: %w", widerrufFehler))
			return
		}

		w.Header().Set("Content-Type", "application/json")
		httpresp.Schreibe(w, []byte(`{"status":"ok"}`))
	}
}
