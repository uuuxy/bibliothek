package auth

import (
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
)

// Regressionstest: Ohne jti waren zwei Logins desselben Kontos innerhalb
// derselben Sekunde byte-identisch — der Logout des einen widerrief über die
// hash-basierte Blacklist auch die Session des anderen.
func TestGenerateToken_UniquePerCall(t *testing.T) {
	a, _ := newTestAuthenticator(t, 12*time.Hour)

	t1, err := a.GenerateToken("user-1", "B-1", RoleAdmin)
	if err != nil {
		t.Fatalf("GenerateToken 1: %v", err)
	}
	t2, err := a.GenerateToken("user-1", "B-1", RoleAdmin)
	if err != nil {
		t.Fatalf("GenerateToken 2: %v", err)
	}

	if t1 == t2 {
		t.Fatalf("zwei Tokens mit identischen Claims dürfen nie byte-identisch sein (jti fehlt?)")
	}
}

// TestVerifyToken_RealtimeRevocation sichert den Echtzeit-Sitzungswiderruf ab: Ein gültig
// signiertes, nicht geblacklistetes Token wird trotzdem abgelehnt, sobald das Konto in der
// DB deaktiviert (aktiv=false) oder gelöscht wurde (keine Zeile). Ohne diese Prüfung behielte
// ein gefeuerter Mitarbeiter bis zum Token-Ablauf (12h) vollen Zugriff.
func TestVerifyToken_RealtimeRevocation(t *testing.T) {
	newToken := func(t *testing.T, a *Authenticator) string {
		t.Helper()
		tok, err := a.GenerateToken("user-1", "B-1", RoleAdmin)
		if err != nil {
			t.Fatalf("GenerateToken: %v", err)
		}
		return tok
	}

	t.Run("aktives Konto wird akzeptiert", func(t *testing.T) {
		a, mock := newTestAuthenticator(t, 12*time.Hour)
		token := newToken(t, a)
		expectNotBlacklisted(mock)
		expectKontoAktiv(mock, true)

		if _, err := a.VerifyToken(token); err != nil {
			t.Errorf("aktives Konto: unerwarteter Fehler: %v", err)
		}
	})

	t.Run("deaktiviertes Konto wird abgelehnt", func(t *testing.T) {
		a, mock := newTestAuthenticator(t, 12*time.Hour)
		token := newToken(t, a)
		expectNotBlacklisted(mock)
		expectKontoAktiv(mock, false)

		if _, err := a.VerifyToken(token); err == nil {
			t.Error("deaktiviertes Konto muss abgelehnt werden")
		}
	})

	t.Run("geloeschtes Konto (keine Zeile) wird abgelehnt", func(t *testing.T) {
		a, mock := newTestAuthenticator(t, 12*time.Hour)
		token := newToken(t, a)
		expectNotBlacklisted(mock)
		mock.ExpectQuery(`SELECT aktiv FROM benutzer`).
			WithArgs(pgxmock.AnyArg()).
			WillReturnError(pgx.ErrNoRows)

		if _, err := a.VerifyToken(token); err == nil {
			t.Error("gelöschtes Konto muss abgelehnt werden")
		}
	})
}
