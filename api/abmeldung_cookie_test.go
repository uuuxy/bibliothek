package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"bibliothek/api/tuer"
	"bibliothek/auth"
	"bibliothek/db"

	"github.com/pashagolub/pgxmock/v5"
)

// Die dritte Stelle, die das Sitzungscookie setzt, neben Anmeldung und Erneuern (siehe
// auth/session_cookie_secure_test.go): das Löschcookie beim Abmelden.
//
// Es trägt keinen Wert mehr, die Schutzattribute zählen trotzdem. Ein Löschcookie ohne Secure
// überschreibt über HTTP ein Secure-Cookie nicht zuverlässig; die Abmeldung wäre dann eine, die
// der Browser still ignoriert. Deshalb muss auch hier der eingestellte Wert ankommen statt
// eines fest verdrahteten.
func TestAbmeldungCookieFolgtDerKonfiguration(t *testing.T) {
	for _, secure := range []bool{true, false} {
		t.Run(map[bool]string{true: "secure=true", false: "secure=false"}[secure], func(t *testing.T) {
			mock, err := pgxmock.NewPool()
			if err != nil {
				t.Fatalf("pgxmock init: %v", err)
			}
			t.Cleanup(mock.Close)
			a, err := auth.NewAuthenticator(testJWTGeheimnis, mock, time.Hour)
			if err != nil {
				t.Fatalf("authenticator init: %v", err)
			}
			t.Cleanup(a.Blacklist.Stop)
			s := &Server{Basis: tuer.Basis{DB: &db.Database{Pool: mock}, Auth: a}, nurHTTPS: secure}

			// Unbrauchbares Token: Die Sperrliste antwortet, die Prüfung des Tokens scheitert
			// danach an der Signatur. Es gibt nichts zu widerrufen, die Tür löscht das Cookie
			// und meldet 200. (Antwortet die Datenbank nicht, ist die Abmeldung unvollständig:
			// abmeldung_widerruf_test.go.)
			mock.ExpectQuery("revoked_tokens").
				WithArgs(pgxmock.AnyArg()).
				WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(false))
			req := httptest.NewRequest(http.MethodPost, "/api/logout", nil)
			req.AddCookie(&http.Cookie{Name: "session_token", Value: "kein.gueltiges.token"})
			rec := httptest.NewRecorder()
			s.abmelden()(rec, req)

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
