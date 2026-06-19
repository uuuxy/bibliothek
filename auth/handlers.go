package auth

import (
	"bibliothek/apierrors"
	"bibliothek/db"
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
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

// realIP extracts the true client IP, honoring X-Forwarded-For from trusted reverse proxies.
func realIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.SplitN(xff, ",", 2)
		return strings.TrimSpace(parts[0])
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
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
		// Brute-force protection: block IPs that exceeded 5 failed logins in 15 minutes
		clientIP := realIP(r)
		if globalLoginLimiter.isBlocked(clientIP) {
			apierrors.SendHTTPError(w, http.StatusTooManyRequests,
				errors.New("zu viele fehlgeschlagene Login-Versuche – bitte 15 Minuten warten"))
			return
		}

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
			globalLoginLimiter.recordFailure(clientIP)
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

		// #nosec G124 - Secure flag is dynamically configured via cookieSecure
		http.SetCookie(w, &http.Cookie{
			Name:     "session_token",
			Value:    token,
			Path:     "/",
			Expires:  time.Now().Add(authenticator.tokenDuration),
			HttpOnly: true,
			Secure:   cookieSecure,
			SameSite: http.SameSiteStrictMode, // Strict: keine Cross-Site-Requests erlaubt
		})

		var permissions []string
		switch roleStr {
		case "admin":
			permissions = []string{"manage_users", "manage_settings", "print_classes", "manage_inventory"}
		case "mitarbeiter":
			permissions = []string{"print_classes", "manage_inventory"}
		case "lehrer":
			permissions = []string{"view_media"}
		case "helfer":
			permissions = []string{}
		default:
			permissions = []string{}
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(LoginResponse{
			UserID:      id,
			Rolle:       role,
			Vorname:     vorname,
			Nachname:    nachname,
			Permissions: permissions,
		})
	}
}
