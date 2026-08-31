package auth

import (
	"bibliothek/apierrors"
	"bibliothek/db"
	"bibliothek/pkg/clientip"
	"bibliothek/pkg/httpresp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"log/slog"
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

// realIP extracts the true client IP via the shared, trusted-proxy-aware
// resolver (see pkg/clientip), so the per-(email+IP) brute-force key is keyed on
// the real client rather than the Caddy proxy address.
func realIP(r *http.Request) string {
	return clientip.FromRequest(r)
}

// LoginRequest represents the payload for login.
//
// NUR E-Mail und Passwort. Hier standen früher zusätzlich barcode_id und pin:
//
//   - pin wurde nie gelesen. validateLoginCredentials verlangt E-Mail und Passwort und
//     steigt sonst aus — einen Barcode/PIN-Anmeldeweg gibt es nicht, der Kiosk scannt
//     Ausweise, meldet damit aber niemanden an.
//   - barcode_id wurde gelesen und wanderte ungeprüft in die Claims des Session-Tokens.
//     Niemand hat den Wert je ausgewertet, ein Loch war es also nicht. Aber ein
//     signiertes Token, das eine vom Client behauptete Kennung trägt, ist eine Falle für
//     den Nächsten, der ihr glaubt: Signiert heisst geprüft, sonst wäre die Signatur
//     sinnlos. Der Barcode kommt jetzt aus der benutzer-Tabelle.
type LoginRequest struct {
	Email    string `json:"email,omitempty"`
	Password string `json:"password,omitempty"`
}

// LoginResponse represents the response containing user information upon successful authentication.
type LoginResponse struct {
	UserID string `json:"user_id"`
	// Email: die eigene Anmelde-Identität des Benutzers (er hat sie gerade selbst
	// eingetippt bzw. besitzt die Sitzung). Der Sperrbildschirm braucht sie für die
	// Wiederanmeldung nach dem Session-Restore, wo der Client sie sonst nicht kennt.
	Email       string   `json:"email"`
	Rolle       Role     `json:"rolle"`
	Vorname     string   `json:"vorname"`
	Nachname    string   `json:"nachname"`
	Permissions []string `json:"permissions"`
}

// loginHandlerFrist umfasst IMAP (imapFrist) UND die DB-Schritte danach. Sie MUSS über
// imapFrist liegen — sonst stirbt der Kontext zwischen „Passwort richtig" und dem
// Benutzer-Lookup, und ein korrektes, langsames Login wird als Fehlversuch gezählt
// (so bis 22.08.2026: 10 s gegen 15 s). Gate: TestLoginHandlerFrist_LaesstIMAPLuft.
const loginHandlerFrist = imapFrist + 10*time.Second

// LoginHandler returns an http.HandlerFunc that performs secure authentication.
// Anmeldung ausschliesslich per E-Mail/Passwort gegen den Schul-Mailserver (IMAP).
func LoginHandler(dbPool db.PgxPoolIface, authenticator *Authenticator, cookieSecure bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		clientIP := realIP(r)

		var req LoginRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			apierrors.SendHTTPError(w, http.StatusBadRequest, err)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), loginHandlerFrist)
		defer cancel()

		// 1. Check if it's an email-based login
		password, ok := validateLoginCredentials(w, req)
		if !ok {
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

		user, verifyErr := verifyIMAPCredentials(ctx, dbPool, req.Email, password)
		if verifyErr != nil {
			// Mailserver-Ausfall ≠ falsches Passwort (Ausfallmatrix 20.08.2026): Vorher
			// bekam der Nutzer bei Server-Down „invalid email or password" UND einen
			// gezählten Fehlversuch — wer sein richtiges Passwort dann erneut probierte,
			// sperrte sich selbst für 15 Minuten. Jetzt 503 ohne recordFailure.
			if errors.Is(verifyErr, ErrMailserverNichtErreichbar) {
				apierrors.SendHTTPError(w, http.StatusServiceUnavailable, ErrMailserverNichtErreichbar)
				return
			}
			// Dasselbe für die Datenbank (31.08.2026): Ein DB-Aussetzer ist kein
			// Passwortfehler — 503 ohne recordFailure, sonst Selbstsperre mit
			// korrektem Passwort. Der echte Grund steht im Log.
			if errors.Is(verifyErr, ErrAnmeldedienstGestoert) {
				slog.Error("Login: Anmeldedienst gestört", "fehler", verifyErr)
				apierrors.SendHTTPError(w, http.StatusServiceUnavailable, ErrAnmeldedienstGestoert)
				return
			}
			globalLoginLimiter.recordFailure(bruteForceKey)
			apierrors.SendHTTPError(w, http.StatusUnauthorized, errors.New("invalid email or password"))
			return
		}

		// Beides ist HTTP 403 und beides bedeutet „Zugangsdaten stimmen, Zugang trotzdem
		// nicht" — die Unterscheidung ist für den Menschen davor. „Konto deaktiviert" ließe
		// eine Lehrkraft, die sich gerade zum ersten Mal gemeldet hat, ratlos zurück und
		// verleitet zum Wiederholen; sie soll wissen, dass sie nur warten muss.
		//
		// KEIN recordFailure hier: Das Passwort war richtig. Sonst sperrte sich jemand mit
		// fünf Versuchen selbst aus, während seine Freischaltung noch aussteht.
		if user.neuAngelegt {
			//nolint:staticcheck // ST1005: nutzer-sichtbare Meldung, steht so im Anmeldeformular
			apierrors.SendHTTPError(w, http.StatusForbidden,
				errors.New("Zugang beantragt — die Bibliothek muss ihn noch freischalten"))
			return
		}
		if !user.aktiv {
			apierrors.SendHTTPError(w, http.StatusForbidden, errors.New("user account is deactivated"))
			return
		}

		role := Role(user.roleStr)
		token, err := authenticator.GenerateToken(user.id, user.barcodeID, role)
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
			SameSite: http.SameSiteStrictMode, // Strict: keine Cross-Site-Requests erlaubt
		})

		permissions, err := loadPermissionsForRole(ctx, dbPool, user.roleStr)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, errors.New("berechtigungen konnten nicht geladen werden"))
			return
		}

		w.Header().Set(headerContentType, contentTypeJSON)
		httpresp.Encode(w, LoginResponse{
			UserID:      user.id,
			Email:       user.email,
			Rolle:       role,
			Vorname:     user.vorname,
			Nachname:    user.nachname,
			Permissions: permissions,
		})
	}
}

