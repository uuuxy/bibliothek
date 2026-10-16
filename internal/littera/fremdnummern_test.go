package littera

import (
	"errors"
	"strings"
	"testing"
)

func TestLeseAusweisnummern(t *testing.T) {
	csv := `Leser,FremdNummer
1,B97601826457
2,B97601826458
1,B97601826459
,B97601826460
3,
`
	r := strings.NewReader(csv)
	nummern, mehrfach, err := leseAusweisnummern(r)
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}

	if len(nummern) != 2 {
		t.Errorf("leseAusweisnummern(…) = %d Nummern, erwartet 2", len(nummern))
	}
	if nummern["1"] != "B97601826459" {
		t.Errorf("leseAusweisnummern(…)[\"1\"] = %q, erwartet die letzte Nummer 'B97601826459'", nummern["1"])
	}
	if nummern["2"] != "B97601826458" {
		t.Errorf("leseAusweisnummern(…)[\"2\"] = %q, erwartet 'B97601826458'", nummern["2"])
	}
	if _, ok := nummern["3"]; ok {
		t.Errorf("Leser 3 ohne FremdNummer sollte ignoriert werden")
	}

	if len(mehrfach) != 1 || mehrfach[0] != "1" {
		t.Errorf("leseAusweisnummern(…): mehrfach = %v, erwartet [1]", mehrfach)
	}
}

func TestLeseAusweisnummernLesefehler(t *testing.T) {
	r := &fehlerReader{}
	_, _, err := leseAusweisnummern(r)
	if err == nil {
		t.Fatal("leseAusweisnummern(fehlerReader) = nil, erwartet einen Fehler")
	}
}

func TestLeseFremdbarcodes(t *testing.T) {
	csv := `Buchungsnummer,Exemplarnummer
1,E-1234
2,E-5678
,E-9999
3,
`
	r := strings.NewReader(csv)
	barcodes, err := leseFremdbarcodes(r)
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}

	if len(barcodes) != 2 {
		t.Errorf("leseFremdbarcodes(…) = %d Barcodes, erwartet 2", len(barcodes))
	}
	if barcodes["1"] != "E-1234" {
		t.Errorf("leseFremdbarcodes(…)[\"1\"] = %q, erwartet 'E-1234'", barcodes["1"])
	}
	if barcodes["2"] != "E-5678" {
		t.Errorf("leseFremdbarcodes(…)[\"2\"] = %q, erwartet 'E-5678'", barcodes["2"])
	}
	if _, ok := barcodes["3"]; ok {
		t.Errorf("Buchungsnummer 3 ohne Exemplarnummer sollte ignoriert werden")
	}
}

func TestLeseFremdbarcodesLesefehler(t *testing.T) {
	r := &fehlerReader{}
	_, err := leseFremdbarcodes(r)
	if err == nil {
		t.Fatal("leseFremdbarcodes(fehlerReader) = nil, erwartet einen Fehler")
	}
}

type fehlerReader struct{}

func (f *fehlerReader) Read(p []byte) (n int, err error) {
	return 0, errors.New("simulierter Lesefehler")
}
