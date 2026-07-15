package api

import (
	"errors"
	"net/http"
	"sync"
	"time"

	"bibliothek/apierrors"
	"bibliothek/pkg/clientip"
)

// failedAttempt stores the count of failed login attempts and the time of the first failure in a window.
type failedAttempt struct {
	count     int
	firstFail time.Time
}

var (
	failedLogins      = make(map[string]*failedAttempt)
	failedLoginsMutex sync.Mutex
)

// evictExpiredLogins entfernt abgelaufene Login-Versuche. Aufrufer muss failedLoginsMutex halten.
// Wird beim Anlegen neuer Einträge ab einer Schwelle aufgerufen — das hält die Map ohne eine
// dauerhaft laufende Hintergrund-Goroutine beschränkt (verhindert Goroutine-Leaks in Tests).
func evictExpiredLogins(now time.Time) {
	for ip, attempt := range failedLogins {
		if now.Sub(attempt.firstFail) > 15*time.Minute {
			delete(failedLogins, ip)
		}
	}
}

// statusWriter intercepts the HTTP status code written by the wrapped handler.
type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

// reserviereLoginVersuch prüft/aktualisiert den Fehlversuchszähler einer IP (unter Lock).
// blocked=true bedeutet: die IP ist derzeit gesperrt (>=5 Versuche im 15-Minuten-Fenster).
func reserviereLoginVersuch(ip string, now time.Time) (attempt *failedAttempt, blocked bool) {
	failedLoginsMutex.Lock()
	defer failedLoginsMutex.Unlock()

	attempt, exists := failedLogins[ip]
	if exists {
		// If the penalty window of 15 minutes has expired, reset the attempt record.
		if now.Sub(attempt.firstFail) > 15*time.Minute {
			attempt.count = 0
			attempt.firstFail = now
		} else if attempt.count >= 5 {
			return attempt, true
		}
		return attempt, false
	}

	// Vor dem Einfügen ab einer Schwelle abgelaufene Einträge entfernen, um die Map
	// ohne Dauer-Goroutine beschränkt zu halten (analog zum IP-Rate-Limiter).
	if len(failedLogins) > 5000 {
		evictExpiredLogins(now)
	}
	attempt = &failedAttempt{
		count:     0,
		firstFail: now,
	}
	failedLogins[ip] = attempt
	return attempt, false
}

// verzeichneLoginErgebnis aktualisiert den Fehlversuchszähler anhand des Antwortstatus
// (401/403 erhöhen, 200 setzt bei Erfolg zurück) — unter Lock.
func verzeichneLoginErgebnis(attempt *failedAttempt, status int, now time.Time) {
	switch status {
	case http.StatusUnauthorized, http.StatusForbidden:
		failedLoginsMutex.Lock()
		attempt.count++
		failedLoginsMutex.Unlock()
	case http.StatusOK:
		// Optional: reset counter on successful login
		failedLoginsMutex.Lock()
		if attempt.count > 0 {
			attempt.count = 0
			attempt.firstFail = now
		}
		failedLoginsMutex.Unlock()
	}
}

// AuthRateLimitMiddleware limits the number of failed authentication attempts to 5 per IP within 15 minutes.
// Further requests within the window will be blocked with a 429 Too Many Requests response.
//
// The IP is resolved via clientip so that requests behind the Caddy reverse
// proxy are keyed on the real client — not on the single proxy address, which
// would let five failed logins lock out every user (global denial of service).
func AuthRateLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := clientip.FromRequest(r)

		now := time.Now()
		attempt, blocked := reserviereLoginVersuch(ip, now)
		if blocked {
			apierrors.SendHTTPError(w, http.StatusTooManyRequests, errors.New("zu viele fehlerhafte Login-Versuche. Bitte warten Sie 15 Minuten"))
			return
		}

		// Intercept the response status
		sw := &statusWriter{
			ResponseWriter: w,
			status:         http.StatusOK, // Default to OK in case the handler doesn't call WriteHeader
		}

		next.ServeHTTP(sw, r)

		// Increment/reset failure count based on the auth result
		verzeichneLoginErgebnis(attempt, sw.status, now)
	})
}
