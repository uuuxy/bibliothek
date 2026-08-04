package api

// csrf.go — Global CSRF protection middleware using the Double-Submit Cookie pattern.
//
// How it works:
//   1. Every response sets a non-HttpOnly cookie "csrf_token" containing a
//      cryptographically random token. The frontend JS reads this cookie.
//   2. On mutating requests (POST/PUT/PATCH/DELETE), the middleware compares
//      the cookie value against the X-CSRF-Token header sent by the frontend.
//   3. If they don't match or are missing, the request is rejected with 403.
//
// This complements SameSite=Strict cookies as a defense-in-depth measure.
// Ausgenommen sind nur die drei Pfade in istPruefungsAusnahme; jede weitere Ausnahme
// nimmt genau diese zweite Schranke wieder heraus.

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"net/http"
	"strings"

	"bibliothek/apierrors"
)

const csrfCookieName = "csrf_token"

// generateGlobalCSRFToken creates a 32-byte cryptographically random token.
func generateGlobalCSRFToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// newCSRFCookie baut das (nicht-HttpOnly) Double-Submit-Cookie. Das Secure-Flag wird
// übergeben, damit es konfigurierbar ist.
func newCSRFCookie(token string, cookieSecure bool) *http.Cookie {
	// #nosec G124 - Secure flag is dynamically configured
	return &http.Cookie{
		Name:     csrfCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: false, // Must be readable by frontend JS
		Secure:   cookieSecure,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   86400, // 24 hours
	}
}

// CSRFTokenHandler is an idempotent bootstrap endpoint (GET /api/csrf-token) that
// guarantees a csrf_token cookie is set and returns the token in the body. It lets
// non-browser API clients obtain a token deterministically — without first triggering
// a 403 on a mutating request that has no prior cookie. Browsers get the cookie via the
// CSRFMiddleware on any GET, but a direct POST without a preceding GET would otherwise fail.
func (s *Server) CSRFTokenHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := ""
		if cookie, err := r.Cookie(csrfCookieName); err == nil {
			token = strings.TrimSpace(cookie.Value)
		}
		if token == "" {
			generated, err := generateGlobalCSRFToken()
			if err != nil {
				apierrors.SendHTTPError(w, http.StatusInternalServerError, errors.New("CSRF-Token konnte nicht erzeugt werden"))
				return
			}
			token = generated
			http.SetCookie(w, newCSRFCookie(token, s.CookieSecure))
		}
		RespondJSON(w, http.StatusOK, map[string]string{"csrf_token": token})
	}
}

// istAPIPfad entscheidet, ob ein Pfad der globalen CSRF-Prüfung unterliegt. Alles
// andere ist statische Auslieferung (gebautes Frontend, Uploads, Swagger) und kennt
// keine Mutation.
//
// Hier stand bis zum 30.07.2026 eine zweite Liste, die /api/admin, /api/books,
// /api/class-books, /api/subjects, /api/lookup/ und /api/auth/status von der Prüfung
// ausnahm — begründet damit, das Inventur-Modul bringe ein eigenes CSRF-System mit.
// Dieses System existierte im Code nicht; der Kommentar war sein einziger Beleg. Die
// echten Inventur-Routen liegen unter /api/inventur/* und liefen immer durch die
// globale Prüfung. Tatsächlich ausgenommen waren sechs Admin-Mutationen, darunter
// PUT /api/admin/permissions und PUT /api/admin/settings/mail — letzteres genügt, um
// den SMTP-Host mitsamt gespeicherter Zugangsdaten auf einen fremden Server
// umzubiegen. Getragen hat das allein SameSite=Strict am Sitzungscookie: Die zweite
// Schranke, die diese Datei verspricht, fehlte für die größte Angriffsfläche.
// /login gehört seit dem 04.08.2026 dazu. Es liegt als einziger mutierender Endpunkt
// nicht unter /api/ und fiel deshalb still aus der Prüfung — klassisches Login-CSRF:
// Eine fremde Seite kann ein Formular mit enctype="text/plain" abschicken, dessen Body
// als JSON durchgeht (kein Preflight, keine CORS-Hürde), und das Opfer damit in die
// Sitzung eines ANGREIFER-Kontos anmelden. Alles, was es danach am Tresen scannt,
// landet in dessen Konto. SameSite=Strict verhindert das nicht: Es regelt das SENDEN
// von Cookies, nicht das Setzen durch die Antwort.
func istAPIPfad(path string) bool {
	return strings.HasPrefix(path, "/api/") || path == "/login"
}

