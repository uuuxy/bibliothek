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
	"bibliothek/repository"
)

// Sperren sperrt die Anmeldung, mit der die Anfrage kommt (POST /api/auth/sperren); danach
// beantwortet der Server ihre Anfragen mit 423, bis Entsperren aufschließt. Die Antwort sagt, ob
// gesperrt wurde: Ohne Zeile mit Prüfwert wird es nicht, denn gesperrt wird nur, was sich auch
// ohne Mailserver oder Verzeichnisdienst wieder aufschließen lässt.
func Sperren(anmeldung *Anmeldung) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := leseSitzung(w, r, anmeldung)
		if !ok {
			return
		}
		gesperrt, err := anmeldung.Sitzungen.sperre(r.Context(), claims.SitzungID)
		if err != nil {
			apierrors.AntworteFehlerMitMeldung(w, http.StatusServiceUnavailable, MeldungPruefungGestoert, err)
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
// (POST /api/auth/entsperren) und antwortet wie die Anmeldung. Das Passwort prüft der Server des
// Kontos (Mailserver oder Verzeichnisdienst) mit denselben Zählern wie beim Anmelden; nur wenn er
// nicht erreichbar ist, entscheidet der Prüfwert, sonst sperrte sein Ausfall die Theke zu.
func Entsperren(pool db.Pool, anmeldung *Anmeldung) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := leseSitzung(w, r, anmeldung)
		if !ok {
			return
		}

		r.Body = http.MaxBytesReader(w, r.Body, anmeldeRumpfMaxBytes)
		var anfrage entsperrenAnfrage
		if err := json.NewDecoder(r.Body).Decode(&anfrage); err != nil {
			apierrors.AntworteFehlerMitMeldung(w, http.StatusBadRequest, apierrors.MeldungAnfrageUnlesbar, err)
			return
		}
		if anfrage.Passwort == "" {
			apierrors.AntworteFehlerMitMeldung(w, http.StatusBadRequest, meldungPasswortFehlt, nil)
			return
		}

		antwort, ok := ladeKontoAntwort(w, r, pool, claims.KontoID)
		if !ok {
			return
		}

		schluessel, ok := anmeldung.pruefeFehlversuche(w, antwort.Email, adresseDerAnfrage(r))
		if !ok {
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), anmeldeFrist)
		defer cancel()

		pruefung := entsperrPruefung{
			kontoID: claims.KontoID, sitzungID: claims.SitzungID, email: antwort.Email,
			passwort: anfrage.Passwort, schluessel: schluessel,
		}
		if !pruefeEntsperrPasswort(ctx, w, pool, anmeldung, pruefung) {
			return
		}

		if err := anmeldung.Sitzungen.entsperre(ctx, claims.SitzungID); err != nil {
			apierrors.AntworteFehlerMitMeldung(w, http.StatusServiceUnavailable, MeldungPruefungGestoert, err)
			return
		}

		w.Header().Set(headerContentType, contentTypeJSON)
		httpresp.SchreibeJSON(w, antwort)
	}
}

// entsperrPruefung ist, was das Aufschließen über Anmeldung und Konto weiß.
type entsperrPruefung struct {
	kontoID, sitzungID, email, passwort, schluessel string
}

// pruefeEntsperrPasswort entscheidet, ob das Passwort aufschließt; bei false ist die
// Fehlerantwort schon geschrieben. Gefragt wird der Server, den das Konto nennt. Ein Fehlversuch
// zählt nur beim falschen Passwort, nicht beim Ausfall des Servers (wie in meldeAn).
func pruefeEntsperrPasswort(ctx context.Context, w http.ResponseWriter, pool db.Pool, anmeldung *Anmeldung, p entsperrPruefung) bool {
	verfahren, kennung, err := repository.LadeAnmeldeverfahren(ctx, pool, p.kontoID)
	if err != nil {
		apierrors.AntworteFehlerMitMeldung(w, http.StatusServiceUnavailable, MeldungPruefungGestoert, err)
		return false
	}
	zugang := kontoZugang{verfahren: verfahren, email: p.email, ldapKennung: kennung}
	serverFehler := anmeldung.pruefePasswort(ctx, p.kontoID, zugang, p.passwort)
	if serverFehler == nil {
		merkeGeaendertesPasswort(ctx, anmeldung, p.sitzungID, p.passwort)
		return true
	}
	if !errors.Is(serverFehler, errAnmeldeserverNichtErreichbar) {
		anmeldung.bremse.merkeFehlversuch(p.schluessel)
		apierrors.AntworteFehlerMitMeldung(w, http.StatusUnauthorized, meldungPasswortFalsch, nil)
		return false
	}

	passt, vorhanden, err := anmeldung.Sitzungen.passwortPasst(ctx, p.sitzungID, p.passwort)
	if err != nil {
		apierrors.AntworteFehlerMitMeldung(w, http.StatusServiceUnavailable, MeldungPruefungGestoert, err)
		return false
	}
	if !vorhanden {
		apierrors.AntworteFehlerMitMeldung(w, http.StatusServiceUnavailable, meldungAnmeldeserverNichtErreichbar, serverFehler)
		return false
	}
	if !passt {
		anmeldung.bremse.merkeFehlversuch(p.schluessel)
		apierrors.AntworteFehlerMitMeldung(w, http.StatusUnauthorized, meldungPasswortFalsch, nil)
		return false
	}
	slog.Warn("Sperre ohne Anmeldeserver aufgeschlossen: Passwort gegen den Prüfwert der Anmeldung geprüft")
	return true
}

// merkeGeaendertesPasswort zieht den Prüfwert nach, wenn der Server ein Passwort angenommen
// hat, das nicht zu ihm passt. Scheitert das, bleibt der alte Prüfwert stehen; aufgeschlossen
// ist trotzdem, der Server hat entschieden.
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
