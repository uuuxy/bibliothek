package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"bibliothek/auth"
	"bibliothek/db"

	"github.com/pashagolub/pgxmock/v4"
)

// Dritte Sitzungscookie-Stelle neben Login und Refresh (siehe
// auth/session_cookie_secure_test.go): das Loeschcookie beim Abmelden.
//
// Es traegt keinen Wert mehr, die Schutzattribute zaehlen trotzdem. Ein Loeschcookie
// ohne Secure ueberschreibt ueber HTTP ein Secure-Cookie NICHT zuverlaessig — die
// Abmeldung waere dann eine, die der Browser still ignoriert. Genau deshalb muss auch
// hier der konfigurierte Wert ankommen statt eines hart verdrahteten.
func TestLogoutCookieFolgtDerKonfiguration(t *testing.T) {
	for _, secure := range []bool{true, false} {
		t.Run(map[bool]string{true: "secure=true", false: "secure=false"}[secure], func(t *testing.T) {
			mock, err := pgxmock.NewPool()
			if err != nil {
				t.Fatalf("pgxmock init: %v", err)
			}
			t.Cleanup(mock.Close)
			a, err := auth.NewAuthenticator(testJWTSecret, mock, time.Hour)
			if err != nil {
				t.Fatalf("authenticator init: %v", err)
			}
			t.Cleanup(a.Blacklist.Stop)
			s := &Server{DB: &db.Database{Pool: mock}, Auth: a, CookieSecure: secure}

			// Unbrauchbares Token: VerifyToken scheitert an der Signatur, noch bevor
			// die Datenbank gefragt wird — der Handler loescht das Cookie trotzdem.
			req := httptest.NewRequest(http.MethodPost, "/api/logout", nil)
			req.AddCookie(&http.Cookie{Name: "session_token", Value: "kein.gueltiges.token"})
			rec := httptest.NewRecorder()
			s.logoutHandler()(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("erwartet 200, bekam %d: %s", rec.Code, rec.Body.String())
			}

			var gefunden *http.Cookie
			for _, c := range rec.Result().Cookies() {
				if c.Name == "session_token" {
					gefunden = c
				}
			}
			if gefunden == nil {
				t.Fatalf("kein Loeschcookie in der Antwort: %+v", rec.Result().Cookies())
			}
			if gefunden.Secure != secure {
				t.Errorf("Secure = %v, erwartet %v — das Flag folgt der Konfiguration nicht",
					gefunden.Secure, secure)
			}
			if !gefunden.HttpOnly {
				t.Error("Loeschcookie muss HttpOnly sein wie das Cookie, das es ersetzt")
			}
			if gefunden.SameSite != http.SameSiteStrictMode {
				t.Errorf("SameSite = %v, erwartet Strict", gefunden.SameSite)
			}
			if gefunden.MaxAge >= 0 {
				t.Errorf("MaxAge = %d — das Cookie muss geloescht werden", gefunden.MaxAge)
			}
			if gefunden.Value != "" {
				t.Errorf("Loeschcookie traegt noch einen Wert: %q", gefunden.Value)
			}
		})
	}
}
