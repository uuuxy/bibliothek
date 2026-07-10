package auth

import (
	"bibliothek/apierrors"
	"bibliothek/db"
	"bibliothek/pkg/httpresp"
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

// loginFailureEntry tracks failed login attempts per IP for brute-force protection.
type loginFailureEntry struct {
	count     int
	windowEnd time.Time
}

// loginFailureLimiter enforces max N failed logins per IP within a sliding window.
// This protects the IMAP server from credential-stuffing and brute-force attacks.
type loginFailureLimiter struct {
	mu      sync.Mutex
	entries map[string]*loginFailureEntry
	maxFail int           // max allowed failures before lockout
	window  time.Duration // rolling window duration
}

func newLoginFailureLimiter(maxFail int, window time.Duration) *loginFailureLimiter {
	return &loginFailureLimiter{
		entries: make(map[string]*loginFailureEntry),
		maxFail: maxFail,
		window:  window,
	}
}

// isBlocked returns true if the IP has exceeded the allowed failure count in the window.
func (l *loginFailureLimiter) isBlocked(ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	e, ok := l.entries[ip]
	if !ok {
		return false
	}
	if time.Now().After(e.windowEnd) {
		delete(l.entries, ip)
		return false
	}
	return e.count >= l.maxFail
}

// recordFailure increments the failure counter for an IP; resets the window on first failure.
func (l *loginFailureLimiter) recordFailure(ip string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	e, ok := l.entries[ip]
	if !ok || time.Now().After(e.windowEnd) {
		l.entries[ip] = &loginFailureEntry{count: 1, windowEnd: time.Now().Add(l.window)}
		return
	}
	e.count++
	// Evict stale entries to prevent unbounded growth (school has limited IPs)
	if len(l.entries) > 2000 {
		for k, v := range l.entries {
			if time.Now().After(v.windowEnd) {
				delete(l.entries, k)
			}
		}
	}
}

// globalLoginLimiter: 5 failed attempts per IP within 15 minutes.
var globalLoginLimiter = newLoginFailureLimiter(5, 15*time.Minute)

// realIP extracts the true client IP.
// X-Forwarded-For and X-Real-IP are only trusted when the direct connection
// comes from a loopback address (i.e. behind the Caddy reverse proxy).
func realIP(r *http.Request) string {
	remoteIP, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		// RemoteAddr ohne Port: unverändert als IP behandeln
		remoteIP = r.RemoteAddr
	}
	if remoteIP != "" {
		parsed := net.ParseIP(remoteIP)
		if parsed != nil && parsed.IsLoopback() {
			if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
				parts := strings.SplitN(xff, ",", 2)
				return strings.TrimSpace(parts[0])
			}
			if xri := r.Header.Get("X-Real-IP"); xri != "" {
				return xri
			}
		}
		return remoteIP
	}
	return r.RemoteAddr
}

// LoginRequest represents the payload for login.
type LoginRequest struct {
	BarcodeID string `json:"barcode_id,omitempty"`
	Email     string `json:"email,omitempty"`
	Password  string `json:"password,omitempty"`
	PIN       string `json:"pin,omitempty"`
}

// LoginResponse represents the response containing user information upon successful authentication.
type LoginResponse struct {
	UserID      string   `json:"user_id"`
	Rolle       Role     `json:"rolle"`
	Vorname     string   `json:"vorname"`
	Nachname    string   `json:"nachname"`
	Permissions []string `json:"permissions"`
}

// LoginHandler returns an http.HandlerFunc that performs secure authentication.
// Supports both email/password (with local DB or school IMAP verification) and barcode/PIN login.
func LoginHandler(dbPool db.PgxPoolIface, authenticator *Authenticator, cookieSecure bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		clientIP := realIP(r)

		var req LoginRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			apierrors.SendHTTPError(w, http.StatusBadRequest, err)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		var id, roleStr, vorname, nachname string
		var aktiv bool
		var authSuccess bool

		// 1. Check if it's an email-based login
		if req.Email == "" {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("email is required"))
			return
		}

		password := req.Password
		if password == "" {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("password is required"))
			return
		}

		// Brute-Force-Schutz: pro (E-Mail|IP) drosseln — NICHT rein pro IP. Sonst würde in einer
		// Schule, in der alle Geräte hinter EINER NAT-IP hängen, ein einziger Nutzer mit 5
		// Fehlversuchen die GESAMTE Schule für 15 Minuten am Login hindern. Der zusammengesetzte
		// Schlüssel sperrt nur das betroffene Konto auf dieser IP.
		bruteForceKey := strings.ToLower(strings.TrimSpace(req.Email)) + "|" + clientIP
		if globalLoginLimiter.isBlocked(bruteForceKey) {
			apierrors.SendHTTPError(w, http.StatusTooManyRequests,
				errors.New("zu viele fehlgeschlagene Login-Versuche – bitte 15 Minuten warten"))
			return
		}

		// ONLY perform IMAP verification (Roundcube SSO)
		if imapErr := AuthenticateIMAP(req.Email, password); imapErr == nil {
			// IMAP succeeded, check if the user is registered in our local DB
			query := `
				SELECT id, rolle, vorname, nachname, aktiv 
				FROM benutzer 
				WHERE LOWER(email) = LOWER($1) 
				LIMIT 1
			`
			err := dbPool.QueryRow(ctx, query, req.Email).Scan(&id, &roleStr, &vorname, &nachname, &aktiv)
			if err == nil {
				authSuccess = true
			}
		}

		if !authSuccess {
			globalLoginLimiter.recordFailure(bruteForceKey)
			apierrors.SendHTTPError(w, http.StatusUnauthorized, errors.New("invalid email or password"))
			return
		}

		if !aktiv {
			apierrors.SendHTTPError(w, http.StatusForbidden, errors.New("user account is deactivated"))
			return
		}

		role := Role(roleStr)
		token, err := authenticator.GenerateToken(id, req.BarcodeID, role)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		http.SetCookie(w, &http.Cookie{ //nolint:gosec // Pre-existing G124
			Name:     "session_token",
			Value:    token,
			Path:     "/",
			Expires:  time.Now().Add(authenticator.tokenDuration),
			HttpOnly: true,
			Secure:   os.Getenv("APP_ENV") != "local",
			SameSite: http.SameSiteStrictMode, // Strict: keine Cross-Site-Requests erlaubt
		})

		permissions, err := loadPermissionsForRole(ctx, dbPool, roleStr)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, errors.New("berechtigungen konnten nicht geladen werden"))
			return
		}

		w.Header().Set("Content-Type", "application/json")
		httpresp.Encode(w, LoginResponse{
			UserID:      id,
			Rolle:       role,
			Vorname:     vorname,
			Nachname:    nachname,
			Permissions: permissions,
		})
	}
}

