package benutzer

import (
	"context"
	"net/http"
	"strings"

	"bibliothek/apierrors"
	"bibliothek/auth"
)

// Trennung von manage_users und Administrator.
//
// manage_users ist ein Recht, das der Administrator weitergeben kann: Die Rechte-Matrix bietet
// es den Rollen unterhalb des Administrators an (frontend/src/lib/rechteKategorien.js). Ohne
// die Regeln dieser Datei käme es einem Administrator gleich, auf drei Wegen:
//
//  1. Selbstbeförderung: PUT /api/benutzer/{eigene-id} mit rolle:"admin". pruefeSelbstschutz
//     lehnt jede Änderung der eigenen Rolle ab.
//  2. Fremdbeförderung: POST oder PUT /api/benutzer mit rolle:"admin" für ein beliebiges
//     Konto. pruefeAdminVergabe verlangt dafür einen Administrator.
//  3. Übernahme eines Admin-Kontos ohne Änderung der Rolle: Die Anmeldung prüft die
//     Zugangsdaten per IMAP und sucht das Konto danach über seine E-Mail-Adresse
//     (auth/anmelden.go, pruefeIMAPZugang). Wer die E-Mail eines Admin-Kontos auf die
//     eigene Schuladresse setzt, meldet sich mit den eigenen Zugangsdaten an und bekommt die
//     Admin-Sitzung. Deshalb schützt pruefeAdminZiel das ganze Konto, nicht nur die Rolle.
//
// Die Regel dahinter: manage_users verwaltet Konten unterhalb der Administratorebene.
// Wer einen Administrator anlegen, verändern oder löschen will, muss selbst einer sein.

// istAdminRolle erkennt die Administratorrolle unabhängig von Groß-/Kleinschreibung.
func istAdminRolle(rolle string) bool {
	return strings.EqualFold(strings.TrimSpace(rolle), string(auth.RolleAdmin))
}

// istAufruferAdmin liest die Rolle des Aufrufers aus den Claims. Die stammt aus der
// Datenbank, nicht aus dem signierten Token — auth.PruefeToken überschreibt sie bei
// jeder Anfrage, eine Herabstufung wirkt also sofort.
func istAufruferAdmin(r *http.Request) bool {
	claims, ok := auth.ClaimsAusKontext(r.Context())
	return ok && istAdminRolle(string(claims.Rolle))
}

// pruefeAdminVergabe verlangt Administratorrechte, wenn die gewünschte Rolle "admin"
// ist. Bei Verstoß wird die HTTP-Antwort geschrieben und false geliefert.
func pruefeAdminVergabe(w http.ResponseWriter, r *http.Request, gewuenschteRolle string) bool {
	if !istAdminRolle(gewuenschteRolle) || istAufruferAdmin(r) {
		return true
	}
	apierrors.AntworteFehlerMitMeldung(w, http.StatusForbidden, "Die Rolle Administrator vergibt nur ein Administrator", nil)
	return false
}

// pruefeAdminZiel verlangt Administratorrechte, wenn das Ziel heute ein Administrator ist,
// gleich welches Feld geändert werden soll. Ein unbekanntes Ziel (kein Treffer) lässt die
// Prüfung passieren; darüber entscheidet die Tür.
func pruefeAdminZiel(ctx context.Context, w http.ResponseWriter, r *http.Request, kontoAbfragen kontoPflege, id string) bool {
	if istAufruferAdmin(r) {
		return true
	}
	rolle, err := kontoAbfragen.LadeRolle(ctx, id)
	if err != nil {
		apierrors.AntworteFehler(w, http.StatusInternalServerError, err)
		return false
	}
	if istAdminRolle(rolle) {
		apierrors.AntworteFehlerMitMeldung(w, http.StatusForbidden, "Das Konto eines Administrators bearbeitet nur ein Administrator", nil)
		return false
	}
	return true
}

// pruefeSelbstschutz verhindert, dass jemand die eigene Rolle ändert oder das eigene Konto
// deaktiviert, gleich welcher Rolle er angehört. Er darf nicht unterstellen, wer sich selbst
// bearbeitet, sei Admin: Dann scheiterte ein Mitarbeiter mit manage_users am Speichern seines
// eigenen Datensatzes, und durch ginge ausgerechnet rolle:"admin".
//
// neueRolle und aktiv sind nil, wenn die Änderung das Feld nicht nennt: Dann bleibt es, wie
// es ist, und es gibt nichts zu schützen.
func pruefeSelbstschutz(w http.ResponseWriter, r *http.Request, id string, neueRolle *string, aktiv *bool) bool {
	claims, ok := auth.ClaimsAusKontext(r.Context())
	if !ok || claims.KontoID != id {
		return true
	}
	if neueRolle != nil && !strings.EqualFold(strings.TrimSpace(*neueRolle), strings.TrimSpace(string(claims.Rolle))) {
		apierrors.AntworteFehlerMitMeldung(w, http.StatusForbidden, "Die eigene Rolle lässt sich nicht ändern. Das muss eine andere Person mit Administratorrechten tun.", nil)
		return false
	}
	if aktiv != nil && !*aktiv {
		apierrors.AntworteFehlerMitMeldung(w, http.StatusForbidden, "Das eigene Konto lässt sich nicht deaktivieren", nil)
		return false
	}
	return true
}
