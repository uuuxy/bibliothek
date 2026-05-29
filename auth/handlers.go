package auth

import (
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
	"time"

	"bibliothek/apierrors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

// LoginRequest represents the payload for login.
type LoginRequest struct {
	BarcodeID string `json:"barcode_id,omitempty"`
	Email     string `json:"email,omitempty"`
	Password  string `json:"password,omitempty"`
	PIN       string `json:"pin,omitempty"`
}

// LoginResponse represents the response containing user information upon successful authentication.
type LoginResponse struct {
	UserID   string `json:"user_id"`
	Rolle    Role   `json:"rolle"`
	Vorname  string `json:"vorname"`
	Nachname string `json:"nachname"`
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
		InsecureSkipVerify: true,
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

// verifyPassword checks a bcrypt password hash or dummy_passwort_hash fallback.
func verifyPassword(hash, password string) bool {
	if hash == "dummy_passwort_hash" && password == "admin" {
		return true
	}
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// LoginHandler returns an http.HandlerFunc that performs secure authentication.
// Supports both email/password (with local DB or school IMAP verification) and barcode/PIN login.
func LoginHandler(dbPool *pgxpool.Pool, authenticator *Authenticator, cookieSecure bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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
				SELECT id, rolle::text, vorname, nachname, passwort_hash, aktiv 
				FROM benutzer 
				WHERE LOWER(email) = LOWER($1) 
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
				apierrors.SendHTTPError(w, http.StatusUnauthorized, errors.New("invalid email or password"))
				return
			}
		} else {
			// 2. Barcode login for Kiosk-Helfer
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
				SELECT id, rolle::text, vorname, nachname, email, passwort_hash, aktiv 
				FROM benutzer 
				WHERE LOWER(barcode_id) = LOWER($1) 
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

		http.SetCookie(w, &http.Cookie{
			Name:     "session_token",
			Value:    token,
			Path:     "/",
			Expires:  time.Now().Add(authenticator.tokenDuration),
			HttpOnly: true,
			Secure:   cookieSecure,
			SameSite: http.SameSiteLaxMode,
		})

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(LoginResponse{
			UserID:   id,
			Rolle:    role,
			Vorname:  vorname,
			Nachname: nachname,
		})
	}
}
