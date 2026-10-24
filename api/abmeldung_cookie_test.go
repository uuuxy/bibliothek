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
// auth/sitzungscookie_test.go): das Löschcookie beim Abmelden.
//
// Es trägt keinen Wert mehr, die Schutzattribute zählen trotzdem: Ein Löschcookie ohne Secure
// überschreibt ein Secure-Cookie nicht zuverlässig, die Abmeldung wäre dann eine, die der Browser
// still ignoriert.
func TestAbmeldungCookieNurUeberHTTPS(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock init: %v", err)
	}
	t.Cleanup(mock.Close)
	a, err := auth.NewAnmeldung(testJWTGeheimnis, mock, time.Hour)
	if err != nil {
		t.Fatalf("authenticator init: %v", err)
	}
	s := &Server{Basis: tuer.Basis{DB: &db.Verbindung{Pool: mock}, Auth: a}}

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
		t.Fatalf("rec.Code = %d, erwartet 200: %s", rec.Code, rec.Body.String())
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
	if !gefunden.Secure {
		t.Error("Loeschcookie muss Secure sein wie das Cookie, das es ersetzt")
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
}
