package betrag

import (
	"encoding/json"
	"testing"
)

func TestCent_Text(t *testing.T) {
	for _, f := range []struct {
		cent Cent
		want string
	}{
		{1250, "12,50"},
		{0, "0,00"},
		{1990, "19,90"},
		{123450, "1234,50"},
		{-320, "-3,20"},
		{5, "0,05"},
		{-5, "-0,05"},
	} {
		if got := f.cent.Text(); got != f.want {
			t.Errorf("Cent(%d).Text() = %q, erwartet %q", f.cent, got, f.want)
		}
	}
	if got := Cent(4150).Euro(); got != "41,50 €" {
		t.Errorf("Cent(4150).Euro() = %q, erwartet „41,50 €“", got)
	}
}

func TestCent_Mal(t *testing.T) {
	// 3 × 0,35 € ist in float64 1,0499999999999998; in Cent 105.
	if got := Cent(35).Mal(3); got != 105 {
		t.Errorf("Cent(35).Mal(3) = %d, erwartet 105", got)
	}
}

func TestLeseEuro(t *testing.T) {
	for _, f := range []struct {
		text string
		want Cent
	}{
		{"12.5", 1250},
		{"12,50", 1250},
		{"12", 1200},
		{"0.1", 10},
		{"-3.2", -320},
		{" 7.00 ", 700},
		{"1e2", 10000},
		// Wie NUMERIC(10,2) beim Schreiben: kaufmännisch auf den Cent, vom Nullpunkt weg.
		{"12.345", 1235},
		{"12.344", 1234},
		{"-0.005", -1},
		{"0.004", 0},
	} {
		got, err := LeseEuro(f.text)
		if err != nil || got != f.want {
			t.Errorf("LeseEuro(%q) = %d, %v, erwartet %d", f.text, got, err, f.want)
		}
	}
	for _, text := range []string{"abc", "", "1,2,3", "1e30"} {
		if got, err := LeseEuro(text); err == nil {
			t.Errorf("LeseEuro(%q) = %d ohne Fehler, erwartet einen Fehler", text, got)
		}
	}
}

func TestCent_JSON(t *testing.T) {
	roh, err := json.Marshal(struct {
		Betrag Cent  `json:"betrag"`
		Leer   *Cent `json:"leer"`
	}{Betrag: 1250})
	if err != nil || string(roh) != `{"betrag":12.50,"leer":null}` {
		t.Errorf("json.Marshal = %s, %v", roh, err)
	}

	var gelesen struct {
		Betrag Cent  `json:"betrag"`
		Leer   *Cent `json:"leer"`
		Fehlt  Cent  `json:"fehlt"`
	}
	gelesen.Fehlt = 99
	if err := json.Unmarshal([]byte(`{"betrag":12.5,"leer":null,"fehlt":null}`), &gelesen); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	if gelesen.Betrag != 1250 || gelesen.Leer != nil || gelesen.Fehlt != 99 {
		t.Errorf("gelesen %+v, erwartet 1250, nil und 99 unverändert", gelesen)
	}
	if err := json.Unmarshal([]byte(`{"betrag":"12,50"}`), &gelesen); err == nil {
		t.Error("ein Betrag als Text wurde angenommen; die Oberfläche schickt Zahlen")
	}
}

func TestCent_ScanUndValue(t *testing.T) {
	for _, f := range []struct {
		quelle any
		want   Cent
	}{
		{"12.50", 1250},
		{[]byte("0.05"), 5},
		{int64(3), 300},
	} {
		var c Cent
		if err := c.Scan(f.quelle); err != nil || c != f.want {
			t.Errorf("Scan(%#v) = %d, %v, erwartet %d", f.quelle, c, err, f.want)
		}
	}
	for _, quelle := range []any{nil, 12.5, "zwölf"} {
		var c Cent
		if err := c.Scan(quelle); err == nil {
			t.Errorf("Scan(%#v) ohne Fehler, erwartet einen Fehler", quelle)
		}
	}
	if wert, err := Cent(-1250).Value(); err != nil || wert != "-12.50" {
		t.Errorf("Value() = %v, %v, erwartet „-12.50“", wert, err)
	}
}
