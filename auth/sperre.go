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

// Sperren sperrt die Anmeldung, mit der die Anfrage kommt (POST /api/auth/sperren); danach
// beantwortet der Server ihre Anfragen mit 423, bis Entsperren aufschließt. Die Antwort sagt, ob
// gesperrt wurde: Ohne Zeile mit Prüfwert wird es nicht, denn gesperrt wird nur, was sich auch
// ohne Mailserver wieder aufschließen lässt.
func Sperren(anmeldung *Anmeldung) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := leseSitzung(w, r, anmeldung)
		if !ok {
			return
		}
		gesperrt, err := anmeldung.Sitzungen.sperre(r.Context(), claims.SitzungID)
		if err != nil {
			apierrors.AntworteFehlerMitMeldung(w, http.StatusServiceUnavailable, ErrPruefungGestoert.Error(), err)
			return
		}
		w.Header().Set(headerContentType, contentTypeJSON)
		httpresp.SchreibeJSON(w, map[string]bool{"gesperrt": gesperrt})
	}
}

type entsperrenAnfrage struct {
	Passwort string `json:"password"`
}

// Entsperren schließt die gesperrte Anmeldung mit dem Passwort ihres Inhabers auf
// (POST /api/auth/entsperren) und antwortet wie die Anmeldung. Das Passwort prüft der
// Mailserver der Schule mit denselben Zählern wie beim Anmelden; nur wenn er nicht erreichbar
// ist, entscheidet der Prüfwert, sonst sperrte sein Ausfall die Theke zu.
func Entsperren(pool db.Pool, anmeldung *Anmeldung) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := leseSitzung(w, r, anmeldung)
		if !ok {
			return
		}

		r.Body = http.MaxBytesReader(w, r.Body, anmeldeRumpfMaxBytes)
		var anfrage entsperrenAnfrage
		if err := json.NewDecoder(r.Body).Decode(&anfrage); err != nil {
			apierrors.AntworteFehler(w, http.StatusBadRequest, err)
			return
		}
		if anfrage.Passwort == "" {
			apierrors.AntworteFehler(w, http.StatusBadRequest, errors.New("password is required"))
			return
		}

		antwort, ok := ladeKontoAntwort(w, r, pool, claims.KontoID)
		if !ok {
			return
		}

		schluessel, ok := pruefeFehlversuche(w, antwort.Email, adresseDerAnfrage(r))
		if !ok {
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), anmeldeFrist)
		defer cancel()

		if !pruefeEntsperrPasswort(ctx, w, anmeldung, claims.SitzungID, antwort.Email, anfrage.Passwort, schluessel) {
			return
		}

		if err := anmeldung.Sitzungen.entsperre(ctx, claims.SitzungID); err != nil {
			apierrors.AntworteFehlerMitMeldung(w, http.StatusServiceUnavailable, ErrPruefungGestoert.Error(), err)
			return
		}

		w.Header().Set(headerContentType, contentTypeJSON)
		httpresp.SchreibeJSON(w, antwort)
	}
}

// pruefeEntsperrPasswort entscheidet, ob das Passwort aufschließt; bei false ist die
// Fehlerantwort schon geschrieben. Ein Fehlversuch zählt nur beim falschen Passwort, nicht beim
// Ausfall des Mailservers (wie in meldeAn).
func pruefeEntsperrPasswort(ctx context.Context, w http.ResponseWriter, anmeldung *Anmeldung, sitzungID, email, passwort, schluessel string) bool {
	imapFehler := pruefeIMAPAnmeldung(ctx, email, passwort)
	if imapFehler == nil {
		merkeGeaendertesPasswort(ctx, anmeldung, sitzungID, passwort)
		return true
	}
	if !errors.Is(imapFehler, errMailserverNichtErreichbar) {
		anmeldeBremse.merkeFehlversuch(schluessel)
		apierrors.AntworteFehler(w, http.StatusUnauthorized, errors.New("invalid email or password"))
		return false
	}

	passt, vorhanden, err := anmeldung.Sitzungen.passwortPasst(ctx, sitzungID, passwort)
	if err != nil {
		apierrors.AntworteFehlerMitMeldung(w, http.StatusServiceUnavailable, ErrPruefungGestoert.Error(), err)
		return false
	}
	if !vorhanden {
		apierrors.AntworteFehler(w, http.StatusServiceUnavailable, errMailserverNichtErreichbar)
		return false
	}
	if !passt {
		anmeldeBremse.merkeFehlversuch(schluessel)
		apierrors.AntworteFehler(w, http.StatusUnauthorized, errors.New("invalid email or password"))
		return false
	}
	slog.Warn("Sperre ohne Mailserver aufgeschlossen: Passwort gegen den Prüfwert der Anmeldung geprüft")
	return true
}

// merkeGeaendertesPasswort zieht den Prüfwert nach, wenn der Mailserver ein Passwort angenommen
// hat, das nicht zu ihm passt. Scheitert das, bleibt der alte Prüfwert stehen; aufgeschlossen
// ist trotzdem, der Mailserver hat entschieden.
func merkeGeaendertesPasswort(ctx context.Context, anmeldung *Anmeldung, sitzungID, passwort string) {
	passt, vorhanden, err := anmeldung.Sitzungen.passwortPasst(ctx, sitzungID, passwort)
	if err != nil {
		slog.Warn("Entsperren: Prüfwert der Anmeldung ließ sich nicht lesen", "fehler", err)
		return
	}
	if !vorhanden || passt {
		return
	}
	if err := anmeldung.Sitzungen.merkePasswort(ctx, sitzungID, passwort); err != nil {
		slog.Warn("Entsperren: Prüfwert der Anmeldung ließ sich nicht nachziehen", "fehler", err)
	}
}
