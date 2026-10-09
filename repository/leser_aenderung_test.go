package repository

import (
	"reflect"
	"testing"
	"time"
)

// alleFelderDerAenderung füllt jedes Feld von LeserAenderung über Reflexion: Ein neues Feld
// ist damit ohne weiteres Zutun mitgeprüft.
func alleFelderDerAenderung(t *testing.T) LeserAenderung {
	t.Helper()
	var a LeserAenderung
	v := reflect.ValueOf(&a).Elem()
	for i := 0; i < v.NumField(); i++ {
		feld := v.Type().Field(i)
		if feld.Type.Kind() != reflect.Pointer {
			t.Fatalf("Feld %s ist kein Zeiger: nil unterscheidet „nicht genannt“ von „geleert“", feld.Name)
		}
		zeiger := reflect.New(feld.Type.Elem())
		switch feld.Type.Elem() {
		case reflect.TypeFor[string]():
			zeiger.Elem().SetString("wert-" + feld.Name)
		case reflect.TypeFor[int]():
			zeiger.Elem().SetInt(2031)
		case reflect.TypeFor[time.Time]():
			zeiger.Elem().Set(reflect.ValueOf(time.Date(2012, 3, 4, 0, 0, 0, 0, time.UTC)))
		default:
			t.Fatalf("Feld %s hat den Typ %s, für den dieser Test keinen Wert kennt", feld.Name, feld.Type.Elem())
		}
		v.Field(i).Set(zeiger)
	}
	return a
}

// Ein Feld der Änderung, das zuweisungen nicht kennt, nähme die Tür an und schriebe es nie.
func TestLeserAenderung_JedesFeldSetztEineSpalte(t *testing.T) {
	a := alleFelderDerAenderung(t)
	felder := reflect.TypeFor[LeserAenderung]().NumField()
	if felder < 10 {
		t.Fatalf("nur %d Felder gefunden, die Reflexion misst offenbar nichts", felder)
	}
	spalten := a.Spalten()
	if len(spalten) != felder {
		t.Fatalf("LeserAenderung hat %d Felder, die Anweisung setzt %d Spalten: %v", felder, len(spalten), spalten)
	}
	gesehen := map[string]bool{}
	for _, s := range spalten {
		if gesehen[s] {
			t.Errorf("die Spalte %s steht zweimal in der Anweisung", s)
		}
		gesehen[s] = true
	}
}

// Die Anweisung nennt jede Spalte mit ihrem eigenen Feld, die Kennung steht zuletzt.
func TestLeserAenderung_AnweisungUndWerte(t *testing.T) {
	a := alleFelderDerAenderung(t)
	query, args := a.anweisung("die-kennung")

	const soll = "UPDATE leser SET aktualisiert_am = CURRENT_TIMESTAMP" +
		", vorname = $1, nachname = $2, barcode_id = $3, klasse = $4, abgaenger_jahr = $5" +
		", geburtsdatum = $6, strasse = $7, hausnummer = $8, plz = $9, ort = $10" +
		", eltern_email = $11, lusd_id = $12, art = $13 WHERE id = $14"
	if query != soll {
		t.Errorf("Anweisung:\n ist  %s\n soll %s", query, soll)
	}
	sollWerte := []any{"wert-Vorname", "wert-Nachname", "wert-Ausweisnummer", "wert-Klasse", 2031,
		a.Geburtsdatum, "wert-Strasse", "wert-Hausnummer", "wert-Plz", "wert-Ort",
		"wert-ElternEmail", "wert-LusdID", "wert-Art", "die-kennung"}
	if !reflect.DeepEqual(args, sollWerte) {
		t.Errorf("Werte:\n ist  %#v\n soll %#v", args, sollWerte)
	}
}

// Ohne genanntes Feld setzt die Anweisung nur den Zeitpunkt; ein einzelnes Feld trägt $1.
func TestLeserAenderung_WenigeFelder(t *testing.T) {
	var keine LeserAenderung
	if !keine.Leer() {
		t.Error("eine Änderung ohne Feld gilt nicht als leer")
	}
	query, args := keine.anweisung("k")
	if query != "UPDATE leser SET aktualisiert_am = CURRENT_TIMESTAMP WHERE id = $1" || !reflect.DeepEqual(args, []any{"k"}) {
		t.Errorf("leere Änderung: %s %#v", query, args)
	}

	ort := "Ortsname"
	eine := LeserAenderung{Ort: &ort}
	if eine.Leer() {
		t.Error("eine Änderung mit einem Feld gilt als leer")
	}
	query, args = eine.anweisung("k")
	if query != "UPDATE leser SET aktualisiert_am = CURRENT_TIMESTAMP, ort = $1 WHERE id = $2" ||
		!reflect.DeepEqual(args, []any{"Ortsname", "k"}) {
		t.Errorf("Änderung mit einem Feld: %s %#v", query, args)
	}
}

// Ausweisnummer, Anschrift und Elternkontakt werden leer zu NULL; ein Name bleibt, was er ist.
func TestLeserAenderung_LeererWert(t *testing.T) {
	leer, raum := "", "  \t"
	a := LeserAenderung{
		Vorname: &leer, Ausweisnummer: &leer, Strasse: &raum, Hausnummer: &leer,
		Plz: &raum, Ort: &leer, ElternEmail: &raum,
	}
	_, args := a.anweisung("k")
	soll := []any{"", nil, nil, nil, nil, nil, nil, "k"}
	if !reflect.DeepEqual(args, soll) {
		t.Errorf("Werte:\n ist  %#v\n soll %#v", args, soll)
	}
}
