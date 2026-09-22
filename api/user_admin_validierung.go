package api

import (
	"context"
	"errors"
	"fmt"
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
	case "admin", "leitung", "kollegium", "mitarbeiter", "helfer":
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

// pruefeKeineLeserzeileOhneKonto bremst, wenn ein Kollege gleichen Namens schon eine
// Leserzeile OHNE Konto hat (OFFEN.md 5.17): nach dem Löschen seines Kontos oder aus der
// Littera-Übernahme. Der Wächter trg_benutzer_hat_leserzeile hängt jedem neuen Konto eine
// frische Leserzeile an — dieselbe Person stünde dann zweimal in der Leserdatei, einmal mit
// Ausweis und Geschichte, einmal leer. Der Weg zum Konto an der vorhandenen Zeile ist die
// Schul-E-Mail in der Akte (LegeKollegiumskonto); dorthin verweist die Antwort (409).
//
// Was zählt, entscheidet UserRepository.LeserzeileOhneKonto.
func pruefeKeineLeserzeileOhneKonto(ctx context.Context, w http.ResponseWriter, userRepo repository.UserRepository, vorname, nachname string) bool {
	vorhanden, err := userRepo.LeserzeileOhneKonto(ctx, vorname, nachname)
	if err != nil {
		apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
		return false
	}
	if vorhanden {
		apierrors.SendHTTPError(w, http.StatusConflict,
			fmt.Errorf("für %s %s steht schon eine Leserzeile ohne Konto in der Leserdatei — das Konto entsteht über die Schul-E-Mail in der Akte, sonst gäbe es dieselbe Person zweimal", vorname, nachname))
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

// meldeAusweisKollision übersetzt die Ablehnung der Datenbank beim Eintragen einer
// Ausweisnummer in eine Auskunft (409) und sagt, ob sie geschrieben wurde. Zwei Wächter
// greifen hinter der Vorprüfung (pruefeBarcodeEindeutig, sie kennt nur die Leserzeilen):
// dieselbe Nummer bei einer anderen Person (Migration 118/125, zwei Arbeitsplätze
// gleichzeitig) und die Nummer eines Buchs (Migration 131). Bis zum 22.09.2026 kam beides
// als 500 „interner Fehler“ an — die drei Schüler-Türen übersetzen es seit ihrer Anlage
// (Rasterdurchgang 22.09.2026, Frage 5).
func meldeAusweisKollision(w http.ResponseWriter, err error) bool {
	switch {
	case repository.IstNummerBuchOderAusweisKollision(err):
		apierrors.SendHTTPError(w, http.StatusConflict,
			errors.New("diese Nummer ist der Barcode eines Buchs und kann kein Ausweis sein"))
		return true
	case repository.IstAusweisKollision(err):
		apierrors.SendHTTPError(w, http.StatusConflict,
			errors.New("diese Ausweisnummer trägt bereits eine andere Person (Schüler oder Kollegium)"))
		return true
	}
	return false
}
