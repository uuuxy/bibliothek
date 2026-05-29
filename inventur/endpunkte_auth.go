package inventur

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"
)

type LoginRequestPayload struct {
	Password string `json:"password"`
}

// handleLogin verarbeitet Admin-Login-Anfragen mit Passwortverifikation.
func (handler *APIHandler) handleLogin(writer http.ResponseWriter, request *http.Request) {
	if request.Body == nil {
		writeError(writer, http.StatusBadRequest, "ungültiges JSON")
		return
	}
	var payload LoginRequestPayload
	if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
		writeError(writer, http.StatusBadRequest, "ungültiges JSON")
		return
	}

	adminPass := strings.TrimSpace(os.Getenv("ADMIN_PASSWORD"))
	if adminPass == "" {
		adminPass = "admin" // Fallback für lokale Entwicklung
	}

	if payload.Password != adminPass {
		writeError(writer, http.StatusUnauthorized, "falsches Passwort")
		return
	}

	tokenString, err := handler.issueToken(true)
	if err != nil {
		writeError(writer, http.StatusInternalServerError, "token fehler")
		return
	}
	csrfToken, err := generateCSRFToken()
	if err != nil {
		writeError(writer, http.StatusInternalServerError, "csrf token fehler")
		return
	}

	handler.setAuthCookie(writer, request, handler.adminCookie, tokenString, handler.adminTokenTTL)
	handler.clearAuthCookie(writer, request, handler.guestCookie)
	handler.setCSRFCookie(writer, request, csrfToken)
	writeJSON(writer, http.StatusOK, map[string]any{"ok": true, "role": "admin"})
}

// handleLoginGuest verarbeitet Gast-Login-Anfragen mit Passwortverifikation.
func (handler *APIHandler) handleLoginGuest(writer http.ResponseWriter, request *http.Request) {
	if request.Body == nil {
		writeError(writer, http.StatusBadRequest, "ungültiges JSON")
		return
	}
	var payload LoginRequestPayload
	if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
		writeError(writer, http.StatusBadRequest, "ungültiges JSON")
		return
	}

	guestPass := strings.TrimSpace(os.Getenv("GUEST_PASSWORD"))
	if guestPass == "" {
		guestPass = "guest" // Fallback für lokale Entwicklung
	}

	if payload.Password != guestPass {
		writeError(writer, http.StatusUnauthorized, "falsches Passwort")
		return
	}

	tokenString, err := handler.issueToken(false)
	if err != nil {
		writeError(writer, http.StatusInternalServerError, "token fehler")
		return
	}
	csrfToken, err := generateCSRFToken()
	if err != nil {
		writeError(writer, http.StatusInternalServerError, "csrf token fehler")
		return
	}

	handler.setAuthCookie(writer, request, handler.guestCookie, tokenString, handler.guestTokenTTL)
	handler.clearAuthCookie(writer, request, handler.adminCookie)
	handler.setCSRFCookie(writer, request, csrfToken)
	writeJSON(writer, http.StatusOK, map[string]any{"ok": true, "role": "guest"})
}

// handleLogout räumt alle Auth- und CSRF-Cookies auf.
func (handler *APIHandler) handleLogout(writer http.ResponseWriter, request *http.Request) {
	handler.clearAuthCookie(writer, request, handler.adminCookie)
	handler.clearAuthCookie(writer, request, handler.guestCookie)
	handler.clearCSRFCookie(writer, request)
	writeJSON(writer, http.StatusOK, map[string]any{"ok": true})
}

// handleAuthStatus gibt den aktuellen Authentifizierungsstatus zurück.
func (handler *APIHandler) handleAuthStatus(writer http.ResponseWriter, request *http.Request) {
	claims, err := handler.extractValidClaimsFromRequest(request)
	if err != nil {
		writeJSON(writer, http.StatusOK, map[string]any{
			"authenticated": false,
			"admin":         false,
		})
		return
	}

	// Generate and set CSRF cookie if missing, so mutation requests succeed
	if _, csrfErr := request.Cookie(handler.csrfCookie); csrfErr != nil {
		if csrfToken, tokenErr := generateCSRFToken(); tokenErr == nil {
			handler.setCSRFCookie(writer, request, csrfToken)
		}
	}

	writeJSON(writer, http.StatusOK, map[string]any{
		"authenticated": true,
		"admin":         claims.Admin,
	})
}
