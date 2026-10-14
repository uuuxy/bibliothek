package tuer

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"bibliothek/apierrors"
	"bibliothek/pkg/kennung"

	"github.com/go-playground/validator/v10"
)

// Pruefer ist der gemeinsame Struct-Validator für alle Eingangs-Payloads. Bewusst EINE
// Instanz: validator.New() baut bei jedem Aufruf einen eigenen Regel-Cache auf.
var Pruefer = newPruefer()

// regelUUIDOderLeer: Eine Kennung, die als UUID in die Datenbank geht, muss eine sein —
// sonst kommt `invalid input syntax for type uuid` (22P02) als 500 zurück (ZAP-Lauf
// 13.09.2026, uuid_eingaben_test.go). Leer bleibt erlaubt: Die Handler melden „… fehlt"
// selbst, und optionale Felder kommen vom Frontend auch als "". Das eingebaute `uuid` kann
// das nicht — `omitempty,uuid` weist einen *string, der auf "" zeigt, mit 400 ab.
// Schreibweise am Feld: `validate:"omitempty,uuid_oder_leer"` (Listen: `omitempty,dive,uuid_oder_leer`).
const regelUUIDOderLeer = "uuid_oder_leer"

func newPruefer() *validator.Validate {
	v := validator.New()
	if err := v.RegisterValidation(regelUUIDOderLeer, func(fl validator.FieldLevel) bool {
		wert := fl.Field().String()
		return wert == "" || kennung.IstUUID(wert)
	}); err != nil {
		panic(err)
	}
	return v
}

// Pruefmeldung übersetzt eine ungültige Kennung in einen Satz; alle anderen
// Validierungsfehler bleiben, wie sie sind.
func Pruefmeldung(err error) error {
	var fehler validator.ValidationErrors
	if errors.As(err, &fehler) {
		for _, f := range fehler {
			if f.Tag() == regelUUIDOderLeer {
				return fmt.Errorf("ungültige Kennung im Feld %s", f.Field())
			}
		}
	}
	return err
}

// LeseUUIDParameter liest einen Query-Parameter, der eine UUID sein muss. Leer bleibt leer;
// alles andere, das keine UUID ist, ist ein Bedienfehler statt eines 500 aus Postgres.
func LeseUUIDParameter(r *http.Request, name string) (string, error) {
	wert := r.URL.Query().Get(name)
	if wert != "" && !kennung.IstUUID(wert) {
		return "", fmt.Errorf("%s ist keine gültige Kennung", name)
	}
	return wert, nil
}

// LeseAnfrageStreng ist DecodeAndValidate mit einem Unterschied: Ein Feld, das
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
func LeseAnfrageStreng[T any](w http.ResponseWriter, r *http.Request, ziel *T) bool {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(ziel); err != nil {
		apierrors.SendHTTPError(w, http.StatusBadRequest, err)
		return false
	}
	if err := Pruefer.Struct(ziel); err != nil {
		apierrors.SendHTTPError(w, http.StatusBadRequest, Pruefmeldung(err))
		return false
	}
	return true
}

// LeseAnfrage decodes the JSON request body and validates the struct. Unbekannte
// Felder werden dabei still verworfen — wo das nicht in Ordnung ist, steht
// DecodeStrictAndValidate darüber.
func LeseAnfrage[T any](w http.ResponseWriter, r *http.Request, ziel *T) bool {
	if err := json.NewDecoder(r.Body).Decode(ziel); err != nil {
		apierrors.SendHTTPError(w, http.StatusBadRequest, err)
		return false
	}
	if err := Pruefer.Struct(ziel); err != nil {
		apierrors.SendHTTPError(w, http.StatusBadRequest, Pruefmeldung(err))
		return false
	}
	return true
}
