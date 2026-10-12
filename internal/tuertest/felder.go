package tuertest

import (
	"fmt"
	"reflect"
	"time"
)

// LeereFelder nennt jedes Feld eines Werts, das seinen Nullwert trägt, auch in den Elementen
// einer Liste. Eine leere Liste zählt als leer.
func LeereFelder(wert reflect.Value, pfad string) []string {
	switch wert.Kind() {
	case reflect.Struct:
		if _, zeit := wert.Interface().(time.Time); zeit {
			break
		}
		var leer []string
		for i := 0; i < wert.NumField(); i++ {
			leer = append(leer, LeereFelder(wert.Field(i), pfad+"."+wert.Type().Field(i).Name)...)
		}
		return leer
	case reflect.Slice:
		if wert.Len() == 0 {
			return []string{pfad}
		}
		var leer []string
		for i := 0; i < wert.Len(); i++ {
			leer = append(leer, LeereFelder(wert.Index(i), fmt.Sprintf("%s[%d]", pfad, i))...)
		}
		return leer
	}
	if wert.IsZero() {
		return []string{pfad}
	}
	return nil
}
