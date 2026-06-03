package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Role defines the authorization levels/roles in the library system.
type Role string

const (
	// RoleAdmin has full permissions for configuration and master-data editing.
	RoleAdmin Role = "ADMIN"
	// RoleLehrer represents teachers who can borrow books and trigger class plans.
	RoleLehrer Role = "LEHRER"
	// RoleMitarbeiter represents library staff executing daily lending operations.
	RoleMitarbeiter Role = "MITARBEITER"
	// RoleHelfer represents helpers executing kiosk checkouts and quick returns.
	RoleHelfer Role = "HELFER"
)

// MarshalJSON converts the uppercase Role constant to a lowercase string for Svelte frontend compatibility.
func (r Role) MarshalJSON() ([]byte, error) {
	return json.Marshal(strings.ToLower(string(r)))
}

// UnmarshalJSON parses a lowercase or uppercase string from JSON into the uppercase Role constant.
func (r *Role) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	*r = Role(strings.ToUpper(s))
	return nil
}

// Claims represents the structure of the JWT payload.
type Claims struct {
	UserID    string `json:"user_id"`
	BarcodeID string `json:"barcode_id"`
	Rolle     Role   `json:"rolle"`
	jwt.RegisteredClaims
}

// Authenticator handles creation and parsing of session JSON Web Tokens.
type Authenticator struct {
	secretKey     []byte
	tokenDuration time.Duration
	Blacklist     *TokenBlacklist
}

// NewAuthenticator creates a new JWT Authenticator instance with the given secret and duration.
func NewAuthenticator(secret string, duration time.Duration) (*Authenticator, error) {
	if len(secret) < 32 {
		return nil, errors.New("JWT secret must be at least 32 bytes for security")
	}
	return &Authenticator{
		secretKey:     []byte(secret),
		tokenDuration: duration,
		Blacklist:     NewTokenBlacklist(),
	}, nil
}

// GenerateToken generates a signed JWT containing user identity and role.
func (a *Authenticator) GenerateToken(userID, barcodeID string, role Role) (string, error) {
	claims := Claims{
		UserID:    userID,
		BarcodeID: barcodeID,
		Rolle:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(a.tokenDuration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "bibliothek-system",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString(a.secretKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return signedToken, nil
}

// VerifyToken parses and validates the provided JWT string and returns its claims.
// It also checks if the token has been revoked in the server-side blacklist.
func (a *Authenticator) VerifyToken(tokenString string) (*Claims, error) {
	if a.Blacklist.IsBlacklisted(tokenString) {
		return nil, errors.New("token has been revoked (logged out)")
	}

	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return a.secretKey, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token claims")
	}

	return claims, nil
}
