package api

// Der Schutz gegen CSRF nach dem Muster Double-Submit-Cookie:
//
//  1. Jede Antwort setzt ein Cookie „csrf_token" ohne HttpOnly mit einem zufälligen Token;
//     die Oberfläche liest es.
//  2. Bei ändernden Anfragen (POST, PUT, PATCH, DELETE) vergleicht die Middleware das Cookie
//     mit dem Kopf X-CSRF-Token, den die Oberfläche mitschickt.
//  3. Fehlt eines oder weichen sie ab, ist die Antwort 403.
//
// Das ergänzt SameSite=Strict an den Cookies als zweite Schranke. Ausgenommen sind nur die
// zwei Pfade in istPruefungsAusnahme; jede weitere Ausnahme nähme genau diese zweite Schranke
// wieder heraus.

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"net/http"
	"strings"

	"bibliothek/api/tuer"
	"bibliothek/apierrors"
)

const csrfCookieName = "csrf_token"

// erzeugeCSRFToken erzeugt ein zufälliges Token aus 32 Byte (crypto/rand).
func erzeugeCSRFToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// newCSRFCookie baut das (nicht-HttpOnly) Double-Submit-Cookie. Das Secure-Flag wird
// übergeben, damit es konfigurierbar ist.
func newCSRFCookie(token string, nurHTTPS bool) *http.Cookie {
	// #nosec G124 - Secure folgt nurHTTPS
	return &http.Cookie{
		Name:     csrfCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: false, // Die Oberfläche muss es lesen können.
		Secure:   nurHTTPS,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   86400, // 24 Stunden
	}
}

// csrfToken liefert das Token (GET /api/csrf-token) und setzt das Cookie, falls es fehlt; ein
// zweiter Aufruf liefert dasselbe Token. Ein Aufrufer ohne Browser bekommt so verlässlich ein
// Token, ohne vorher an einer ändernden Anfrage ohne Cookie mit 403 zu scheitern. Browser
// bekommen das Cookie schon bei jedem GET über csrfMiddleware.
func (s *Server) csrfToken() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := ""
		if cookie, err := r.Cookie(csrfCookieName); err == nil {
			token = strings.TrimSpace(cookie.Value)
		}
		if token == "" {
			erzeugt, err := erzeugeCSRFToken()
			if err != nil {
				apierrors.AntworteFehler(w, http.StatusInternalServerError, errors.New("CSRF-Token konnte nicht erzeugt werden"))
				return
			}
			token = erzeugt
			http.SetCookie(w, newCSRFCookie(token, s.nurHTTPS))
		}
		tuer.AntworteJSON(w, http.StatusOK, map[string]string{"csrf_token": token})
	}
}

// istAPIPfad entscheidet, ob ein Pfad der Prüfung unterliegt. Alles andere ist statische
// Auslieferung (gebaute Oberfläche, Uploads, Swagger) und ändert nichts.
//
// Die Prüfung gilt für alle Pfade unter /api/, auch für die des Inventur-Moduls und der
// Verwaltung: PUT /api/admin/settings/mail genügte, um den SMTP-Host samt Zugangsdaten auf
// einen fremden Server umzubiegen.
//
// /login gehört dazu, obwohl es nicht unter /api/ liegt: Ohne die Prüfung könnte eine fremde
// Seite ein Formular mit enctype="text/plain" abschicken, dessen Rumpf als JSON durchgeht (kein
// Preflight, keine CORS-Hürde), und das Opfer in die Sitzung eines Kontos des Angreifers
// anmelden; alles, was es danach an der Theke scannt, landete in dessen Konto. SameSite=Strict
// verhindert das nicht, es regelt das Senden von Cookies, nicht das Setzen durch die Antwort.
// /login/ gilt als Anfang, nicht nur der genaue Pfad: Ein künftiges /login/code oder
// /login/barcode fiele sonst lautlos aus der Prüfung, weil an keiner Stelle eine Ausnahme
// dafür eingetragen werden müsste.
func istAPIPfad(pfad string) bool {
	return strings.HasPrefix(pfad, "/api/") || pfad == "/login" || strings.HasPrefix(pfad, "/login/")
}

