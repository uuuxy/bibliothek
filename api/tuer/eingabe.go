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

// Pruefer prüft die Felder einer Anfrage nach ihren validate-Tags. Es gibt nur diesen einen:
// validator.New() legt bei jedem Aufruf einen eigenen Cache der Regeln an.
var Pruefer = newPruefer()

// regelUUIDOderLeer verlangt eine UUID oder nichts. Eine Kennung, die als UUID in die Datenbank
// geht, muss eine sein, sonst kommt `invalid input syntax for type uuid` (22P02) als 500
// zurück; uuid_eingaben_test.go hält das an jeder Tür. Leer bleibt erlaubt, weil die Türen „…
// fehlt“ selbst melden und optionale Felder von der Oberfläche auch als "" kommen; das
// eingebaute `uuid` kann das nicht, `omitempty,uuid` weist einen *string, der auf "" zeigt, mit
// 400 ab.
//
// Am Feld steht `validate:"omitempty,uuid_oder_leer"`, an einer Liste
// `validate:"omitempty,dive,required,uuid_oder_leer"`: Ohne `required` ginge ein leeres Element
// durch.
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

// Pruefmeldung macht aus dem Fehler einer ungültigen Kennung einen Satz für die Oberfläche;
// jeden anderen Prüffehler gibt sie unverändert zurück.
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

// LeseUUIDParameter liest einen Parameter der Adresse, der eine UUID sein muss. Leer bleibt
// leer; ein anderer Wert, der keine UUID ist, wird ein Fehler, den die Tür als Bedienfehler
// meldet, statt eines 500 aus Postgres.
func LeseUUIDParameter(r *http.Request, name string) (string, error) {
	wert := r.URL.Query().Get(name)
	if wert != "" && !kennung.IstUUID(wert) {
		return "", fmt.Errorf("%s ist keine gültige Kennung", name)
	}
	return wert, nil
}

// LeseAnfrageStreng liest wie LeseAnfrage, lehnt aber ein Feld, das das Ziel nicht kennt, mit
// 400 ab, statt es still zu verwerfen.
//
// Nicht jede Tür liest streng: Mehrere nehmen das ganze Objekt zurück, das sie ausgeliefert
// haben, samt der Felder, die nur der Server füllt (`id`, `verfuegbar`, `gesamt`, `sortOrder`
// am Titel); dort brächte Strenge nur ein 400. Entsteht der Rumpf dagegen aus einer
// geschlossenen Liste benannter Schlüssel, wie bei den Kategorien der Einstellungen, ist ein
// unbekanntes Feld immer ein Fehler. Ohne Strenge verschwände es still, und die Oberfläche
// meldete „gespeichert“: 200 trotz Datenverlust, die teuerste Klasse von Fehlern im Projekt.
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

// LeseAnfrage liest den JSON-Rumpf der Anfrage in das Ziel und prüft seine Felder. Bei einem
// Fehler hat sie die Antwort (400) schon geschrieben und liefert false. Ein Feld, das das Ziel
// nicht kennt, verwirft sie still; wo das nicht genügt, gilt LeseAnfrageStreng.
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
