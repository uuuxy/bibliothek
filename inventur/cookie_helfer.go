package inventur

import (
	"net/http"
	"strings"
	"time"
)

// cookieDomainForRequest ermittelt die Cookie-Domain basierend auf dem Request-Host.
func (handler *APIHandler) cookieDomainForRequest(request *http.Request) string {
	host := strings.TrimSpace(strings.ToLower(request.Host))
	if idx := strings.Index(host, ":"); idx >= 0 {
		host = host[:idx]
	}
	domain := strings.TrimSpace(strings.ToLower(handler.cookieDomain))
	if domain == "" || host == "" {
		return ""
	}
	if host == domain || strings.HasSuffix(host, "."+domain) {
		return domain
	}
	return ""
}

// setAuthCookie setzt ein HttpOnly-Cookie für die Authentifizierung.
func (handler *APIHandler) setAuthCookie(writer http.ResponseWriter, request *http.Request, name string, token string, ttl time.Duration) {
	cookie := &http.Cookie{
		Name:     name,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   handler.cookieSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(ttl.Seconds()),
	}
	if domain := handler.cookieDomainForRequest(request); domain != "" {
		cookie.Domain = domain
	}
	http.SetCookie(writer, cookie)
}

// clearAuthCookie entfernt ein Auth-Cookie durch Setzen von MaxAge=-1.
func (handler *APIHandler) clearAuthCookie(writer http.ResponseWriter, request *http.Request, name string) {
	cookie := &http.Cookie{
		Name:     name,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   handler.cookieSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	}
	if domain := handler.cookieDomainForRequest(request); domain != "" {
		cookie.Domain = domain
	}
	http.SetCookie(writer, cookie)
}
