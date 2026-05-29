package auth

import (
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestVerifyPassword(t *testing.T) {
	// 1. Ensure the dummy backdoor is removed
	if verifyPassword("dummy_passwort_hash", "admin") {
		t.Errorf("Expected verifyPassword with backdoor credentials to return false, got true")
	}

	// 2. Test valid bcrypt
	password := "my_secure_password"
	hashBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("Failed to generate bcrypt hash: %v", err)
	}
	hash := string(hashBytes)

	if !verifyPassword(hash, password) {
		t.Errorf("Expected verifyPassword to return true for correct password")
	}

	// 3. Test invalid bcrypt
	if verifyPassword(hash, "wrong_password") {
		t.Errorf("Expected verifyPassword to return false for incorrect password")
	}
}