// istPruefungsAusnahme nennt die mutierenden Pfade, die ohne Token durchmüssen —
// jeder mit seinem Grund, damit die Liste nicht wieder unbemerkt wächst:
//
//   - /api/auth/logout: Der Aufruf löscht nur das eigene Cookie; ein erzwungenes
//     Abmelden ist die Obergrenze des Schadens, und ein abgelaufenes Token darf sich
//     nicht am fehlenden CSRF-Cookie festhaken.
//   - /api/auth/refresh: Läuft im Frontend absichtlich ohne apiFetch (authStore), also
//     ohne Header — und erneuert nur eine Sitzung, die der Browser schon hat.
//
// Hier stand außerdem /login/barcode mit der Begründung "vor dem Login gibt es keine
// Sitzung". Diese Route existiert im gesamten Repository nicht: kein mux.Handle, kein
// Aufruf im Frontend, und validateLoginCredentials verlangt zwingend E-Mail UND
// Passwort — die Felder barcode_id/pin in LoginRequest wertet niemand aus. Eine
// Ausnahme für einen Pfad, den es nicht gibt, schützt nichts und würde scharf, sobald
// jemand die Route anlegt. Entfernt.
func istPruefungsAusnahme(path string) bool {
	return path == "/api/auth/logout" ||
		path == "/api/auth/refresh"
}

// refreshCSRFCookie setzt das csrf_token-Cookie, falls noch keines (bzw. ein leeres)
// existiert, damit das Frontend es auslesen kann. Der Bootstrap-Endpunkt verwaltet
// sein Cookie selbst und wird hier übersprungen, um doppelte Set-Cookie-Header zu
// vermeiden.
func refreshCSRFCookie(w http.ResponseWriter, r *http.Request, isAPIPath bool, path string, cookieSecure bool) {
	if !isAPIPath || path == "/api/csrf-token" {
		return
	}

	existingToken := ""
	if cookie, err := r.Cookie(csrfCookieName); err == nil {
		existingToken = cookie.Value
	}
	// Generate a new token only if one doesn't exist yet
	if existingToken != "" {
		return
	}
	token, err := generateGlobalCSRFToken()
	if err == nil {
		http.SetCookie(w, newCSRFCookie(token, cookieSecure))
	}
}

// validateCSRFDoubleSubmit vergleicht das csrf_token-Cookie mit dem X-CSRF-Token-
// Header in konstanter Laufzeit und liefert bei Fehlschlag einen sprechenden Fehler.
func validateCSRFDoubleSubmit(r *http.Request) error {
	csrfCookie, err := r.Cookie(csrfCookieName)
	if err != nil {
		return errors.New("CSRF-Validierung fehlgeschlagen: Cookie fehlt")
	}
	cookieVal := strings.TrimSpace(csrfCookie.Value)
	headerVal := strings.TrimSpace(r.Header.Get("X-CSRF-Token"))

	if cookieVal == "" || headerVal == "" {
		return errors.New("CSRF-Validierung fehlgeschlagen: Token fehlt")
	}
	if subtle.ConstantTimeCompare([]byte(cookieVal), []byte(headerVal)) != 1 {
		return errors.New("CSRF-Validierung fehlgeschlagen: Token stimmt nicht überein")
	}
	return nil
}

// CSRFMiddleware returns an HTTP middleware that enforces the Double-Submit
// Cookie CSRF pattern on all mutating API requests.
//
// Ausgenommen bleiben ausschließlich die in istPruefungsAusnahme genannten Pfade und
// alles, was keine API ist (statische Auslieferung) — die Begründung steht dort
// jeweils an der Zeile.
func (s *Server) CSRFMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		isAPIPath := istAPIPfad(path)

		// Always set/refresh the CSRF cookie so the frontend can read it.
		refreshCSRFCookie(w, r, isAPIPath, path, s.CookieSecure)

		// Only validate on mutating methods for API paths
		isMutation := r.Method == http.MethodPost ||
			r.Method == http.MethodPut ||
			r.Method == http.MethodPatch ||
			r.Method == http.MethodDelete

		isExempt := !isAPIPath || istPruefungsAusnahme(path)

		if isMutation && !isExempt {
			if err := validateCSRFDoubleSubmit(r); err != nil {
				apierrors.SendHTTPError(w, http.StatusForbidden, err)
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}
