package auth

import (
	"testing"
	"time"

	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
)

func TestNewAuthenticator(t *testing.T) {
	mockPool, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mockPool.Close()

	t.Run("Valid Secret", func(t *testing.T) {
		secret := "12345678901234567890123456789012" // 32 bytes
		auth, err := NewAuthenticator(secret, mockPool, time.Hour)
		assert.NoError(t, err)
		assert.NotNil(t, auth)
		assert.NotNil(t, auth.Blacklist)
		auth.Blacklist.Stop()
	})

	t.Run("Invalid Secret", func(t *testing.T) {
		secret := "short-secret"
		auth, err := NewAuthenticator(secret, mockPool, time.Hour)
		assert.Error(t, err)
		assert.Nil(t, auth)
		assert.Equal(t, "JWT secret must be at least 32 bytes for security", err.Error())
	})
}

func TestGenerateToken(t *testing.T) {
	mockPool, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mockPool.Close()

	secret := "12345678901234567890123456789012"
	auth, _ := NewAuthenticator(secret, mockPool, time.Hour)
	defer auth.Blacklist.Stop()

	token, err := auth.GenerateToken("user-123", "B-12345", RoleLehrer)
	assert.NoError(t, err)
	assert.NotEmpty(t, token)
}

func TestVerifyToken(t *testing.T) {
	mockPool, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mockPool.Close()

	secret := "12345678901234567890123456789012"
	auth, _ := NewAuthenticator(secret, mockPool, time.Hour)
	defer auth.Blacklist.Stop()

	userID := "user-123"
	barcodeID := "B-12345"
	role := RoleLehrer

	token, err := auth.GenerateToken(userID, barcodeID, role)
	assert.NoError(t, err)

	t.Run("Valid Token Not Blacklisted", func(t *testing.T) {
		// Mock the query to IsBlacklisted which returns false (not revoked)
		mockPool.ExpectQuery("SELECT EXISTS").WithArgs(pgxmock.AnyArg()).WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(false))

		claims, err := auth.VerifyToken(token)
		assert.NoError(t, err)
		assert.NotNil(t, claims)
		assert.Equal(t, userID, claims.UserID)
		assert.Equal(t, barcodeID, claims.BarcodeID)
		assert.Equal(t, role, claims.Rolle)

		if err := mockPool.ExpectationsWereMet(); err != nil {
			t.Errorf("there were unfulfilled expectations: %s", err)
		}
	})

	t.Run("Revoked Token", func(t *testing.T) {
		// Mock the query to IsBlacklisted which returns true (revoked)
		mockPool.ExpectQuery("SELECT EXISTS").WithArgs(pgxmock.AnyArg()).WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(true))

		claims, err := auth.VerifyToken(token)
		assert.Error(t, err)
		assert.Nil(t, claims)
		assert.Equal(t, "token has been revoked (logged out)", err.Error())

		if err := mockPool.ExpectationsWereMet(); err != nil {
			t.Errorf("there were unfulfilled expectations: %s", err)
		}
	})

	t.Run("Invalid Token Format", func(t *testing.T) {
		// Invalid token will not reach IsBlacklisted if we pass a random string, wait, yes it does. IsBlacklisted just checks the hash.
		// Wait, let's see IsBlacklisted. It hashes whatever string you pass. So yes, IsBlacklisted is called.
		mockPool.ExpectQuery("SELECT EXISTS").WithArgs(pgxmock.AnyArg()).WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(false))

		claims, err := auth.VerifyToken("invalid-token-string")
		assert.Error(t, err)
		assert.Nil(t, claims)
		assert.Contains(t, err.Error(), "failed to parse token")

		if err := mockPool.ExpectationsWereMet(); err != nil {
			t.Errorf("there were unfulfilled expectations: %s", err)
		}
	})
}
