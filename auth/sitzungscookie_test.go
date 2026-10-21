package auth

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/pashagolub/pgxmock/v5"
)

// Die Tests prüfen, dass der ermittelte Wert für Secure am Cookie der Sitzung ankommt.
// cookie_secure_test.go prüft nur, wie ermittleCookieSecure() APP_ENV und COOKIE_SECURE
// auswertet, nicht, was die Türen damit machen; für das CSRF-Cookie steht der Beleg in
// api/csrf_cookie_test.go.
//
// Ersetzt man `Secure: nurHTTPS` in Anmelden und SitzungVerlaengern durch ein festes `false`,
// bliebe die übrige Go-Suite grün. Die Tests hier lesen die Kopfzeile Set-Cookie, die der
// Browser bekommt, in beide Richtungen: Ein fest verdrahtetes true fiele ebenso auf wie ein fest
// verdrahtetes false.

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

func TestAnmelden_CookieFolgtDerKonfiguration(t *testing.T) {
	for _, nurHTTPS := range []bool{true, false} {
		t.Run(map[bool]string{true: "secure=true", false: "secure=false"}[nurHTTPS], func(t *testing.T) {
			aktiviereIMAPAttrappe(t)
			a, mock := newTestAnmeldung(t, 12*time.Hour)
			mock.ExpectQuery(benutzerSelect).
				WithArgs("nberger@schule.de").
				WillReturnRows(pgxmock.NewRows([]string{"id", "rolle", "vorname", "nachname", "aktiv", "email", "beantragt"}).
					AddRow("u-admin", "admin", "Nina", "Berger", true, "nina@example.org", false))
			erwarteSitzungAngelegt(mock, "u-admin")

			anfrage := httptest.NewRequest(http.MethodPost, "/login",
				strings.NewReader(`{"email":"nberger@schule.de","password":"egal"}`))
			rec := httptest.NewRecorder()
			Anmelden(mock, a, nurHTTPS)(rec, anfrage)

			if rec.Code != http.StatusOK {
				t.Fatalf("rec.Code = %d, erwartet 200: %s", rec.Code, rec.Body.String())
			}
			c := sitzungsCookie(t, rec)
			if c.Secure != nurHTTPS {
				t.Errorf("Secure = %v, erwartet %v — das Flag folgt der Konfiguration nicht", c.Secure, nurHTTPS)
			}
			pruefeSchutzattribute(t, c)
		})
	}
}

func TestSitzungVerlaengern_CookieFolgtDerKonfiguration(t *testing.T) {
	for _, nurHTTPS := range []bool{true, false} {
		t.Run(map[bool]string{true: "secure=true", false: "secure=false"}[nurHTTPS], func(t *testing.T) {
			// Eine Stunde übrig bei zwölf Stunden Laufzeit: Die Tür verlängert und stellt ein neues
			// Cookie aus.
			aussteller, _ := newTestAnmeldung(t, 1*time.Hour)
			token, err := aussteller.ErzeugeToken("user-1", RolleMitarbeiter, testSitzungID)
			if err != nil {
				t.Fatalf("ErzeugeToken: %v", err)
			}
			a, mock := newTestAnmeldung(t, 12*time.Hour)
			erwarteNichtWiderrufen(mock)
			erwarteKontoStatus(mock, true, RolleMitarbeiter)
			erwarteSitzungVerlaengert(mock, 1)

			anfrage := httptest.NewRequest(http.MethodPost, "/api/auth/refresh", nil)
			anfrage.AddCookie(&http.Cookie{Name: "session_token", Value: token})
			rec := httptest.NewRecorder()
			SitzungVerlaengern(a, nurHTTPS)(rec, anfrage)

			if rec.Code != http.StatusOK {
				t.Fatalf("rec.Code = %d, erwartet 200: %s", rec.Code, rec.Body.String())
			}
			c := sitzungsCookie(t, rec)
			if c.Secure != nurHTTPS {
				t.Errorf("Secure = %v, erwartet %v — das Flag folgt der Konfiguration nicht", c.Secure, nurHTTPS)
			}
			pruefeSchutzattribute(t, c)
		})
	}
}
