// Package httpresp schreibt den Rumpf einer Antwort. Sind Status und Kopf gesendet, lässt sich
// ein Fehler beim Schreiben nicht mehr als Status melden; er steht dann im Log.
package httpresp

import (
	"encoding/json"
	"io"
	"log/slog"
	"reflect"
)

// Schreibe schreibt b nach w und einen Fehler dabei ins Log.
func Schreibe(w io.Writer, b []byte) {
	if _, err := w.Write(b); err != nil {
		slog.Warn("httpresp: writing response body failed", "fehler", err)
	}
}

// SchreibeJSON schreibt antwort als JSON nach w und einen Fehler dabei ins Log.
func SchreibeJSON(w io.Writer, antwort any) {
	if err := json.NewEncoder(w).Encode(alsListe(antwort)); err != nil {
		slog.Error("httpresp: encoding JSON response failed", "fehler", err)
	}
}

// alsListe ersetzt eine Liste ohne Wert durch eine leere, damit eine Antwort mit einer Liste
// immer [] trägt und nie null. Die Oberfläche ruft auf einer Liste length und map und bricht
// an null ab, gerade an einer frischen Anlage, an der viele Listen leer sind. Die Stelle ist
// hier, weil jede JSON-Antwort durch sie läuft.
//
// Ersetzt wird auch in den Werten einer Map, eine Ebene tief, weil viele Antworten ihre Liste
// in einem Umschlag {"data": …} tragen. In Strukturen und tiefer nicht: Deren Felder legt
// die Tür fest, die sie schreibt.
func alsListe(antwort any) any {
	if antwort == nil {
		return antwort
	}
	v := reflect.ValueOf(antwort)
	switch v.Kind() {
	case reflect.Slice:
		if v.IsNil() {
			return reflect.MakeSlice(v.Type(), 0, 0).Interface()
		}
	case reflect.Map:
		for _, key := range v.MapKeys() {
			wert := v.MapIndex(key)
			// Die Werte einer map[string]any stecken in einer Schnittstelle.
			if wert.Kind() == reflect.Interface && !wert.IsNil() {
				wert = wert.Elem()
			}
			if wert.Kind() == reflect.Slice && wert.IsNil() {
				v.SetMapIndex(key, reflect.MakeSlice(wert.Type(), 0, 0))
			}
		}
	}
	return antwort
}
