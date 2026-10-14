package tuer

import (
	"context"
	"errors"
	"net/http"

	"bibliothek/apierrors"
	"bibliothek/auth"
)

// PruefeSitzung liest das Session-Cookie und verifiziert das Token. Bei Fehler
// werden der passende HTTP-Status und der Fehler zurückgegeben.
func (b *Basis) PruefeSitzung(r *http.Request) (*auth.Claims, int, error) {
	cookie, err := r.Cookie("session_token")
	if err != nil {
		if errors.Is(err, http.ErrNoCookie) {
			return nil, http.StatusUnauthorized, err
		}
		return nil, http.StatusBadRequest, err
	}
	// VerifyToken prüft neben Signatur und Blacklist auch den Echtzeit-Kontostatus
	// (aktiv / nicht gelöscht) und die Sperre nach Inaktivität direkt in der DB — siehe
	// auth.Authenticator.VerifyToken. Deshalb ist hier keine zusätzliche
	// Zombie-Session-Prüfung nötig.
	claims, err := b.Auth.VerifyToken(cookie.Value)
	if errors.Is(err, auth.ErrPruefungGestoert) {
		// Datenbank-Aussetzer: abgelehnt, aber nicht als abgelaufene Sitzung. Eine 401
		// meldet den Arbeitsplatz im Client ab (apiFetch → sitzungAbgelaufen).
		return nil, http.StatusServiceUnavailable, err
	}
	if errors.Is(err, auth.ErrSitzungGesperrt) {
		// Gesperrt nach Inaktivität: Die Anmeldung gilt weiter, antwortet aber nicht, bis
		// das Passwort eingegeben ist. 423 statt 401, damit der Client den Sperrbildschirm
		// zeigt und nicht abmeldet.
		return nil, http.StatusLocked, err
	}
	if err != nil {
		return nil, http.StatusUnauthorized, err
	}

	return claims, 0, nil
}

// sendeSitzungsfehler schickt den Fehler aus ClaimsAusRequest an den Client. Beim 503
// (Datenbank-Aussetzer) sieht der Client nur den Sentinel-Satz: Die Ursache, die
// VerifyToken mitwrappt, ist Betriebsinnenleben und gehört ins Log, nicht in die Meldung
// für die Theke (Register 12.09.2026, siehe apierrors.SendHTTPErrorMitMeldung).
func sendeSitzungsfehler(w http.ResponseWriter, status int, err error) {
	if errors.Is(err, auth.ErrPruefungGestoert) {
		apierrors.SendHTTPErrorMitMeldung(w, status, auth.ErrPruefungGestoert.Error(), err)
		return
	}
	apierrors.SendHTTPError(w, status, err)
}

// erlaubeZugriff injiziert die Claims in den Request-Kontext und ruft den nächsten
// Handler auf.
func erlaubeZugriff(w http.ResponseWriter, r *http.Request, next http.Handler, claims *auth.Claims) {
	ctx := context.WithValue(r.Context(), auth.ClaimsContextKey, claims)
	next.ServeHTTP(w, r.WithContext(ctx))
}

// VerlangeAnmeldung verlangt nur eine gültige Sitzung, kein bestimmtes Recht.
//
// Für Endpunkte, die JEDER angemeldete Client öffnet, unabhängig von seiner Rolle. Heute
// sind es vier (Stand 06.09.2026, im Rasterdurchgang nachgezählt — der Satz behauptete
// vorher „genau einer", und ein Kommentar, der eine Schutzbehauptung aufstellt, muss
// stimmen): der SSE-Stream /events, den der authStore direkt nach dem Login aufbaut,
// GET /api/einstellungen/sitzung, und die zwei lesenden Wege zum LMF-Plan
// (/api/lmf-termine und dessen PDF — veröffentlichter Plan, Klassennamen ohne
// Schülerbezug, PII-Matrix Stufe 0). Der Stream war der erste Fall und bleibt die
// Begründung:
// Ihn an ein Fachrecht zu hängen war eine Zeitbombe: Wer das Recht nicht hat, bekommt
// keine saubere Absage, sondern eine Reconnect-Schleife plus das Offline-Overlay aus
// App.svelte, das nach 25 s ohne Herzschlag zuschlägt. Für die Helfer-Rolle war das schon
// einmal der Grund, den Stream von view_students auf perform_actions umzuhängen (siehe
// routes_system.go) — mit der Portal-Rolle KOLLEGIUM wäre derselbe Fehler ein zweites Mal
// aufgetreten. Der Stream trägt nur operative Aktions-Events (Typ, IDs, Buchtitel,
// Zeitstempel) und keine Schüler-PII, deshalb ist die Sitzung hier die richtige Schwelle.
func (b *Basis) VerlangeAnmeldung() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, status, err := b.PruefeSitzung(r)
			if err != nil {
				sendeSitzungsfehler(w, status, err)
				return
			}
			erlaubeZugriff(w, r, next, claims)
		})
	}
}
