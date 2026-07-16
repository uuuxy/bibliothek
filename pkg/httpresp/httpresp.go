// Package httpresp bündelt das Schreiben von HTTP-Antwortkörpern. Sind Status und
// Header erst einmal gesendet, lässt sich ein fehlgeschlagener Schreibvorgang nicht
// mehr in einen Fehlerstatus umwandeln – die einzig sinnvolle Reaktion ist Logging.
package httpresp

import (
	"encoding/json"
	"io"
	"log"
	"reflect"
)

// Write schreibt b nach w und protokolliert einen etwaigen Schreibfehler.
func Write(w io.Writer, b []byte) {
	if _, err := w.Write(b); err != nil {
		log.Printf("httpresp: writing response body failed: %v", err)
	}
}

// Encode serialisiert payload als JSON nach w und protokolliert einen etwaigen Fehler.
func Encode(w io.Writer, payload any) {
	if err := json.NewEncoder(w).Encode(alsListe(payload)); err != nil {
		log.Printf("httpresp: encoding JSON response failed: %v", err)
	}
}

// alsListe ersetzt eine nil-Slice durch eine leere Slice, damit Listen-Endpunkte
// IMMER [] liefern und niemals null.
//
// Hintergrund: In Go ist "var xs []T" ohne Treffer nil, und json.Marshal macht daraus
// null (nicht []). Clients, die auf einer Liste .length/.map aufrufen, brechen dann ab —
// ausgerechnet auf einer frisch aufgesetzten Installation, wo viele Listen leer sind.
// Genau so ist die Schuelerdatei beim Erst-Deployment gecrasht.
//
// Die Normalisierung gehoert hierher, an die eine Stelle, durch die jede JSON-Antwort
// laeuft: Der Vertrag "eine Liste ist ein Array" gilt damit fuer alle Endpunkte — auch
// fuer kuenftige, ohne dass jemand daran denken muss. Einzelne Handler haben das bisher
// von Hand abgefangen (z. B. inventur/class_books_handler.go); noetig ist das nun nicht
// mehr.
//
// Neben der obersten Ebene werden auch Map-WERTE normalisiert (eine Ebene tief), weil
// viele Endpunkte Listen im Umschlag {"data": ...} liefern — dort saesse das null
// sonst eine Ebene tiefer und der Schutz liefe ins Leere. Bewusst keine Rekursion in
// Structs oder tiefere Ebenen: Deren Felder sind typisierte Vertraege, die der
// jeweilige Autor kennt; hier geht es nur um die generischen Antwort-Umschlaege.
func alsListe(payload any) any {
	if payload == nil {
		return payload
	}
	v := reflect.ValueOf(payload)
	switch v.Kind() {
	case reflect.Slice:
		if v.IsNil() {
			return reflect.MakeSlice(v.Type(), 0, 0).Interface()
		}
	case reflect.Map:
		for _, key := range v.MapKeys() {
			wert := v.MapIndex(key)
			// Map-Werte sind meist interface{} (map[string]any) — erst auspacken.
			if wert.Kind() == reflect.Interface && !wert.IsNil() {
				wert = wert.Elem()
			}
			if wert.Kind() == reflect.Slice && wert.IsNil() {
				v.SetMapIndex(key, reflect.MakeSlice(wert.Type(), 0, 0))
			}
		}
	}
	return payload
}