// loadPermissionsForRole lädt die effektiven Rechte aus der konfigurierbaren
// role_permissions-Tabelle, damit das Frontend exakt das anzeigt, was der Admin im
// PermissionManager freigeschaltet hat. Admin hat implizit alle Rechte ("*"),
// analog zum Bypass in der RequirePermission-Middleware.
func loadPermissionsForRole(ctx context.Context, dbPool db.PgxPoolIface, roleStr string) ([]string, error) {
	if strings.EqualFold(roleStr, string(RoleAdmin)) {
		return []string{"*"}, nil
	}

	permissions := []string{}
	permRows, err := dbPool.Query(ctx, `
		SELECT permission
		FROM role_permissions
		WHERE UPPER(role) = UPPER($1) AND allowed = true
	`, roleStr)
	if err != nil {
		return nil, err
	}
	defer permRows.Close()
	for permRows.Next() {
		var p string
		if err := permRows.Scan(&p); err != nil {
			return nil, err
		}
		permissions = append(permissions, p)
	}
	if err := permRows.Err(); err != nil {
		return nil, err
	}
	return permissions, nil
}

// MeHandler liefert den Benutzer der aktuellen Session — gleicher Response-Body wie
// der Login. Der SPA-Boot nutzt ihn, um eine bestehende Session wiederherzustellen,
// statt bei jedem Reload den Login-Screen zu zeigen.
func MeHandler(dbPool db.PgxPoolIface, authenticator *Authenticator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("session_token")
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusUnauthorized, errors.New("keine aktive Sitzung"))
			return
		}

		claims, err := authenticator.VerifyToken(cookie.Value)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusUnauthorized, errors.New("sitzung abgelaufen oder ungültig"))
			return
		}

		ctx := r.Context()

		// Rolle und Stammdaten aus der DB — nicht aus den Claims: Rolle oder
		// Aktiv-Status können sich seit Token-Ausstellung geändert haben.
		var roleStr, vorname, nachname string
		var aktiv bool
		err = dbPool.QueryRow(ctx, `
			SELECT rolle, vorname, nachname, aktiv
			FROM benutzer
			WHERE id = $1
			LIMIT 1
		`, claims.UserID).Scan(&roleStr, &vorname, &nachname, &aktiv)
		if err != nil || !aktiv {
			apierrors.SendHTTPError(w, http.StatusUnauthorized, errors.New("keine aktive Sitzung"))
			return
		}

		permissions, err := loadPermissionsForRole(ctx, dbPool, roleStr)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, errors.New("berechtigungen konnten nicht geladen werden"))
			return
		}

		w.Header().Set("Content-Type", "application/json")
		httpresp.Encode(w, LoginResponse{
			UserID:      claims.UserID,
			Rolle:       Role(roleStr),
			Vorname:     vorname,
			Nachname:    nachname,
			Permissions: permissions,
		})
	}
}

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
				httpresp.Encode(w, map[string]string{"status": "ok", "refresh": "skipped"})
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
		http.SetCookie(w, &http.Cookie{ //nolint:gosec // Pre-existing G124
			Name:     "session_token",
			Value:    newToken,
			Path:     "/",
			Expires:  time.Now().Add(authenticator.tokenDuration),
			HttpOnly: true,
			Secure:   os.Getenv("APP_ENV") != "local",
			SameSite: http.SameSiteStrictMode,
		})

		w.Header().Set("Content-Type", "application/json")
		httpresp.Encode(w, map[string]string{"status": "ok", "refresh": "renewed"})
	}
}
