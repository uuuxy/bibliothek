package auth

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/pashagolub/pgxmock/v4"
)

const testSecret = "test-secret-with-at-least-32-bytes!!"

// newTestAuthenticator baut einen Authenticator mit gemocktem Blacklist-Pool.
// VerifyToken prüft die Blacklist per SELECT — der Mock antwortet "nicht gesperrt".
func newTestAuthenticator(t *testing.T, duration time.Duration) (*Authenticator, pgxmock.PgxPoolIface) {
	t.Helper()
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock: %v", err)
	}
	t.Cleanup(mock.Close)
	a, err := NewAuthenticator(testSecret, mock, duration)
	if err != nil {
		t.Fatalf("NewAuthenticator: %v", err)
	}
	t.Cleanup(a.Blacklist.Stop)
	return a, mock
}

func expectNotBlacklisted(mock pgxmock.PgxPoolIface) {
	mock.ExpectQuery(`SELECT EXISTS`).
		WithArgs(pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(false))
}

func doRefresh(t *testing.T, a *Authenticator, cookie *http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/auth/refresh", nil)
	if cookie != nil {
		req.AddCookie(cookie)
	}
	rec := httptest.NewRecorder()
	RefreshTokenHandler(a, false)(rec, req)
	return rec
}

func TestRefreshTokenHandler_NoCookieReturns401(t *testing.T) {
	a, _ := newTestAuthenticator(t, 12*time.Hour)
	rec := doRefresh(t, a, nil)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("erwartet 401, bekam %d", rec.Code)
	}
}

func TestRefreshTokenHandler_InvalidTokenReturns401(t *testing.T) {
	a, mock := newTestAuthenticator(t, 12*time.Hour)
	expectNotBlacklisted(mock)
	rec := doRefresh(t, a, &http.Cookie{Name: "session_token", Value: "kein-jwt"})
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("erwartet 401, bekam %d", rec.Code)
	}
}

func TestRefreshTokenHandler_FreshTokenIsSkippedWithoutNewCookie(t *testing.T) {
	// Frisch ausgestelltes Token: Restlaufzeit ≈ 100% > 50% → kein Refresh.
	a, mock := newTestAuthenticator(t, 12*time.Hour)
	token, err := a.GenerateToken("user-1", "B-1", RoleAdmin)
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}
	expectNotBlacklisted(mock)

	rec := doRefresh(t, a, &http.Cookie{Name: "session_token", Value: token})

	if rec.Code != http.StatusOK {
		t.Fatalf("erwartet 200, bekam %d: %s", rec.Code, rec.Body.String())
	}
	if got := rec.Body.String(); !strings.Contains(got, `"refresh":"skipped"`) {
		t.Errorf("erwartet refresh=skipped, Body: %s", got)
	}
	if len(rec.Result().Cookies()) != 0 {
		t.Errorf("bei skipped darf kein neues Cookie gesetzt werden")
	}
}

func TestRefreshTokenHandler_OldTokenIsRenewedWithNewCookie(t *testing.T) {
	// Token wurde mit kurzer Laufzeit ausgestellt (Restlaufzeit 1h). Der Handler
	// läuft mit 12h-Fenster: 1h < 6h → Sliding Window greift, neues Cookie.
	issuer, _ := newTestAuthenticator(t, 1*time.Hour)
	token, err := issuer.GenerateToken("user-1", "B-1", RoleMitarbeiter)
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}

	a, mock := newTestAuthenticator(t, 12*time.Hour)
	expectNotBlacklisted(mock)

	rec := doRefresh(t, a, &http.Cookie{Name: "session_token", Value: token})

	if rec.Code != http.StatusOK {
		t.Fatalf("erwartet 200, bekam %d: %s", rec.Code, rec.Body.String())
	}
	if got := rec.Body.String(); !strings.Contains(got, `"refresh":"renewed"`) {
		t.Errorf("erwartet refresh=renewed, Body: %s", got)
	}

	cookies := rec.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Name != "session_token" || cookies[0].Value == "" {
		t.Fatalf("erwartet neues session_token-Cookie, bekam %+v", cookies)
	}
	if !cookies[0].HttpOnly {
		t.Errorf("Session-Cookie muss HttpOnly sein")
	}

	// Das neue Token muss gültig sein und die Claims unverändert tragen.
	expectNotBlacklisted(mock)
	claims, err := a.VerifyToken(cookies[0].Value)
	if err != nil {
		t.Fatalf("neues Token ungültig: %v", err)
	}
	if claims.UserID != "user-1" || claims.Rolle != RoleMitarbeiter {
		t.Errorf("Claims nicht übernommen: %+v", claims)
	}
}

func TestRefreshTokenHandler_BlacklistedTokenReturns401(t *testing.T) {
	// Logout blacklistet das Token — danach darf Refresh es nicht wiederbeleben.
	a, mock := newTestAuthenticator(t, 12*time.Hour)
	token, err := a.GenerateToken("user-1", "B-1", RoleAdmin)
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}

	mock.ExpectQuery(`SELECT EXISTS`).
		WithArgs(pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(true))

	rec := doRefresh(t, a, &http.Cookie{Name: "session_token", Value: token})
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("erwartet 401 für widerrufenes Token, bekam %d", rec.Code)
	}
}
