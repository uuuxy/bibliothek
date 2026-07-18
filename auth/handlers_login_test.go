package auth

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/pashagolub/pgxmock/v4"
)

const benutzerSelect = `SELECT id, rolle, vorname, nachname, aktiv`

func doLogin(t *testing.T, a *Authenticator, mock pgxmock.PgxPoolIface, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	LoginHandler(mock, a, false)(rec, req)
	return rec
}

func TestLoginHandler_MissingCredentialsReturn400(t *testing.T) {
	t.Setenv("IMAP_HOST", "mock")
	a, mock := newTestAuthenticator(t, 12*time.Hour)

	if rec := doLogin(t, a, mock, `{"password":"x"}`); rec.Code != http.StatusBadRequest {
		t.Errorf("ohne email: erwartet 400, bekam %d", rec.Code)
	}
	if rec := doLogin(t, a, mock, `{"email":"a@b.de"}`); rec.Code != http.StatusBadRequest {
		t.Errorf("ohne password: erwartet 400, bekam %d", rec.Code)
	}
}

func TestLoginHandler_UnknownUserReturns401(t *testing.T) {
	// Mock-IMAP akzeptiert jedes Passwort — die Ablehnung muss aus der DB kommen
	// (IMAP-Konto existiert, aber kein registrierter Bibliotheks-Benutzer).
	t.Setenv("IMAP_HOST", "mock")
	a, mock := newTestAuthenticator(t, 12*time.Hour)

	mock.ExpectQuery(benutzerSelect).
		WithArgs("unbekannt@schule.de").
		WillReturnRows(pgxmock.NewRows([]string{"id", "rolle", "vorname", "nachname", "aktiv"}))

	rec := doLogin(t, a, mock, `{"email":"unbekannt@schule.de","password":"egal"}`)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("erwartet 401, bekam %d: %s", rec.Code, rec.Body.String())
	}
	if len(rec.Result().Cookies()) != 0 {
		t.Errorf("bei fehlgeschlagenem Login darf kein Cookie gesetzt werden")
	}
}

func TestLoginHandler_DeactivatedUserReturns403(t *testing.T) {
	t.Setenv("IMAP_HOST", "mock")
	a, mock := newTestAuthenticator(t, 12*time.Hour)

	mock.ExpectQuery(benutzerSelect).
		WithArgs("inaktiv@schule.de").
		WillReturnRows(pgxmock.NewRows([]string{"id", "rolle", "vorname", "nachname", "aktiv"}).
			AddRow("u-1", "mitarbeiter", "Ex", "Kollege", false))

	rec := doLogin(t, a, mock, `{"email":"inaktiv@schule.de","password":"egal"}`)
	if rec.Code != http.StatusForbidden {
		t.Errorf("erwartet 403, bekam %d: %s", rec.Code, rec.Body.String())
	}
}

func TestLoginHandler_SuccessSetsCookieAndReturnsLoginShape(t *testing.T) {
	t.Setenv("IMAP_HOST", "mock")
	a, mock := newTestAuthenticator(t, 12*time.Hour)

	mock.ExpectQuery(benutzerSelect).
		WithArgs("pflasch@schule.de").
		WillReturnRows(pgxmock.NewRows([]string{"id", "rolle", "vorname", "nachname", "aktiv"}).
			AddRow("u-admin", "admin", "Peter", "Flasch", true))

	rec := doLogin(t, a, mock, `{"email":"pflasch@schule.de","password":"egal"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("erwartet 200, bekam %d: %s", rec.Code, rec.Body.String())
	}

	cookies := rec.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Name != "session_token" || cookies[0].Value == "" {
		t.Fatalf("erwartet session_token-Cookie, bekam %+v", cookies)
	}
	if !cookies[0].HttpOnly {
		t.Errorf("Session-Cookie muss HttpOnly sein")
	}

	var resp LoginResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Antwort kein LoginResponse-JSON: %v", err)
	}
	if resp.UserID != "u-admin" || resp.Rolle != RoleAdmin || resp.Vorname != "Peter" {
		t.Errorf("LoginResponse falsch: %+v", resp)
	}
	if len(resp.Permissions) != 1 || resp.Permissions[0] != "*" {
		t.Errorf("Admin muss implizit '*' bekommen: %+v", resp.Permissions)
	}

	// Das ausgestellte Token muss verifizierbar sein und die Identität tragen.
	expectNotBlacklisted(mock)
	expectKontoAktiv(mock, true)
	claims, err := a.VerifyToken(cookies[0].Value)
	if err != nil {
		t.Fatalf("ausgestelltes Token ungültig: %v", err)
	}
	if claims.UserID != "u-admin" || claims.Rolle != RoleAdmin {
		t.Errorf("Claims falsch: %+v", claims)
	}
}

func TestLoginHandler_BruteForceLimiterBlocksSixthAttempt(t *testing.T) {
	// Der Limiter drosselt pro (E-Mail|IP) — 5 Fehlversuche, dann 429.
	// Eindeutige E-Mail, da der Limiter prozessweit global ist.
	t.Setenv("IMAP_HOST", "mock")
	a, mock := newTestAuthenticator(t, 12*time.Hour)

	email := fmt.Sprintf("brute-%d@schule.de", time.Now().UnixNano())
	body := fmt.Sprintf(`{"email":%q,"password":"falsch"}`, email)

	for i := 1; i <= 5; i++ {
		mock.ExpectQuery(benutzerSelect).
			WithArgs(email).
			WillReturnRows(pgxmock.NewRows([]string{"id", "rolle", "vorname", "nachname", "aktiv"}))
		if rec := doLogin(t, a, mock, body); rec.Code != http.StatusUnauthorized {
			t.Fatalf("Versuch %d: erwartet 401, bekam %d", i, rec.Code)
		}
	}

	// 6. Versuch: geblockt, OHNE dass die DB noch gefragt wird
	rec := doLogin(t, a, mock, body)
	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("erwartet 429 nach 5 Fehlversuchen, bekam %d", rec.Code)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("offene Erwartungen (6. Versuch hätte die DB nicht erreichen dürfen): %v", err)
	}
}
