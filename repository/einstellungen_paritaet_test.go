package repository

import (
	"reflect"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

// Ein Feld der Einstellungen hat zwei Enden: paareAusPatch schreibt es, setzeEinstellung liest
// es zurück. Fehlt das zweite, speichert das Feld scheinbar und ist beim nächsten Laden leer;
// der Vergleich von Oberfläche und Patch sieht das nicht, weil dort alles stimmt. So ging
// alarm_empfaenger verloren, und der Alarm fiel auf alle aktiven Admins zurück.
//
// Der Test am Postgres daneben belegt dasselbe für drei Felder, dieser für alle und ohne
// Datenbank. Er meldet sich, sobald jemand ein Feld hinzufügt und das andere Ende vergisst.
func TestEinstellungen_SchreibpfadUndLesepfadSindDeckungsgleich(t *testing.T) {
	// Ein Patch, in dem jedes Feld gesetzt ist: Nur dann liefert paareAusPatch jeden
	// Schlüssel, den der Schreibpfad kennt.
	patch := &EinstellungenPatch{}
	fuelleAlleFelder(t, reflect.ValueOf(patch).Elem())

	schreibbar := map[string]bool{}
	for _, p := range paareAusPatch(patch) {
		if schreibbar[p[0]] {
			// Zwei Paare mit demselben Schlüssel lassen das Upsert mit SQLSTATE 21000
			// auflaufen ("cannot affect row a second time"), im Betrieb und nicht im Test.
			t.Errorf("Schlüssel %q wird doppelt geschrieben", p[0])
		}
		schreibbar[p[0]] = true
	}

	// Der Lesepfad lässt sich ganz prüfen, ohne ihn abzutippen: setzeEinstellung verändert die
	// Einstellungen nur, wenn es den Schlüssel kennt.
	for schluessel := range schreibbar {
		vorher := einstellungenVorgabe()
		nachher := einstellungenVorgabe()
		wert := probewert(schluessel)
		setzeEinstellung(nachher, schluessel, &wert)
		if cmp.Equal(vorher, nachher) {
			t.Errorf("Schlüssel %q wird geschrieben, aber beim Laden nie zurückgelesen: "+
				"Das Feld speichert scheinbar und bleibt leer. "+
				"Es fehlt der case-Zweig in setzeEinstellung.", schluessel)
		}
	}

	if len(schreibbar) == 0 {
		t.Fatal("kein einziger Schlüssel gefunden: Der Test misst nichts mehr")
	}
}

// probewert liefert je Schlüssel einen Wert, der sich von der Vorgabe unterscheidet. Ein Wert,
// der der Vorgabe gleicht, machte den Test blind.
func probewert(schluessel string) string {
	switch {
	// Die Vorgabe ist false, die Regel der Arbeitshilfe; also unterscheidet nur "true".
	case schluessel == "ersatzwert_immer_kaufpreis":
		return "true"
	case strings.HasSuffix(schluessel, "_aktiv"), schluessel == "preise_erfassen":
		// Die Schalter haben verschiedene Vorgaben: "false" weicht von
		// bestellbedarf_warnung_aktiv und preise_erfassen ab, "true" vom Leseclub.
		if schluessel == "ferien_leseclub_aktiv" {
			return "true"
		}
		return "false"
	case schluessel == "lmf_stichtag":
		return "01-02"
	case schluessel == "ferien_leseclub_zieldatum":
		return "2030-01-02"
	case strings.HasPrefix(schluessel, "schule_"), strings.HasPrefix(schluessel, "etikett_eigentumsvermerk"),
		schluessel == "oeffentliche_adresse", schluessel == "alarm_empfaenger":
		return "Probewert"
	default:
		return "12345"
	}
}

// fuelleAlleFelder setzt jedes Zeigerfeld des Patches auf einen Wert. Welchen, ist gleich: Es
// geht nur darum, dass paareAusPatch keinen Schlüssel auslässt.
func fuelleAlleFelder(t *testing.T, v reflect.Value) {
	t.Helper()
	for i := range v.NumField() {
		f := v.Field(i)
		if f.Kind() != reflect.Pointer {
			t.Fatalf("Feld %q ist kein Zeiger: Im Patch heißt nil 'nicht mitgeschickt', "+
				"und ein Wertfeld kann das nicht ausdrücken", v.Type().Field(i).Name)
		}
		neu := reflect.New(f.Type().Elem())
		switch neu.Elem().Kind() {
		case reflect.String:
			neu.Elem().SetString("x")
		case reflect.Int:
			neu.Elem().SetInt(12345)
		case reflect.Bool:
			neu.Elem().SetBool(true)
		default:
			t.Fatalf("unbekannter Feldtyp %s in EinstellungenPatch", neu.Elem().Kind())
		}
		f.Set(neu)
	}
}
