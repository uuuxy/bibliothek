package benutzer

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
// Eindeutigkeit prüfen. Bewusst getrennt von den Handlern (konto_anlegen_aendern.go) und
// von der Rechtetrennung (admin_schutz.go) — drei verschiedene Fragen:
// „ist die Eingabe wohlgeformt", „wer ruft auf", „was macht der Endpunkt".

// normalisiereRolle bildet die Eingaberolle auf einen gültigen DB-Enum-Wert
// ab; unbekannte Rollen werden auf "mitarbeiter" zurückgesetzt.
//
// Die Rückfallebene ist die HARMLOSESTE Rolle, nicht die nächstliegende: Ein Tippfehler
// darf niemals nach oben führen. Wer "admin" schreiben will, muss es exakt treffen — und
// die Vergabe gestattet zusätzlich nur pruefeAdminVergabe.
func normalisiereRolle(rolle string) string {
	klein := strings.ToLower(strings.TrimSpace(rolle))
	switch klein {
	case "admin", "leitung", "kollegium", "mitarbeiter", "helfer":
		return klein
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
func pruefeEmailEindeutig(ctx context.Context, w http.ResponseWriter, kontoAbfragen repository.KontoAbfragen, email, ausgenommeneID string) bool {
	vergeben, err := kontoAbfragen.IstEmailVergeben(ctx, email, ausgenommeneID)
	if err != nil {
		apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
		return false
	}
	if vergeben {
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
// Schul-E-Mail in der Akte (LegeKollegiumskontoAn); dorthin verweist die Antwort (409).
//
// Was zählt, entscheidet KontoAbfragen.HatLeserzeileOhneKonto.
func pruefeKeineLeserzeileOhneKonto(ctx context.Context, w http.ResponseWriter, kontoAbfragen repository.KontoAbfragen, vorname, nachname string) bool {
	vorhanden, err := kontoAbfragen.HatLeserzeileOhneKonto(ctx, vorname, nachname)
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

// ausweisPruefung bündelt die Parameter für die Prüfung der Barcode-Eindeutigkeit.
type ausweisPruefung struct {
	nummer          string
	ausgenommeneID  string
	konfliktmeldung string
}

// pruefeAusweisnummerEindeutig liefert den optionalen Barcode-Pointer und validiert dessen
// Eindeutigkeit. Ist kein Barcode gesetzt, wird (nil, true) geliefert. Bei Konflikt
// oder DB-Fehler wird die HTTP-Antwort direkt geschrieben (ok=false).
func pruefeAusweisnummerEindeutig(ctx context.Context, w http.ResponseWriter, kontoAbfragen repository.KontoAbfragen, pruefung ausweisPruefung) (nummer *string, ok bool) {
	if pruefung.nummer == "" {
		return nil, true
	}
	vergeben, err := kontoAbfragen.IstAusweisnummerVergeben(ctx, pruefung.nummer, pruefung.ausgenommeneID)
	if err != nil {
		apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
		return nil, false
	}
	if vergeben {
		apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New(pruefung.konfliktmeldung))
		return nil, false
	}
	return &pruefung.nummer, true
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
