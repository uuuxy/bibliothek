package auth

import (
	"bibliothek/db"
	"bufio"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"bibliothek/apierrors"
	"github.com/jackc/pgx/v5"

	"golang.org/x/crypto/bcrypt"
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

// AuthenticateIMAP verifies the email and password against the configured IMAP server.
func AuthenticateIMAP(serverHostPort, email, password string) (bool, error) {
	if serverHostPort == "" {
		return false, errors.New("IMAP server host:port not configured")
	}

	dialer := &net.Dialer{
		Timeout: 5 * time.Second,
	}
	conn, err := tls.DialWithDialer(dialer, "tcp", serverHostPort, &tls.Config{
		InsecureSkipVerify: false,
	})
	if err != nil {
		return false, fmt.Errorf("IMAP connection failed: %w", err)
	}
	defer conn.Close()

	_ = conn.SetDeadline(time.Now().Add(5 * time.Second))
	reader := bufio.NewReader(conn)

	_, err = reader.ReadString('\n')
	if err != nil {
		return false, fmt.Errorf("failed to read greeting: %w", err)
	}

	escapedEmail := strings.ReplaceAll(email, "\"", "\\\"")
	escapedPassword := strings.ReplaceAll(password, "\"", "\\\"")

	loginCmd := fmt.Sprintf("a001 LOGIN \"%s\" \"%s\"\r\n", escapedEmail, escapedPassword)
	_, err = conn.Write([]byte(loginCmd))
	if err != nil {
		return false, fmt.Errorf("failed to send login command: %w", err)
	}

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return false, fmt.Errorf("failed to read response: %w", err)
		}
		line = strings.ToLower(line)
		if strings.HasPrefix(line, "a001 ") {
			if strings.Contains(line, " ok ") || strings.Contains(line, "ok ") {
				return true, nil
			}
			return false, nil
		}
	}
}

// verifyPassword checks a bcrypt password hash.
func verifyPassword(hash, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
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

		var id, roleStr, vorname, nachname, email, passwortHash string
		var aktiv bool
		var authSuccess bool

		// 1. Check if it's an email-based login
		if req.Email != "" {
			password := req.Password
			if password == "" {
				password = req.PIN
			}
			if password == "" {
				apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("password is required"))
				return
			}

			// Look up user in DB by email
			query := `
				SELECT b.id, COALESCE(br.rolle, 'HELFER'), b.vorname, b.nachname, b.passwort_hash, b.aktiv 
				FROM benutzer b
				LEFT JOIN benutzer_rollen br ON b.id = br.benutzer_id
				WHERE LOWER(b.email) = LOWER($1) 
				LIMIT 1
			`
			err := dbPool.QueryRow(ctx, query, req.Email).Scan(&id, &roleStr, &vorname, &nachname, &passwortHash, &aktiv)
			if err == nil {
				// Try local DB verification first
				if verifyPassword(passwortHash, password) {
					authSuccess = true
				}
			}

			// Try IMAP verification if local DB failed or user not found locally
			if !authSuccess {
				imapServer := os.Getenv("IMAP_SERVER")
				if imapServer != "" {
					ok, imapErr := AuthenticateIMAP(imapServer, req.Email, password)
					if imapErr == nil && ok {
						// If user exists in DB, login succeeds
						if err == nil {
							authSuccess = true
						}
					}
				}
			}

			if !authSuccess {
				globalLoginLimiter.recordFailure(clientIP)
				apierrors.SendHTTPError(w, http.StatusUnauthorized, errors.New("invalid email or password"))
				return
			}
		} else {
			// Barcode login for Kiosk-Helfer
			barcodeID := req.BarcodeID
			pin := req.PIN
			if pin == "" {
				pin = req.Password
			}

			// Support barcode:pin combined scanners
			if pin == "" && strings.Contains(barcodeID, ":") {
				parts := strings.SplitN(barcodeID, ":", 2)
				barcodeID = parts[0]
				pin = parts[1]
			}

			if barcodeID == "" {
				apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("barcode_id or email is required"))
				return
			}

			if pin == "" {
				apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("PIN or password is required"))
				return
			}

			query := `
				SELECT b.id, COALESCE(br.rolle, 'HELFER'), b.vorname, b.nachname, b.email, b.passwort_hash, b.aktiv 
				FROM benutzer b
				LEFT JOIN benutzer_rollen br ON b.id = br.benutzer_id
				WHERE LOWER(b.barcode_id) = LOWER($1) OR (LOWER($1) = 'admin' AND LOWER(b.barcode_id) = 'admin-1')
				LIMIT 1
			`
			err := dbPool.QueryRow(ctx, query, barcodeID).Scan(&id, &roleStr, &vorname, &nachname, &email, &passwortHash, &aktiv)
			if err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					apierrors.SendHTTPError(w, http.StatusUnauthorized, err)
					return
				}
				apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
				return
			}

			if !verifyPassword(passwortHash, pin) {
				globalLoginLimiter.recordFailure(clientIP)
				apierrors.SendHTTPError(w, http.StatusUnauthorized, errors.New("invalid PIN"))
				return
			}
			authSuccess = true
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
		if role == RoleAdmin {
			permissions = []string{"*"}
		} else {
			rows, err := dbPool.Query(ctx, "SELECT permission FROM role_permissions WHERE UPPER(role) = UPPER($1) AND allowed = true", string(role))
			if err == nil {
				for rows.Next() {
					var p string
					if err := rows.Scan(&p); err == nil {
						permissions = append(permissions, p)
					}
				}
				rows.Close()
			}
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
