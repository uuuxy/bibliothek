package auth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/pashagolub/pgxmock/v4"
)

func doMe(t *testing.T, a *Authenticator, mock pgxmock.PgxPoolIface, cookie *http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	if cookie != nil {
		req.AddCookie(cookie)
	}
	rec := httptest.NewRecorder()
	MeHandler(mock, a)(rec, req)
	return rec
}

func TestMeHandler_NoCookieReturns401(t *testing.T) {
	a, mock := newTestAuthenticator(t, 12*time.Hour)
	rec := doMe(t, a, mock, nil)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("erwartet 401, bekam %d", rec.Code)
	}
}

func TestMeHandler_ActiveAdminGetsLoginShape(t *testing.T) {
	a, mock := newTestAuthenticator(t, 12*time.Hour)
	token, err := a.GenerateToken("user-1", "B-1", RoleAdmin)
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}

	expectNotBlacklisted(mock)
	expectKontoAktiv(mock, true)
	mock.ExpectQuery(`SELECT rolle, vorname, nachname, aktiv`).
		WithArgs("user-1").
		WillReturnRows(pgxmock.NewRows([]string{"rolle", "vorname", "nachname", "aktiv"}).
			AddRow("admin", "Peter", "Flasch", true))
	// Admin bekommt implizit "*" — kein role_permissions-Query

	rec := doMe(t, a, mock, &http.Cookie{Name: "session_token", Value: token})
	if rec.Code != http.StatusOK {
		t.Fatalf("erwartet 200, bekam %d: %s", rec.Code, rec.Body.String())
	}

	var resp LoginResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Antwort kein LoginResponse-JSON: %v", err)
	}
	if resp.UserID != "user-1" || resp.Rolle != RoleAdmin || resp.Vorname != "Peter" || resp.Nachname != "Flasch" {
		t.Errorf("Stammdaten falsch: %+v", resp)
	}
	if len(resp.Permissions) != 1 || resp.Permissions[0] != "*" {
		t.Errorf("Admin muss implizit '*' bekommen: %+v", resp.Permissions)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("offene Erwartungen: %v", err)
	}
}

func TestMeHandler_NonAdminLoadsConfiguredPermissions(t *testing.T) {
	a, mock := newTestAuthenticator(t, 12*time.Hour)
	token, err := a.GenerateToken("user-2", "B-2", RoleMitarbeiter)
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}

	expectNotBlacklisted(mock)
	expectKontoAktiv(mock, true)
	mock.ExpectQuery(`SELECT rolle, vorname, nachname, aktiv`).
		WithArgs("user-2").
		WillReturnRows(pgxmock.NewRows([]string{"rolle", "vorname", "nachname", "aktiv"}).
			AddRow("mitarbeiter", "Mia", "Muster", true))
	mock.ExpectQuery(`SELECT permission`).
		WithArgs("mitarbeiter").
		WillReturnRows(pgxmock.NewRows([]string{"permission"}).
			AddRow("view_books").
			AddRow("view_students"))

	rec := doMe(t, a, mock, &http.Cookie{Name: "session_token", Value: token})
	if rec.Code != http.StatusOK {
		t.Fatalf("erwartet 200, bekam %d: %s", rec.Code, rec.Body.String())
	}

	var resp LoginResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Antwort kein LoginResponse-JSON: %v", err)
	}
	if len(resp.Permissions) != 2 || resp.Permissions[0] != "view_books" {
		t.Errorf("konfigurierte Rechte nicht durchgereicht: %+v", resp.Permissions)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("offene Erwartungen: %v", err)
	}
}

func TestMeHandler_DeactivatedUserReturns401(t *testing.T) {
	// Rolle/Aktiv kommen aus der DB, nicht aus den Claims: Ein zwischenzeitlich
	// deaktivierter Benutzer darf seine Session nicht wiederherstellen.
	a, mock := newTestAuthenticator(t, 12*time.Hour)
	token, err := a.GenerateToken("user-3", "B-3", RoleMitarbeiter)
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}

	expectNotBlacklisted(mock)
	// VerifyToken lehnt den deaktivierten Benutzer bereits ab (aktiv=false) — MeHandlers
	// eigene Stammdaten-Query wird gar nicht mehr erreicht.
	expectKontoAktiv(mock, false)

	rec := doMe(t, a, mock, &http.Cookie{Name: "session_token", Value: token})
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("erwartet 401 für deaktivierten Benutzer, bekam %d", rec.Code)
	}
}