// loginUser bündelt die aus der benutzer-Tabelle geladenen Login-Felder.
type loginUser struct {
	id    string
	email string
	// barcodeID kommt aus der benutzer-Tabelle, NICHT aus der Anfrage — siehe LoginRequest.
	barcodeID string
	roleStr   string
	vorname   string
	nachname  string
	aktiv     bool
	// neuAngelegt: Diese Zeile ist gerade erst durch die Selbstanmeldung entstanden
	// (siehe selbstanmeldung.go) und ist zwangsläufig inaktiv. Nur für die Antwort an
	// den Anmeldenden gedacht — er soll „Zugang beantragt" lesen statt „Konto
	// deaktiviert" und nicht in Endlosschleife weiterprobieren.
	neuAngelegt bool
}

// validateLoginCredentials erzwingt das Vorhandensein von E-Mail und Passwort.
// ok=false: die Fehlerantwort wurde bereits geschrieben.
func validateLoginCredentials(w http.ResponseWriter, req LoginRequest) (password string, ok bool) {
	if req.Email == "" {
		apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("email is required"))
		return "", false
	}
	if req.Password == "" {
		apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("password is required"))
		return "", false
	}
	return req.Password, true
}

// verifyIMAPCredentials prüft die Zugangsdaten per IMAP (Roundcube-SSO) und lädt bei
// Erfolg den lokal registrierten Benutzer. Ein Fehler bedeutet: kein gültiger Login.
// ErrMailserverNichtErreichbar wird UNVERÄNDERT durchgereicht, damit der Handler den
// Transport-Ausfall vom falschen Passwort unterscheiden kann (503 statt 401+Sperre).
func verifyIMAPCredentials(ctx context.Context, dbPool db.PgxPoolIface, email, password string) (loginUser, error) {
	// ONLY perform IMAP verification (Roundcube SSO)
	if imapErr := AuthenticateIMAP(ctx, email, password); imapErr != nil {
		return loginUser{}, imapErr
	}

	// IMAP succeeded, check if the user is registered in our local DB
	var u loginUser
	query := `
		SELECT id, coalesce(barcode_id, ''), rolle, vorname, nachname, aktiv, email
		FROM benutzer
		WHERE LOWER(email) = LOWER($1)
		LIMIT 1
	`
	if err := dbPool.QueryRow(ctx, query, email).Scan(&u.id, &u.barcodeID, &u.roleStr, &u.vorname, &u.nachname, &u.aktiv, &u.email); err != nil {
		// NUR „Zeile nicht vorhanden" ist der Selbstanmelde-Fall. Jeder andere Fehler
		// (Verbindungsabriss, Pool erschöpft, ctx-Frist) ist ein Ausfall des
		// Anmeldedienstes — bis zum 31.08.2026 lief er in den 401-Pfad und wurde als
		// Fehlversuch gezählt: Bei einem DB-Aussetzer sperrte sich eine Lehrkraft mit
		// korrektem Passwort selbst (dieselbe Klasse wie der Mailserver-Fall, 20.08.).
		if !errors.Is(err, pgx.ErrNoRows) {
			return loginUser{}, fmt.Errorf("%w: benutzer lesen: %v", ErrAnmeldedienstGestoert, err)
		}
		// Kein lokaler Eintrag. Wenn die Selbstanmeldung für diese Domain freigegeben ist,
		// entsteht hier eine INAKTIVE Zugangsanfrage — kein Zugang, nur ein Eintrag, den
		// die Bibliothek freischalten kann. Ist sie nicht freigegeben, bleibt es beim
		// bisherigen Verhalten (kein Login).
		neu, anlegeErr := legeZugangsanfrageAn(ctx, dbPool, email)
		if anlegeErr != nil {
			// „Nicht erlaubt" ist der Normalfall (Domain nicht freigegeben) und braucht
			// keine Logzeile. Alles andere ist ein DB-Fehler, der vorher als nacktes 401
			// „invalid email or password" endete — in der CI (29.08.2026) war so nicht
			// zu sehen, WARUM die Zugangsanfrage nicht entstand.
			if !errors.Is(anlegeErr, errSelbstanmeldungNichtErlaubt) {
				slog.Error("Selbstanmeldung: Zugangsanfrage konnte nicht angelegt werden", "fehler", anlegeErr)
			}
			return loginUser{}, anlegeErr
		}
		return neu, nil
	}
	return u, nil
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
		var roleStr, vorname, nachname, email string
		var aktiv bool
		err = dbPool.QueryRow(ctx, `
			SELECT rolle, vorname, nachname, aktiv, email
			FROM benutzer
			WHERE id = $1
			LIMIT 1
		`, claims.UserID).Scan(&roleStr, &vorname, &nachname, &aktiv, &email)
		// Fehler-Kollaps (Sweep 29.08.2026): Ein DB-Fehler ist keine „abgelaufene Sitzung" —
		// vorher warf ein Verbindungsabbruch jeden Nutzer mit 401 aus der Anwendung, und
		// der Client löschte daraufhin die Sitzung.
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		if err != nil || !aktiv {
			apierrors.SendHTTPError(w, http.StatusUnauthorized, errors.New("keine aktive Sitzung"))
			return
		}

		permissions, err := loadPermissionsForRole(ctx, dbPool, roleStr)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, errors.New("berechtigungen konnten nicht geladen werden"))
			return
		}

		w.Header().Set(headerContentType, contentTypeJSON)
		httpresp.Encode(w, LoginResponse{
			UserID:      claims.UserID,
			Email:       email,
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
				w.Header().Set(headerContentType, contentTypeJSON)
				httpresp.Encode(w, map[string]string{"status": "ok", "refresh": "skipped"})
				return
			}
		}

		// Neues Token mit frischer Laufzeit. claims.Rolle ist hier bereits die
		// AKTUELLE Rolle aus der Datenbank — VerifyToken überschreibt die im alten
		// Token signierte. Vorher schrieb der Refresh die alte Rolle fort und machte
		// aus einer 12-Stunden-Staleness eine unbegrenzte.
		newToken, err := authenticator.GenerateToken(claims.UserID, claims.BarcodeID, claims.Rolle)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		// Set the new session cookie
		http.SetCookie(w, &http.Cookie{
			Name:     "session_token",
			Value:    newToken,
			Path:     "/",
			Expires:  time.Now().Add(authenticator.tokenDuration),
			HttpOnly: true,
			Secure:   cookieSecure,
			SameSite: http.SameSiteStrictMode,
		})

		w.Header().Set(headerContentType, contentTypeJSON)
		httpresp.Encode(w, map[string]string{"status": "ok", "refresh": "renewed"})
	}
}
