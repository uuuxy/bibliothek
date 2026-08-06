package api

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"bibliothek/apierrors"
	"bibliothek/auth"
	"bibliothek/repository"
)

// Trennung von manage_users und Administrator.
//
// manage_users ist ein delegierbares Recht: Der PermissionManager bietet es für
// MITARBEITER, LEHRER und HELFER an (frontend/src/lib/permissionMetadata.js). Geseedet
// ist es nur für ADMIN — vergeben wurde es damit aber trotzdem, und bis zum 06.08.2026
// war es gleichbedeutend mit Administrator. Drei Wege führten dahin:
//
//  1. Selbstbeförderung. PUT /api/benutzer/{eigene-id} mit rolle:"admin" ging durch.
//     Der Selbstschutz unterstellte, wer sich selbst bearbeitet, sei bereits Admin,
//     und ließ ausgerechnet den einen Fall zu, den er verhindern sollte.
//  2. Fremdbeförderung. POST/PUT /api/benutzer mit rolle:"admin" für ein beliebiges
//     Konto — normalisiereBenutzerRolle nahm "admin" ohne jede Prüfung entgegen.
//  3. Übernahme eines Admin-Kontos ganz OHNE Rollenänderung. Die Anmeldung prüft die
//     Zugangsdaten per IMAP und sucht den Benutzer danach über seine E-Mail-Adresse
//     (auth/handlers.go, verifyIMAPCredentials). Wer die E-Mail eines Admin-Datensatzes
//     auf die eigene Schuladresse setzt, meldet sich beim nächsten Login mit den EIGENEN
//     IMAP-Zugangsdaten an und bekommt die Admin-Sitzung. Deshalb schützt pruefeAdminZiel
//     den ganzen Datensatz und nicht nur das Rollenfeld.
//
// Die Regel dahinter: manage_users verwaltet Konten unterhalb der Administratorebene.
// Wer einen Administrator anlegen, verändern oder löschen will, muss selbst einer sein.

// istAdminRolle erkennt die Administratorrolle unabhängig von Groß-/Kleinschreibung.
func istAdminRolle(rolle string) bool {
	return strings.EqualFold(strings.TrimSpace(rolle), string(auth.RoleAdmin))
}

// aufruferIstAdmin liest die Rolle des Aufrufers aus den Claims. Die stammt aus der
// Datenbank, nicht aus dem signierten Token — auth.VerifyToken überschreibt sie bei
// jeder Anfrage, eine Herabstufung wirkt also sofort.
func aufruferIstAdmin(r *http.Request) bool {
	claims, ok := auth.GetClaims(r.Context())
	return ok && istAdminRolle(string(claims.Rolle))
}

// pruefeAdminVergabe verlangt Administratorrechte, wenn die gewünschte Rolle "admin"
// ist. Bei Verstoß wird die HTTP-Antwort geschrieben und false geliefert.
func pruefeAdminVergabe(w http.ResponseWriter, r *http.Request, gewuenschteRolle string) bool {
	if !istAdminRolle(gewuenschteRolle) || aufruferIstAdmin(r) {
		return true
	}
	apierrors.SendHTTPError(w, http.StatusForbidden,
		errors.New("die Rolle Administrator kann nur von einem Administrator vergeben werden"))
	return false
}

// pruefeAdminZiel verlangt Administratorrechte, wenn das Ziel HEUTE ein Administrator
// ist — unabhängig davon, welches Feld geändert werden soll. Ein unbekanntes Ziel
// (kein Treffer) lässt die Prüfung passieren; darüber entscheidet der Handler.
func pruefeAdminZiel(ctx context.Context, w http.ResponseWriter, r *http.Request, userRepo repository.UserRepository, id string) bool {
	if aufruferIstAdmin(r) {
		return true
	}
	rolle, err := userRepo.GetRolleByID(ctx, id)
	if err != nil {
		apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
		return false
	}
	if istAdminRolle(rolle) {
		apierrors.SendHTTPError(w, http.StatusForbidden,
			errors.New("das Konto eines Administrators kann nur von einem Administrator bearbeitet werden"))
		return false
	}
	return true
}

// pruefeSelbstschutz verhindert, dass ein Benutzer die EIGENE Rolle ändert oder das
// eigene Konto deaktiviert — gleich welcher Rolle er angehört.
//
// Die Vorgängerfassung prüfte nur auf != "ADMIN" und unterstellte damit, jeder
// Selbstbearbeiter sei Admin. Für einen MITARBEITER mit manage_users hatte das zwei
// Folgen: Das Speichern des eigenen Datensatzes mit UNVERÄNDERTER Rolle scheiterte
// mit „eigene Admin-Rolle kann nicht herabgestuft werden", und die einzige
// Selbstbearbeitung, die durchging, war rolle:"admin". Der Wächter verhinderte die
// Eskalation nicht — er ließ nur sie übrig.
func pruefeSelbstschutz(w http.ResponseWriter, r *http.Request, id string, neueRolle string, aktiv bool) bool {
	claims, ok := auth.GetClaims(r.Context())
	if !ok || claims.UserID != id {
		return true
	}
	if !strings.EqualFold(strings.TrimSpace(neueRolle), strings.TrimSpace(string(claims.Rolle))) {
		apierrors.SendHTTPError(w, http.StatusForbidden,
			errors.New("die eigene Rolle kann nicht geändert werden — das muss eine andere Person mit Administratorrechten tun"))
		return false
	}
	if !aktiv {
		apierrors.SendHTTPError(w, http.StatusForbidden, errors.New("eigenes Konto kann nicht deaktiviert werden"))
		return false
	}
	return true
}
