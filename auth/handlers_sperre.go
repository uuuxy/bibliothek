package auth

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"bibliothek/apierrors"
	"bibliothek/db"
	"bibliothek/pkg/httpresp"
)

// SperrenHandler sperrt die Anmeldung, mit der die Anfrage kommt (POST /api/auth/sperren);
// danach beantwortet der Server ihre Anfragen mit 423, bis EntsperrenHandler aufschließt.
// Die Antwort sagt, ob gesperrt wurde: Ohne Zeile mit Prüfwert wird es nicht, denn gesperrt
// wird nur, was sich auch ohne Mailserver wieder aufschließen lässt.
func SperrenHandler(authenticator *Authenticator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := leseSitzung(w, r, authenticator)
		if !ok {
			return
		}
		gesperrt, err := authenticator.Sitzungen.Sperre(r.Context(), claims.SitzungID)
		if err != nil {
			apierrors.SendHTTPErrorMitMeldung(w, http.StatusServiceUnavailable, ErrPruefungGestoert.Error(), err)
			return
		}
		w.Header().Set(headerContentType, contentTypeJSON)
		httpresp.Encode(w, map[string]bool{"gesperrt": gesperrt})
	}
}

type entsperrenRequest struct {
	Password string `json:"password"`
}

// EntsperrenHandler schließt die gesperrte Anmeldung mit dem Passwort ihres Inhabers auf
// (POST /api/auth/entsperren) und antwortet wie die Anmeldung. Das Passwort prüft der
// Mailserver der Schule mit denselben Zählern wie beim Anmelden; nur wenn er nicht erreichbar
// ist, entscheidet der Prüfwert — sonst sperrte sein Ausfall die Theke zu.
func EntsperrenHandler(dbPool db.PgxPoolIface, authenticator *Authenticator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := leseSitzung(w, r, authenticator)
		if !ok {
			return
		}

		r.Body = http.MaxBytesReader(w, r.Body, loginRumpfMaxBytes)
		var req entsperrenRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			apierrors.SendHTTPError(w, http.StatusBadRequest, err)
			return
		}
		if req.Password == "" {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("password is required"))
			return
		}

		antwort, ok := ladeKontoAntwort(w, r, dbPool, claims.UserID)
		if !ok {
			return
		}

		bruteForceKey, ok := checkBruteForceLimit(w, antwort.Email, realIP(r))
		if !ok {
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), loginHandlerFrist)
		defer cancel()

		if !pruefeEntsperrPasswort(ctx, w, authenticator, claims.SitzungID, antwort.Email, req.Password, bruteForceKey) {
			return
		}

		if err := authenticator.Sitzungen.Entsperre(ctx, claims.SitzungID); err != nil {
			apierrors.SendHTTPErrorMitMeldung(w, http.StatusServiceUnavailable, ErrPruefungGestoert.Error(), err)
			return
		}

		w.Header().Set(headerContentType, contentTypeJSON)
		httpresp.Encode(w, antwort)
	}
}

// pruefeEntsperrPasswort entscheidet, ob das Passwort aufschließt; bei false ist die
// Fehlerantwort schon geschrieben. Ein Fehlversuch zählt nur beim falschen Passwort, nicht
// beim Ausfall des Mailservers (wie in authenticateUser).
func pruefeEntsperrPasswort(ctx context.Context, w http.ResponseWriter, authenticator *Authenticator, sitzungID, email, passwort, bruteForceKey string) bool {
	imapErr := AuthenticateIMAP(ctx, email, passwort)
	if imapErr == nil {
		merkeGeaendertesPasswort(ctx, authenticator, sitzungID, passwort)
		return true
	}
	if !errors.Is(imapErr, ErrMailserverNichtErreichbar) {
		globalLoginLimiter.recordFailure(bruteForceKey)
		apierrors.SendHTTPError(w, http.StatusUnauthorized, errors.New("invalid email or password"))
		return false
	}

	passt, vorhanden, err := authenticator.Sitzungen.PasswortPasst(ctx, sitzungID, passwort)
	if err != nil {
		apierrors.SendHTTPErrorMitMeldung(w, http.StatusServiceUnavailable, ErrPruefungGestoert.Error(), err)
		return false
	}
	if !vorhanden {
		apierrors.SendHTTPError(w, http.StatusServiceUnavailable, ErrMailserverNichtErreichbar)
		return false
	}
	if !passt {
		globalLoginLimiter.recordFailure(bruteForceKey)
		apierrors.SendHTTPError(w, http.StatusUnauthorized, errors.New("invalid email or password"))
		return false
	}
	slog.Warn("Sperre ohne Mailserver aufgeschlossen: Passwort gegen den Prüfwert der Anmeldung geprüft")
	return true
}

// merkeGeaendertesPasswort zieht den Prüfwert nach, wenn der Mailserver ein Passwort
// angenommen hat, das nicht zu ihm passt. Scheitert das, bleibt der alte Prüfwert stehen;
// aufgeschlossen ist trotzdem, der Mailserver hat entschieden.
func merkeGeaendertesPasswort(ctx context.Context, authenticator *Authenticator, sitzungID, passwort string) {
	passt, vorhanden, err := authenticator.Sitzungen.PasswortPasst(ctx, sitzungID, passwort)
	if err != nil {
		slog.Warn("Entsperren: Prüfwert der Anmeldung ließ sich nicht lesen", "fehler", err)
		return
	}
	if !vorhanden || passt {
		return
	}
	if err := authenticator.Sitzungen.MerkePasswort(ctx, sitzungID, passwort); err != nil {
		slog.Warn("Entsperren: Prüfwert der Anmeldung ließ sich nicht nachziehen", "fehler", err)
	}
}
