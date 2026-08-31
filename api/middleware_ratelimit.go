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

// maxLoginFehlversucheProIP ist die VOLUMETRISCHE Obergrenze dieser Middleware —
// bewusst hoch, nicht als Konto-Schutz gedacht.
//
// Vorher standen hier 5. Das war dieselbe Zahl wie im Handler (auth/handlers.go),
// nur mit dem entscheidenden Unterschied, dass der Handler pro (E-Mail|IP) zählt
// und diese Schicht rein pro IP. In einer Schule, deren Geräte über EINE NAT-Adresse
// herauskommen, genügten damit fünf Vertipper irgendeines Nutzers, um den Login für
// ALLE anderen 15 Minuten lang zu sperren — die Härtung des Handlers auf (E-Mail|IP)
// wurde von der äußeren Schicht wieder eingesammelt.
//
// Der Kontoschutz gegen gezieltes Raten liegt jetzt allein beim (E-Mail|IP)-Limiter
// im Handler. Was hier bleibt, ist die Bremse gegen Credential-Stuffing, das sich
// über viele Konten verteilt und den IMAP-Server der Schule flutet: 50 Fehlversuche
// in 15 Minuten aus EINER Quelle erreicht kein normaler Standort, ein Skript sofort.
const maxLoginFehlversucheProIP = 50

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
// blocked=true bedeutet: die IP ist derzeit gesperrt (siehe maxLoginFehlversucheProIP,
// gezählt im 15-Minuten-Fenster).
func reserviereLoginVersuch(ip string, now time.Time) (attempt *failedAttempt, blocked bool) {
	failedLoginsMutex.Lock()
	defer failedLoginsMutex.Unlock()

	attempt, exists := failedLogins[ip]
	if exists {
		// If the penalty window of 15 minutes has expired, reset the attempt record.
		if now.Sub(attempt.firstFail) > 15*time.Minute {
			attempt.count = 0
			attempt.firstFail = now
		} else if attempt.count >= maxLoginFehlversucheProIP {
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
// (NUR 401 erhöht, 200 setzt bei Erfolg zurück) — unter Lock.
//
// 403 zählt seit dem 31.08.2026 NICHT mehr: Bei „Zugang beantragt" (Selbstanmeldung)
// und „Konto deaktiviert" war das Passwort richtig — der Handler zählt deshalb bewusst
// nicht (auth/handlers.go), und diese Schicht darf die Entscheidung nicht einsammeln.
// Zum Schuljahresbeginn hätten sonst zwanzig Selbstanmelde-403 hinter der EINEN
// NAT-Adresse der Schule den Login für alle gesperrt — auch für die Bibliothekskraft,
// die die Freischaltungen vornimmt. Für Credential-Stuffing (der Zweck dieser Bremse)
// ist 401 das Signal, nicht 403.
func verzeichneLoginErgebnis(attempt *failedAttempt, status int, now time.Time) {
	switch status {
	case http.StatusUnauthorized:
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

// AuthRateLimitMiddleware begrenzt fehlgeschlagene Anmeldungen pro IP auf
// maxLoginFehlversucheProIP innerhalb von 15 Minuten; weitere Versuche im Fenster
// beantwortet sie mit 429.
//
// Sie ist die VOLUMETRISCHE Schranke. Der Schutz eines einzelnen Kontos gegen
// gezieltes Raten sitzt im Login-Handler und zählt pro (E-Mail|IP) — siehe
// maxLoginFehlversucheProIP für den Grund, warum beide Schichten unterschiedliche
// Schwellen haben müssen.
//
// The IP is resolved via clientip so that requests behind the Caddy reverse
// proxy are keyed on the real client — not on the single proxy address, which
// would let a handful of failed logins lock out every user (global denial of service).
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