// istPruefungsAusnahme nennt die ändernden Pfade, die ohne Token durchmüssen, jeder mit
// seinem Grund, damit die Liste nicht unbemerkt wächst:
//
//   - /api/auth/logout: Der Aufruf löscht nur das eigene Cookie; ein erzwungenes Abmelden ist
//     die Obergrenze des Schadens, und ein abgelaufenes Token darf sich nicht am fehlenden
//     CSRF-Cookie festhaken.
//   - /api/auth/refresh: Läuft in der Oberfläche absichtlich ohne apiFetch (authStore), also
//     ohne den Kopf, und erneuert nur eine Sitzung, die der Browser schon hat.
func istPruefungsAusnahme(pfad string) bool {
	return pfad == "/api/auth/logout" ||
		pfad == "/api/auth/refresh"
}

// erneuereCSRFCookie setzt das Cookie csrf_token, falls noch keines (oder ein leeres) da
// ist, damit die Oberfläche es lesen kann. Die Tür csrfToken verwaltet ihr Cookie selbst und
// wird hier übersprungen, um doppelte Köpfe Set-Cookie zu vermeiden.
func erneuereCSRFCookie(w http.ResponseWriter, r *http.Request, istAPI bool, pfad string, nurHTTPS bool) {
	if !istAPI || pfad == "/api/csrf-token" {
		return
	}

	vorhanden := ""
	if cookie, err := r.Cookie(csrfCookieName); err == nil {
		vorhanden = cookie.Value
	}
	// Ein neues Token nur, wenn noch keines da ist.
	if vorhanden != "" {
		return
	}
	token, err := erzeugeCSRFToken()
	if err == nil {
		http.SetCookie(w, newCSRFCookie(token, nurHTTPS))
	}
}

// meldungCSRF antwortet auf eine ändernde Anfrage ohne passendes Token; neu laden holt ein
// frisches Cookie. Welches Stück fehlte, steht im Log.
const meldungCSRF = "Die Sicherheitsprüfung der Anfrage ist fehlgeschlagen. Bitte die Seite neu laden."

// pruefeCSRFToken vergleicht das csrf_token-Cookie mit dem X-CSRF-Token-
// Header in konstanter Laufzeit und liefert bei Fehlschlag einen sprechenden Fehler.
func pruefeCSRFToken(r *http.Request) error {
	cookie, err := r.Cookie(csrfCookieName)
	if err != nil {
		return errors.New("CSRF-Validierung fehlgeschlagen: Cookie fehlt")
	}
	ausCookie := strings.TrimSpace(cookie.Value)
	ausKopf := strings.TrimSpace(r.Header.Get("X-CSRF-Token"))

	if ausCookie == "" || ausKopf == "" {
		return errors.New("CSRF-Validierung fehlgeschlagen: Token fehlt")
	}
	if subtle.ConstantTimeCompare([]byte(ausCookie), []byte(ausKopf)) != 1 {
		return errors.New("CSRF-Validierung fehlgeschlagen: Token stimmt nicht überein")
	}
	return nil
}

// csrfMiddleware verlangt bei jeder ändernden Anfrage an die Schnittstelle das Token nach dem
// Muster Double-Submit-Cookie.
//
// Ausgenommen bleiben nur die Pfade in istPruefungsAusnahme und alles, was keine Schnittstelle
// ist (statische Auslieferung); die Begründung steht dort an jeder Zeile.
func (s *Server) csrfMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pfad := r.URL.Path

		istAPI := istAPIPfad(pfad)

		// Das Cookie immer setzen oder erneuern, damit die Oberfläche es lesen kann.
		erneuereCSRFCookie(w, r, istAPI, pfad, s.nurHTTPS)

		// Geprüft werden nur ändernde Anfragen an die Schnittstelle.
		aendert := r.Method == http.MethodPost ||
			r.Method == http.MethodPut ||
			r.Method == http.MethodPatch ||
			r.Method == http.MethodDelete

		ausgenommen := !istAPI || istPruefungsAusnahme(pfad)

		if aendert && !ausgenommen {
			if err := pruefeCSRFToken(r); err != nil {
				apierrors.AntworteFehlerMitMeldung(w, http.StatusForbidden, meldungCSRF, err)
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}
