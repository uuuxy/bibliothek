package tuer

import (
	"context"
	"errors"
	"net/http"

	"bibliothek/apierrors"
	"bibliothek/auth"
)

// PruefeSitzung liest das Cookie der Sitzung und prüft das Token. Bei einem Fehler liefert sie
// dazu den Status, mit dem die Tür antwortet.
func (b *Basis) PruefeSitzung(r *http.Request) (*auth.Claims, int, error) {
	cookie, err := r.Cookie("session_token")
	if err != nil {
		if errors.Is(err, http.ErrNoCookie) {
			return nil, http.StatusUnauthorized, err
		}
		return nil, http.StatusBadRequest, err
	}
	// VerifyToken prüft neben Signatur und Sperrliste auch, ob das Konto noch aktiv und die
	// Sitzung nach Inaktivität nicht gesperrt ist, beides in der Datenbank; eine zweite Prüfung
	// braucht es hier nicht.
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

// sendeSitzungsfehler schickt den Fehler aus PruefeSitzung an den Client. Beim 503 (Aussetzer
// der Datenbank) sieht er nur den Satz von auth.ErrPruefungGestoert: Die Ursache, die
// VerifyToken mitliefert, gehört ins Log und nicht in die Meldung an der Theke
// (apierrors.AntworteFehlerMitMeldung).
func sendeSitzungsfehler(w http.ResponseWriter, status int, err error) {
	if errors.Is(err, auth.ErrPruefungGestoert) {
		apierrors.AntworteFehlerMitMeldung(w, status, auth.ErrPruefungGestoert.Error(), err)
		return
	}
	apierrors.AntworteFehler(w, status, err)
}

// erlaubeZugriff legt die Angaben der Sitzung in den Kontext der Anfrage und ruft next.
func erlaubeZugriff(w http.ResponseWriter, r *http.Request, next http.Handler, claims *auth.Claims) {
	ctx := context.WithValue(r.Context(), auth.ClaimsContextKey, claims)
	next.ServeHTTP(w, r.WithContext(ctx))
}

// VerlangeAnmeldung verlangt eine gültige Sitzung und kein bestimmtes Recht: für Endpunkte, die
// jede angemeldete Rolle öffnet. Welche das sind, hält api/nur_sitzung_liste_test.go fest;
// keiner davon darf Schülerdaten führen.
//
// Der erste Fall war der SSE-Stream /events, den die Oberfläche direkt nach der Anmeldung
// aufbaut. Hinge er an einem Fachrecht, bekäme eine Rolle ohne dieses Recht keine Absage,
// sondern eine Schleife neuer Verbindungen und nach 25 s ohne Herzschlag das Offline-Overlay
// aus App.svelte. Der Stream trägt nur Ereignisse der Theke (Art, Kennungen, Titel, Zeit) und
// keine Schülerdaten; deshalb genügt hier die Sitzung.
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
