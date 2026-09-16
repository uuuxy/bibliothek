package repository

import (
	"reflect"
	"strings"
	"testing"
)

// Jede Zahl des Patches steht in zahlenFelder — sonst wird sie nie geschrieben.
//
// Die Liste ist seit dem 16.09.2026 der einzige Weg, auf dem eine Zahl in die Datenbank
// kommt (paarSammler.zahlen). Wer ein neues Zahlenfeld an EinstellungenPatch hängt und
// die Liste vergisst, bekommt kein rotes Licht, sondern ein Feld, das sich speichern
// lässt und beim nächsten Laden wieder leer ist — dieselbe Klasse, gegen die es die
// Schlüssel-Paritäts-Gates schon gibt, nur eine Ebene tiefer.
func TestZahlenFelder_KennenJedesZahlenfeldDesPatches(t *testing.T) {
	patch := &EinstellungenPatch{}
	gelistet := map[string]bool{}
	for _, f := range zahlenFelder(patch) {
		if f.label == "" {
			t.Errorf("Schlüssel %q ohne Beschriftung — die Meldung nennt dem Menschen sonst "+
				"nur einen Datenbank-Schlüssel", f.schluessel)
		}
		if f.min > f.max {
			t.Errorf("Schlüssel %q hat eine leere Spanne (%d bis %d): Jeder Wert wäre falsch",
				f.schluessel, f.min, f.max)
		}
		if gelistet[f.schluessel] {
			t.Errorf("Schlüssel %q steht doppelt in der Liste — das Upsert liefe mit "+
				"SQLSTATE 21000 auf", f.schluessel)
		}
		gelistet[f.schluessel] = true
	}

	typ := reflect.TypeOf(*patch)
	for i := range typ.NumField() {
		feld := typ.Field(i)
		if feld.Type.Kind() != reflect.Pointer || feld.Type.Elem().Kind() != reflect.Int {
			continue
		}
		schluessel := strings.Split(feld.Tag.Get("json"), ",")[0]
		if !gelistet[schluessel] {
			t.Errorf("EinstellungenPatch.%s (%q) fehlt in zahlenFelder — das Feld ließe sich "+
				"schicken, würde nie gespeichert und wäre beim nächsten Laden wieder leer",
				feld.Name, schluessel)
		}
	}
}

// Selbstprobe des Detektors: Er muss ein fehlendes Feld auch wirklich finden. Ohne diese
// Gegenprobe wäre der Test oben grün, wenn zahlenFelder eines Tages nichts mehr liefert.
func TestZahlenFelder_DetektorFindetEinFehlendesFeld(t *testing.T) {
	typ := reflect.TypeOf(EinstellungenPatch{})
	zahlenImStruct := 0
	for i := range typ.NumField() {
		if f := typ.Field(i); f.Type.Kind() == reflect.Pointer && f.Type.Elem().Kind() == reflect.Int {
			zahlenImStruct++
		}
	}
	if zahlenImStruct != len(zahlenFelder(&EinstellungenPatch{})) {
		t.Fatalf("%d Zahlenfelder im Struct, %d in der Liste — eines der beiden Enden ist "+
			"nachzuziehen", zahlenImStruct, len(zahlenFelder(&EinstellungenPatch{})))
	}
	if zahlenImStruct == 0 {
		t.Fatal("kein einziges Zahlenfeld gefunden — der Detektor misst nichts")
	}
}
