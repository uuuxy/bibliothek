package tuer

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strings"

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
	// Die Meldung nennt ein Feld so, wie die Anfrage es schickt.
	v.RegisterTagNameFunc(func(feld reflect.StructField) string {
		name, _, _ := strings.Cut(feld.Tag.Get("json"), ",")
		if name == "" || name == "-" {
			return feld.Name
		}
		return name
	})
	if err := v.RegisterValidation(regelUUIDOderLeer, func(fl validator.FieldLevel) bool {
		wert := fl.Field().String()
		return wert == "" || kennung.IstUUID(wert)
	}); err != nil {
		panic(err)
	}
	return v
}

// Pruefmeldung macht aus dem Fehler der Prüfung einen Satz für die Oberfläche. Sie nennt das
// erste abgelehnte Feld mit seinem Pfad in der Anfrage, in einer Liste mit der Stelle
// (items[1].titel_id), und sagt, was dort stehen muss (M3, Text fields: „error text should
// describe how to avoid the error").
func Pruefmeldung(err error) string {
	var fehler validator.ValidationErrors
	if !errors.As(err, &fehler) || len(fehler) == 0 {
		return apierrors.MeldungAnfrageUnlesbar
	}
	f := fehler[0]
	// Der Pfad beginnt mit dem Namen des Typs im Go-Code; die Anfrage kennt ihn nicht.
	_, pfad, _ := strings.Cut(f.Namespace(), ".")
	feld := "„" + pfad + "“"
	switch f.Tag() {
	case "required":
		return "Bitte das Feld " + feld + " ausfüllen"
	case regelUUIDOderLeer, "uuid":
		return "Ungültige Kennung im Feld " + feld
	case "email":
		return fmt.Sprintf("„%v“ ist keine E-Mail-Adresse", f.Value())
	case "oneof":
		return "Im Feld " + feld + " ist nur einer dieser Werte möglich: " + strings.ReplaceAll(f.Param(), " ", ", ")
	case "min", "max":
		return grenzmeldung(f, feld)
	}
	return "Das Feld " + feld + " ist ungültig"
}

// grenzmeldung nennt die Grenze aus min oder max: bei einer Liste die Zahl der Einträge, sonst
// den Wert.
func grenzmeldung(f validator.FieldError, feld string) string {
	liste := f.Kind() == reflect.Slice || f.Kind() == reflect.Array || f.Kind() == reflect.Map
	switch {
	case liste && f.Tag() == "min":
		return "Das Feld " + feld + " braucht mindestens " + f.Param() + " Einträge"
	case liste:
		return "Das Feld " + feld + " hat höchstens " + f.Param() + " Einträge"
	case f.Tag() == "min":
		return "Das Feld " + feld + " muss mindestens " + f.Param() + " sein"
	}
	return "Das Feld " + feld + " darf höchstens " + f.Param() + " sein"
}

// LeseUUIDParameter liest einen Parameter der Adresse, der eine UUID sein muss. Leer bleibt
// leer; ein anderer Wert, der keine UUID ist, wird ein Bedienfehler mit Meldung statt eines 500
// aus Postgres.
func LeseUUIDParameter(r *http.Request, name string) (string, error) {
	wert := r.URL.Query().Get(name)
	if wert != "" && !kennung.IstUUID(wert) {
		return "", apierrors.BadRequest(fmt.Sprintf("In der Adresse ist „%s“ keine gültige Kennung", name), nil)
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
		apierrors.AntworteFehlerMitMeldung(w, http.StatusBadRequest, apierrors.MeldungAnfrageUnlesbar, err)
		return false
	}
	if err := Pruefer.Struct(ziel); err != nil {
		apierrors.AntworteFehlerMitMeldung(w, http.StatusBadRequest, Pruefmeldung(err), err)
		return false
	}
	return true
}

// LeseAnfrage liest den JSON-Rumpf der Anfrage in das Ziel und prüft seine Felder. Bei einem
// Fehler hat sie die Antwort (400) schon geschrieben und liefert false. Ein Feld, das das Ziel
// nicht kennt, verwirft sie still; wo das nicht genügt, gilt LeseAnfrageStreng.
func LeseAnfrage[T any](w http.ResponseWriter, r *http.Request, ziel *T) bool {
	if err := json.NewDecoder(r.Body).Decode(ziel); err != nil {
		apierrors.AntworteFehlerMitMeldung(w, http.StatusBadRequest, apierrors.MeldungAnfrageUnlesbar, err)
		return false
	}
	if err := Pruefer.Struct(ziel); err != nil {
		apierrors.AntworteFehlerMitMeldung(w, http.StatusBadRequest, Pruefmeldung(err), err)
		return false
	}
	return true
}
