package repository

import (
	"reflect"
	"strings"
	"testing"
)

// Die blinde Hälfte des Audit-Rasters, als Gate.
//
// Ein Einstellungsfeld hat ZWEI Enden: pairsAusPatch schreibt es, applyEinstellung
// liest es zurück. Fehlt das zweite, speichert das Feld scheinbar erfolgreich und ist
// beim nächsten Laden für immer leer — der Payload↔Struct-Vergleich sieht das nicht,
// weil dort alles stimmt. Genau so verschwand am 17.08.2026 alarm_empfaenger, und der
// Kritisch-Alarm fiel still auf „alle aktiven Admins" zurück.
//
// Der Roundtrip-Test daneben beweist dasselbe für DREI Felder am echten Postgres.
// Dieser hier beweist es für ALLE — ohne Datenbank, in Millisekunden, und er meldet
// sich von selbst, sobald jemand ein Feld hinzufügt und das andere Ende vergisst.
func TestEinstellungen_SchreibpfadUndLesepfadSindDeckungsgleich(t *testing.T) {
	// Ein Patch, in dem JEDES Feld gesetzt ist: Nur dann liefert pairsAusPatch auch
	// jeden Schlüssel, den der Schreibpfad überhaupt kennt.
	patch := &EinstellungenPatch{}
	fuelleAlleFelder(t, reflect.ValueOf(patch).Elem())

	schreibbar := map[string]bool{}
	for _, p := range pairsAusPatch(patch) {
		if schreibbar[p[0]] {
			// Zwei Paare mit demselben Schlüssel lassen das Upsert mit
			// SQLSTATE 21000 auflaufen ("cannot affect row a second time") —
			// im Betrieb, nicht im Test.
			t.Errorf("Schlüssel %q wird doppelt geschrieben", p[0])
		}
		schreibbar[p[0]] = true
	}

	// Der Lesepfad ist erschöpfend prüfbar, ohne ihn abzutippen: applyEinstellung
	// verändert die Struktur nur, wenn es den Schlüssel kennt.
	for key := range schreibbar {
		vorher := standardEinstellungen()
		nachher := standardEinstellungen()
		wert := probewert(key)
		applyEinstellung(nachher, key, &wert)
		if reflect.DeepEqual(vorher, nachher) {
			t.Errorf("Schlüssel %q wird geschrieben, aber von GetSettings nie zurückgelesen "+
				"— das Feld speichert scheinbar und bleibt für immer leer. "+
				"Fehlender case-Zweig in applyEinstellung.", key)
		}
	}

	if len(schreibbar) == 0 {
		t.Fatal("kein einziger Schlüssel gefunden — der Test misst nichts mehr")
	}
}

// probewert liefert je Schlüssel einen Wert, der sich von der Vorgabe unterscheidet.
// Ein Wert, der zufällig der Vorgabe entspricht, würde den Test blind machen.
func probewert(key string) string {
	switch {
	case strings.HasSuffix(key, "_aktiv"), key == "preise_erfassen":
		// Beide Schalter haben unterschiedliche Vorgaben; "false" weicht von
		// bestellbedarf_warnung_aktiv und preise_erfassen ab, "true" von leseclub.
		if key == "ferien_leseclub_aktiv" {
			return "true"
		}
		return "false"
	case key == "lmf_stichtag":
		return "01-02"
	case key == "ferien_leseclub_zieldatum":
		return "2030-01-02"
	case strings.HasPrefix(key, "schule_"), key == "etikett_eigentumsvermerk",
		key == "oeffentliche_adresse", key == "alarm_empfaenger":
		return "Probewert"
	default:
		return "12345"
	}
}

// fuelleAlleFelder setzt jedes Zeigerfeld des Patches auf einen Wert — welchen, ist
// gleichgültig; es geht nur darum, dass pairsAusPatch keinen Schlüssel auslässt.
func fuelleAlleFelder(t *testing.T, v reflect.Value) {
	t.Helper()
	for i := range v.NumField() {
		f := v.Field(i)
		if f.Kind() != reflect.Pointer {
			t.Fatalf("Feld %q ist kein Zeiger — im Patch heisst nil 'nicht mitgeschickt', "+
				"und ein Wertfeld kann das nicht ausdruecken", v.Type().Field(i).Name)
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
