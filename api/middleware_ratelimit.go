package api

import (
	"errors"
	"net"
	"net/http"
	"sync"
	"time"

	"bibliothek/apierrors"
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

// init starts a background goroutine to clean up expired failed login records from memory.
func init() {
	go func() {
		for {
			time.Sleep(15 * time.Minute)
			now := time.Now()
			failedLoginsMutex.Lock()
			for ip, attempt := range failedLogins {
				if now.Sub(attempt.firstFail) > 15*time.Minute {
					delete(failedLogins, ip)
				}
			}
			failedLoginsMutex.Unlock()
		}
	}()
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

// AuthRateLimitMiddleware limits the number of failed authentication attempts to 5 per IP within 15 minutes.
// Further requests within the window will be blocked with a 429 Too Many Requests response.
func AuthRateLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			ip = r.RemoteAddr
		}

		// Check if IP is currently blocked
		failedLoginsMutex.Lock()
		attempt, exists := failedLogins[ip]
		now := time.Now()

		if exists {
			// If the penalty window of 15 minutes has expired, reset the attempt record.
			if now.Sub(attempt.firstFail) > 15*time.Minute {
				attempt.count = 0
				attempt.firstFail = now
			} else if attempt.count >= 5 {
				failedLoginsMutex.Unlock()
				apierrors.SendHTTPError(w, http.StatusTooManyRequests, errors.New("zu viele fehlerhafte Login-Versuche. Bitte warten Sie 15 Minuten"))
				return
			}
		} else {
			attempt = &failedAttempt{
				count:     0,
				firstFail: now,
			}
			failedLogins[ip] = attempt
		}
		failedLoginsMutex.Unlock()

		// Intercept the response status
		sw := &statusWriter{
			ResponseWriter: w,
			status:         http.StatusOK, // Default to OK in case the handler doesn't call WriteHeader
		}

		next.ServeHTTP(sw, r)

		// Increment failure count if authentication failed
		switch sw.status {
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
	})
}
