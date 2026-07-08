package auth

import (
	"testing"
	"time"
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
