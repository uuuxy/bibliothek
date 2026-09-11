package auth

import (
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/pashagolub/pgxmock/v4"
)

// Kann die Datenbank bei der Token-Prüfung nicht antworten, ist das keine abgelaufene
// Sitzung. Bis zum 11.09.2026 antworteten Me und Refresh dann mit 401; der Client meldete
// daraufhin ab (authStore: Refresh-401 → handleLogout), und ein kurzer Datenbank-Aussetzer
// warf alle Arbeitsplätze hinaus. Die Anfrage bleibt verweigert, aber mit 503.
func TestTokenPruefungGestoert_Ist503(t *testing.T) {
	faelle := []struct {
		name  string
		mocks func(pgxmock.PgxPoolIface)
	}{
		{"Sperrliste nicht erreichbar", func(m pgxmock.PgxPoolIface) {
			m.ExpectQuery(`SELECT EXISTS`).WithArgs(pgxmock.AnyArg()).
				WillReturnError(errors.New("connection reset by peer"))
		}},
		{"Kontostatus nicht erreichbar", func(m pgxmock.PgxPoolIface) {
			expectNotBlacklisted(m)
			m.ExpectQuery(`SELECT aktiv, rolle FROM benutzer`).WithArgs(pgxmock.AnyArg()).
				WillReturnError(errors.New("context deadline exceeded"))
		}},
	}
	for _, f := range faelle {
		t.Run("Me/"+f.name, func(t *testing.T) {
			a, mock := newTestAuthenticator(t, 12*time.Hour)
			token, err := a.GenerateToken("user-1", "B-1", RoleAdmin)
			if err != nil {
				t.Fatal(err)
			}
			f.mocks(mock)
			rec := doMe(t, a, mock, &http.Cookie{Name: "session_token", Value: token})
			if rec.Code != http.StatusServiceUnavailable {
				t.Errorf("Status %d, want 503 — eine 401 meldet den Arbeitsplatz ab", rec.Code)
			}
		})
		t.Run("Refresh/"+f.name, func(t *testing.T) {
			a, mock := newTestAuthenticator(t, 12*time.Hour)
			token, err := a.GenerateToken("user-1", "B-1", RoleAdmin)
			if err != nil {
				t.Fatal(err)
			}
			f.mocks(mock)
			rec := doRefresh(t, a, &http.Cookie{Name: "session_token", Value: token})
			if rec.Code != http.StatusServiceUnavailable {
				t.Errorf("Status %d, want 503 — eine 401 meldet den Arbeitsplatz ab", rec.Code)
			}
		})
	}
}
