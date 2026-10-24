package auth

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/pashagolub/pgxmock/v5"
)

// Die Tests lesen das Cookie der Sitzung so, wie der Browser es bekommt. Fehlte Secure in Anmelden
// oder SitzungVerlaengern, bliebe die übrige Go-Suite grün, und das Cookie ginge im Schulnetz auch
// über eine unverschlüsselte Verbindung. Für das CSRF-Cookie steht der Beleg in
// api/csrf_cookie_test.go, für das Löschcookie in api/abmeldung_cookie_test.go.

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

// pruefeSchutzattribute hält fest, was für jedes Cookie der Sitzung gilt.
func pruefeSchutzattribute(t *testing.T, c *http.Cookie) {
	t.Helper()
	if !c.Secure {
		t.Error("Sitzungscookie muss Secure sein — sonst schickt der Browser es auch über HTTP")
	}
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

func TestAnmelden_CookieNurUeberHTTPS(t *testing.T) {
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
	Anmelden(mock, a)(rec, anfrage)

	if rec.Code != http.StatusOK {
		t.Fatalf("rec.Code = %d, erwartet 200: %s", rec.Code, rec.Body.String())
	}
	pruefeSchutzattribute(t, sitzungsCookie(t, rec))
}

func TestSitzungVerlaengern_CookieNurUeberHTTPS(t *testing.T) {
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
	erwarteSperrzustand(mock, false)
	erwarteSitzungVerlaengert(mock, 1)

	anfrage := httptest.NewRequest(http.MethodPost, "/api/auth/refresh", nil)
	anfrage.AddCookie(&http.Cookie{Name: "session_token", Value: token})
	rec := httptest.NewRecorder()
	SitzungVerlaengern(a)(rec, anfrage)

	if rec.Code != http.StatusOK {
		t.Fatalf("rec.Code = %d, erwartet 200: %s", rec.Code, rec.Body.String())
	}
	pruefeSchutzattribute(t, sitzungsCookie(t, rec))
}
