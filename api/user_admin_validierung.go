package api

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"bibliothek/apierrors"
	"bibliothek/repository"
)

// Feldprüfungen der Benutzerverwaltung: Rolle normalisieren, E-Mail und Barcode auf
// Eindeutigkeit prüfen. Bewusst getrennt von den Handlern (user_admin_mutations.go) und
// von der Rechtetrennung (user_admin_eskalation.go) — drei verschiedene Fragen:
// „ist die Eingabe wohlgeformt", „wer ruft auf", „was macht der Endpunkt".

// normalisiereBenutzerRolle bildet die Eingaberolle auf einen gültigen DB-Enum-Wert
// ab; unbekannte Rollen werden auf "mitarbeiter" zurückgesetzt.
//
// Die Rückfallebene ist die HARMLOSESTE Rolle, nicht die nächstliegende: Ein Tippfehler
// darf niemals nach oben führen. Wer "admin" schreiben will, muss es exakt treffen — und
// die Vergabe gestattet zusätzlich nur pruefeAdminVergabe.
func normalisiereBenutzerRolle(rolle string) string {
	dbEnumRole := strings.ToLower(strings.TrimSpace(rolle))
	switch dbEnumRole {
	case "admin", "kollegium", "mitarbeiter", "helfer":
		return dbEnumRole
	default:
		return "mitarbeiter"
	}
}

// pruefeEmailEindeutig prüft die E-Mail-Eindeutigkeit (excludeID leer bei Neuanlage,
// sonst die eigene ID). Bei Konflikt oder DB-Fehler wird die HTTP-Antwort direkt
// geschrieben und false zurückgegeben.
//
// Die Eindeutigkeit ist hier mehr als Datenhygiene: Die Anmeldung findet den Benutzer
// über seine E-Mail (auth/handlers.go). Zwei Datensätze mit derselben Adresse hießen,
// dass nicht mehr feststeht, wessen Konto ein Login öffnet.
func pruefeEmailEindeutig(ctx context.Context, w http.ResponseWriter, userRepo repository.UserRepository, email, excludeID string) bool {
	exists, err := userRepo.CheckEmailExists(ctx, email, excludeID)
	if err != nil {
		apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
		return false
	}
	if exists {
		apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("ein Benutzer mit dieser E-Mail existiert bereits"))
		return false
	}
	return true
}

// BarcodePruefOptionen bündelt die Parameter für die Prüfung der Barcode-Eindeutigkeit.
type BarcodePruefOptionen struct {
	BarcodeID   string
	ExcludeID   string
	KonfliktMsg string
}

// pruefeBarcodeEindeutig liefert den optionalen Barcode-Pointer und validiert dessen
// Eindeutigkeit. Ist kein Barcode gesetzt, wird (nil, true) geliefert. Bei Konflikt
// oder DB-Fehler wird die HTTP-Antwort direkt geschrieben (ok=false).
func pruefeBarcodeEindeutig(ctx context.Context, w http.ResponseWriter, userRepo repository.UserRepository, opt BarcodePruefOptionen) (barcode *string, ok bool) {
	if opt.BarcodeID == "" {
		return nil, true
	}
	exists, err := userRepo.CheckBarcodeExists(ctx, opt.BarcodeID, opt.ExcludeID)
	if err != nil {
		apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
		return nil, false
	}
	if exists {
		apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New(opt.KonfliktMsg))
		return nil, false
	}
	return &opt.BarcodeID, true
}
