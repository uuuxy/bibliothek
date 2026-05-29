package inventur

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"net/http"
	"strings"
)

func isMutationMethod(method string) bool {
	switch method {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	default:
		return false
	}
}

func generateCSRFToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

func (handler *APIHandler) setCSRFCookie(writer http.ResponseWriter, request *http.Request, token string) {
	cookie := &http.Cookie{
		Name:     handler.csrfCookie,
		Value:    token,
		Path:     "/",
		HttpOnly: false,
		Secure:   handler.cookieSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(handler.adminTokenTTL.Seconds()),
	}
	if domain := handler.cookieDomainForRequest(request); domain != "" {
		cookie.Domain = domain
	}
	http.SetCookie(writer, cookie)
}

func (handler *APIHandler) clearCSRFCookie(writer http.ResponseWriter, request *http.Request) {
	cookie := &http.Cookie{
		Name:     handler.csrfCookie,
		Value:    "",
		Path:     "/",
		HttpOnly: false,
		Secure:   handler.cookieSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	}
	if domain := handler.cookieDomainForRequest(request); domain != "" {
		cookie.Domain = domain
	}
	http.SetCookie(writer, cookie)
}

func (handler *APIHandler) validateCSRF(request *http.Request) error {
	if !isMutationMethod(request.Method) {
		return nil
	}
	csrfCookie, err := request.Cookie(handler.csrfCookie)
	if err != nil {
		return errors.New("missing csrf cookie")
	}
	cookieToken := strings.TrimSpace(csrfCookie.Value)
	headerToken := strings.TrimSpace(request.Header.Get(handler.csrfHeader))
	if cookieToken == "" || headerToken == "" {
		return errors.New("missing csrf token")
	}
	if subtle.ConstantTimeCompare([]byte(cookieToken), []byte(headerToken)) != 1 {
		return errors.New("csrf token mismatch")
	}
	return nil
}
