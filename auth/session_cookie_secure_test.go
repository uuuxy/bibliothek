package auth

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/pashagolub/pgxmock/v4"
)

// Diese Datei schließt eine Lücke, die am 04.09.2026 beim Nachprüfen der abgewiesenen
// CodeQL-Meldungen auffiel (go/cookie-secure-not-set, Alerts #14/#15/#16).
//
// Die Abweisungen begründeten sich mit „getestet in cookie_secure_test.go". Das stimmte
// nur zur Hälfte: Jene Datei prüft ermittleCookieSecure() — also die AUSWERTUNG von
// APP_ENV und COOKIE_SECURE. Dass der ermittelte Wert am Sitzungscookie auch ankommt,
// prüfte sie nicht. Für das CSRF-Cookie gab es diesen Beleg (api/csrf_cookie_test.go),
// für das Sitzungscookie nicht.
//
// Nachgewiesen: Wird `Secure: cookieSecure` in LoginHandler und RefreshTokenHandler
// durch ein hartes `false` ersetzt, läuft die gesamte Go-Testsuite weiterhin grün.
// Genau das verhindern die folgenden Tests — sie prüfen die Set-Cookie-Kopfzeile, die
// der Browser tatsächlich bekommt, und zwar in BEIDE Richtungen: Ein hart verdrahtetes
// true fiele hier ebenso auf wie ein hart verdrahtetes false.

// sitzungsCookie holt das session_token aus der Antwort oder bricht ab.
func sitzungsCookie(t *testing.T, rec *httptest.ResponseRecorder) *http.Cookie {
	t.Helper()
	for _, c := range rec.Result().Cookies() {
		if c.Name == "session_token" {
			return c
		}
	}
	t.Fatalf("kein session_token-Cookie in der Antwort: %+v", rec.Result().Cookies())
	return nil
}

// pruefeSchutzattribute hält fest, was unabhängig von der Konfiguration gilt.
func pruefeSchutzattribute(t *testing.T, c *http.Cookie) {
	t.Helper()
	if !c.HttpOnly {
		t.Error("Sitzungscookie muss HttpOnly sein — sonst liest es jedes eingeschleuste Skript")
	}
	if c.SameSite != http.SameSiteStrictMode {
		t.Errorf("SameSite = %v, erwartet Strict", c.SameSite)
	}
	if c.Path != "/" {
		t.Errorf("Path = %q, erwartet /", c.Path)
	}
}

func TestLoginCookieFolgtDerKonfiguration(t *testing.T) {
	for _, secure := range []bool{true, false} {
		t.Run(map[bool]string{true: "secure=true", false: "secure=false"}[secure], func(t *testing.T) {
			aktiviereMockIMAP(t)
			a, mock := newTestAuthenticator(t, 12*time.Hour)
			mock.ExpectQuery(benutzerSelect).
				WithArgs("pflasch@schule.de").
				WillReturnRows(pgxmock.NewRows([]string{"id", "barcode_id", "rolle", "vorname", "nachname", "aktiv", "email"}).
					AddRow("u-admin", "BC-TEST", "admin", "Peter", "Flasch", true, "peter@example.org"))

			req := httptest.NewRequest(http.MethodPost, "/login",
				strings.NewReader(`{"email":"pflasch@schule.de","password":"egal"}`))
			rec := httptest.NewRecorder()
			LoginHandler(mock, a, secure)(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("erwartet 200, bekam %d: %s", rec.Code, rec.Body.String())
			}
			c := sitzungsCookie(t, rec)
			if c.Secure != secure {
				t.Errorf("Secure = %v, erwartet %v — das Flag folgt der Konfiguration nicht", c.Secure, secure)
			}
			pruefeSchutzattribute(t, c)
		})
	}
}

func TestRefreshCookieFolgtDerKonfiguration(t *testing.T) {
	for _, secure := range []bool{true, false} {
		t.Run(map[bool]string{true: "secure=true", false: "secure=false"}[secure], func(t *testing.T) {
			// Restlaufzeit 1 h gegen 12-h-Fenster: Der Sliding Refresh greift und
			// stellt ein neues Cookie aus.
			issuer, _ := newTestAuthenticator(t, 1*time.Hour)
			token, err := issuer.GenerateToken("user-1", "B-1", RoleMitarbeiter)
			if err != nil {
				t.Fatalf("GenerateToken: %v", err)
			}
			a, mock := newTestAuthenticator(t, 12*time.Hour)
			expectNotBlacklisted(mock)
			expectKontoStatus(mock, true, RoleMitarbeiter)

			req := httptest.NewRequest(http.MethodPost, "/api/auth/refresh", nil)
			req.AddCookie(&http.Cookie{Name: "session_token", Value: token})
			rec := httptest.NewRecorder()
			RefreshTokenHandler(a, secure)(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("erwartet 200, bekam %d: %s", rec.Code, rec.Body.String())
			}
			c := sitzungsCookie(t, rec)
			if c.Secure != secure {
				t.Errorf("Secure = %v, erwartet %v — das Flag folgt der Konfiguration nicht", c.Secure, secure)
			}
			pruefeSchutzattribute(t, c)
		})
	}
}
