package api

import (
	"encoding/json"
	"net/http"

	"bibliothek/apierrors"
	"bibliothek/pkg/httpresp"

	"github.com/go-playground/validator/v10"
)

// Validate ist der gemeinsame Struct-Validator für alle Eingangs-Payloads. Bewusst EINE
// Instanz: validator.New() baut bei jedem Aufruf einen eigenen Regel-Cache auf.
var Validate = validator.New()

// DecodeStrictAndValidate ist DecodeAndValidate mit einem Unterschied: Ein Feld, das
// das Ziel-Struct nicht kennt, ist ein FEHLER (400) statt eines stillen Verlusts.
//
// Warum nicht überall? Weil mehrere Endpunkte bewusst das ganze Objekt entgegennehmen,
// das sie vorher ausgeliefert haben — samt der Felder, die nur der Server füllt
// (`id`, `verfuegbar`, `gesamt`, `sortOrder` beim Buch). Für die wäre Strenge kein
// Gewinn, sondern ein 400 auf einem Bildschirm, der heute funktioniert.
//
// Wo die Nutzlast dagegen aus einer GESCHLOSSENEN Liste entsteht — die
// Einstellungs-Kategorien bauen ihren Patch aus benannten Schlüsseln —, ist ein
// unbekanntes Feld immer ein Fehler, und zwar der teuerste: Am 23.08.2026 schickte die
// Mail-Kategorie ihre Felder noch in Unterstrich-Schreibweise, der Decoder verwarf sie
// still, und die Oberfläche meldete "gespeichert" (Commit 488f51d9). Genau diese
// Klasse — 200 trotz Datenverlust — ist die teuerste des Projekts.
func DecodeStrictAndValidate[T any](w http.ResponseWriter, r *http.Request, target *T) bool {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(target); err != nil {
		apierrors.SendHTTPError(w, http.StatusBadRequest, err)
		return false
	}
	if err := Validate.Struct(target); err != nil {
		apierrors.SendHTTPError(w, http.StatusBadRequest, err)
		return false
	}
	return true
}

// DecodeAndValidate decodes the JSON request body and validates the struct. Unbekannte
// Felder werden dabei still verworfen — wo das nicht in Ordnung ist, steht
// DecodeStrictAndValidate darüber.
func DecodeAndValidate[T any](w http.ResponseWriter, r *http.Request, target *T) bool {
	if err := json.NewDecoder(r.Body).Decode(target); err != nil {
		apierrors.SendHTTPError(w, http.StatusBadRequest, err)
		return false
	}
	if err := Validate.Struct(target); err != nil {
		apierrors.SendHTTPError(w, http.StatusBadRequest, err)
		return false
	}
	return true
}

// RespondJSON encodes the payload as JSON and sends it with the given HTTP status code.
// It sets the Content-Type header to application/json.
func RespondJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	httpresp.Encode(w, payload)
}

// RespondSuccess is a convenience function that sends a {"status": "success"} JSON response.
func RespondSuccess(w http.ResponseWriter) {
	RespondJSON(w, http.StatusOK, map[string]string{"status": "success"})
}
