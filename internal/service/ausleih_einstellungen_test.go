package service

import (
	"fmt"
	"reflect"
	"testing"

	"bibliothek/repository"
)

// Jedes Feld der Einstellungen der Ausleihe kommt aus dem gleichnamigen Feld der Einstellungen.
// Die Quelle trägt je Feld einen eigenen Wert; ein Feld, das hier dazukommt und nicht gefüllt
// wird, bleibt leer und fällt auf.
func TestAusleihEinstellungenAus_JedesFeldAusSeinerQuelle(t *testing.T) {
	quelle := &repository.SystemEinstellungen{}
	q := reflect.ValueOf(quelle).Elem()
	ziel := reflect.TypeOf(SystemEinstellungen{})
	for i := range ziel.NumField() {
		name := ziel.Field(i).Name
		feld := q.FieldByName(name)
		if !feld.IsValid() {
			t.Fatalf("repository.SystemEinstellungen hat kein Feld %s", name)
		}
		switch feld.Kind() {
		case reflect.Int:
			feld.SetInt(int64(100 + i))
		case reflect.String:
			feld.SetString(fmt.Sprintf("Wert von %s", name))
		case reflect.Bool:
			feld.SetBool(true)
		case reflect.Pointer:
			wert := fmt.Sprintf("Wert von %s", name)
			feld.Set(reflect.ValueOf(&wert))
		default:
			t.Fatalf("Feld %s hat eine Art, die der Test nicht füllt: %s", name, feld.Kind())
		}
	}

	ist := reflect.ValueOf(ausleihEinstellungenAus(quelle)).Elem()
	for i := range ziel.NumField() {
		name := ziel.Field(i).Name
		if !reflect.DeepEqual(ist.Field(i).Interface(), q.FieldByName(name).Interface()) {
			t.Errorf("%s: %v, erwartet den Wert der Quelle %v", name, ist.Field(i).Interface(), q.FieldByName(name).Interface())
		}
	}
}
