package inventur

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	Admin bool `json:"admin"`
	V     int  `json:"v"`
	jwt.RegisteredClaims
}

func parseDurationEnv(key string, fallback time.Duration) time.Duration {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(raw)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

func parseIntEnv(key string, fallback int) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}

func parseBoolEnv(key string, fallback bool) bool {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	value, err := strconv.ParseBool(raw)
	if err != nil {
		return fallback
	}
	return value
}

func extractTokenFromRequest(request *http.Request, cookieNames ...string) (string, error) {
	for _, cookieName := range cookieNames {
		if cookieName == "" {
			continue
		}
		cookie, err := request.Cookie(cookieName)
		if err == nil {
			token := strings.TrimSpace(cookie.Value)
			if token != "" {
				return token, nil
			}
		}
	}

	return "", errors.New("missing auth token")
}

func (handler *APIHandler) extractValidClaimsFromRequest(request *http.Request) (*Claims, error) {
	// First check the inventur-specific cookies
	hadToken := false
	for _, cookie := range request.Cookies() {
		if cookie.Name == handler.adminCookie || cookie.Name == handler.guestCookie {
			token := strings.TrimSpace(cookie.Value)
			if token == "" {
				continue
			}
			hadToken = true
			claims, err := handler.parseAndValidateClaims(token)
			if err == nil {
				return claims, nil
			}
		}
	}

	// Fallback to checking the main app's session_token
	sessionCookie, err := request.Cookie("session_token")
	if err == nil {
		sessionToken := strings.TrimSpace(sessionCookie.Value)
		if sessionToken != "" {
			token, err := jwt.Parse(sessionToken, func(t *jwt.Token) (interface{}, error) {
				if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
				}
				return handler.jwtKey, nil
			})
			if err == nil && token.Valid {
				if mapClaims, ok := token.Claims.(jwt.MapClaims); ok {
					role, _ := mapClaims["rolle"].(string)
					switch role {
					case "admin", "mitarbeiter":
						return &Claims{
							Admin: true,
							V:     handler.tokenVersion,
							RegisteredClaims: jwt.RegisteredClaims{
								Subject: "admin",
							},
						}, nil
					case "lehrer":
						return &Claims{
							Admin: false,
							V:     handler.tokenVersion,
							RegisteredClaims: jwt.RegisteredClaims{
								Subject: "guest",
							},
						}, nil
					}
				}
			}
		}
	}

	if !hadToken {
		return nil, errors.New("missing auth token")
	}
	return nil, errors.New("invalid auth token")
}

func (handler *APIHandler) issueToken(isAdmin bool) (string, error) {
	now := time.Now()
	ttl := handler.guestTokenTTL
	subject := "guest"
	if isAdmin {
		ttl = handler.adminTokenTTL
		subject = "admin"
	}

	claims := &Claims{
		Admin: isAdmin,
		V:     handler.tokenVersion,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    handler.jwtIssuer,
			Subject:   subject,
			Audience:  jwt.ClaimStrings{handler.jwtAudience},
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
			ID:        subject + ":" + strconv.FormatInt(now.UnixNano(), 10),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(handler.jwtKey)
}

func (handler *APIHandler) parseAndValidateClaims(tokenStr string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(
		tokenStr,
		claims,
		func(token *jwt.Token) (interface{}, error) {
			if token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
				return nil, http.ErrAbortHandler
			}
			return handler.jwtKey, nil
		},
		jwt.WithIssuer(handler.jwtIssuer),
		jwt.WithAudience(handler.jwtAudience),
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithLeeway(30*time.Second),
	)
	if err != nil || !token.Valid {
		return nil, errors.New("invalid token")
	}

	if claims.V != handler.tokenVersion {
		return nil, errors.New("token version mismatch")
	}
	if claims.Subject != "admin" && claims.Subject != "guest" {
		return nil, fmt.Errorf("invalid subject: %s", claims.Subject)
	}
	if claims.Admin && claims.Subject != "admin" {
		return nil, errors.New("admin claim mismatch")
	}
	if !claims.Admin && claims.Subject != "guest" {
		return nil, errors.New("guest claim mismatch")
	}

	return claims, nil
}
