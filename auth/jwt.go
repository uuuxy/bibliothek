package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5"
)

// Role definiert die Berechtigungsstufen/Rollen im Bibliothekssystem.
type Role string

const (
	// RoleAdmin hat volle Berechtigungen für Konfiguration und Stammdaten-Bearbeitung.
	RoleAdmin Role = "ADMIN"
	// RoleLehrer repräsentiert Lehrer, die Bücher ausleihen und Klassenpläne auslösen können.
	RoleLehrer Role = "LEHRER"
	// RoleMitarbeiter repräsentiert Bibliotheksmitarbeiter, die das tägliche Ausleihgeschäft durchführen.
	RoleMitarbeiter Role = "MITARBEITER"
	// RoleHelfer repräsentiert Helfer, die Kiosk-Ausleihen und schnelle Rückgaben durchführen.
	RoleHelfer Role = "HELFER"
)

// MarshalJSON konvertiert die großgeschriebene Role-Konstante in einen kleingeschriebenen String für Svelte-Frontend-Kompatibilität.
func (r Role) MarshalJSON() ([]byte, error) {
	return json.Marshal(strings.ToLower(string(r)))
}

// UnmarshalJSON parst einen klein- oder großgeschriebenen String aus JSON in die großgeschriebene Role-Konstante.
func (r *Role) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	*r = Role(strings.ToUpper(s))
	return nil
}

// Claims repräsentiert die Struktur des JWT-Payloads.
type Claims struct {
	UserID    string `json:"user_id"`
	BarcodeID string `json:"barcode_id"`
	Rolle     Role   `json:"rolle"`
	jwt.RegisteredClaims
}

// Authenticator verarbeitet die Erstellung und das Parsen von Session-JSON Web Tokens.
type Authenticator struct {
	secretKey     []byte
	tokenDuration time.Duration
	Blacklist     *TokenBlacklist
	pool          DatabasePool
}

// NewAuthenticator erstellt eine neue JWT-Authenticator-Instanz mit dem angegebenen Secret und der Dauer.
func NewAuthenticator(secret string, pool DatabasePool, duration time.Duration) (*Authenticator, error) {
	if len(secret) < 32 {
		return nil, errors.New("JWT secret must be at least 32 bytes for security")
	}
	return &Authenticator{
		secretKey:     []byte(secret),
		tokenDuration: duration,
		Blacklist:     NewTokenBlacklist(pool),
		pool:          pool,
	}, nil
}

// GenerateToken generiert ein signiertes JWT, das Benutzeridentität und Rolle enthält.
func (a *Authenticator) GenerateToken(userID, barcodeID string, role Role) (string, error) {
	// jti macht jedes Token einzigartig. Ohne sie sind zwei Logins desselben
	// Kontos innerhalb derselben Sekunde byte-identisch (iat/exp haben
	// Sekunden-Granularität) — ein Logout des einen würde über die hash-basierte
	// Blacklist auch die Session des anderen widerrufen.
	jtiBytes := make([]byte, 16)
	if _, err := rand.Read(jtiBytes); err != nil {
		return "", fmt.Errorf("failed to generate token id: %w", err)
	}

	claims := Claims{
		UserID:    userID,
		BarcodeID: barcodeID,
		Rolle:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        hex.EncodeToString(jtiBytes),
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

// VerifyToken parst und validiert den bereitgestellten JWT-String und gibt dessen Claims zurück.
// Es prüft außerdem, ob das Token in der serverseitigen Blacklist widerrufen wurde.
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

	// Echtzeit-Sitzungswiderruf: Signatur und Blacklist allein reichen nicht. Wird ein
	// Mitarbeiter deaktiviert (aktiv=false) oder gelöscht, kennt die Blacklist sein Token
	// nicht (er hat sich nie ausgeloggt) — er behielte sonst bis zum natürlichen Ablauf des
	// (12h-)Tokens vollen Zugriff. Deshalb bei JEDER Verifikation den aktuellen DB-Status
	// prüfen. Eigener Timeout-Context wie in IsBlacklisted (unabhängig vom Request-Kontext).
	if err := a.pruefeKontoAktiv(claims.UserID); err != nil {
		return nil, err
	}

	return claims, nil
}

// pruefeKontoAktiv stellt sicher, dass das Konto zum Zeitpunkt des Requests noch existiert
// und aktiv ist. Fail-closed wie die Blacklist: bei DB-Fehler wird der Zugriff verweigert.
func (a *Authenticator) pruefeKontoAktiv(userID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	var aktiv bool
	if err := a.pool.QueryRow(ctx, `SELECT aktiv FROM benutzer WHERE id = $1`, userID).Scan(&aktiv); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return errors.New("user account no longer exists")
		}
		return errors.New("account status could not be verified")
	}
	if !aktiv {
		return errors.New("user account is deactivated")
	}
	return nil
}
