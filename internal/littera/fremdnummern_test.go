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
	nummern, mehrfach, err := LeseAusweisnummern(r)
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}

	if len(nummern) != 2 {
		t.Errorf("erwartet 2 Nummern, erhalten %d", len(nummern))
	}
	if nummern["1"] != "B97601826459" {
		t.Errorf("erwartet letzte Nummer für Leser 1 ('B97601826459'), erhalten %q", nummern["1"])
	}
	if nummern["2"] != "B97601826458" {
		t.Errorf("erwartet 'B97601826458' für Leser 2, erhalten %q", nummern["2"])
	}
	if _, ok := nummern["3"]; ok {
		t.Errorf("Leser 3 ohne FremdNummer sollte ignoriert werden")
	}

	if len(mehrfach) != 1 || mehrfach[0] != "1" {
		t.Errorf("erwartet '1' in mehrfach, erhalten %v", mehrfach)
	}
}

func TestLeseAusweisnummernLesefehler(t *testing.T) {
	r := &fehlerReader{}
	_, _, err := LeseAusweisnummern(r)
	if err == nil {
		t.Fatal("erwartet Fehler bei Lesefehler")
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
	barcodes, err := LeseFremdbarcodes(r)
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}

	if len(barcodes) != 2 {
		t.Errorf("erwartet 2 Barcodes, erhalten %d", len(barcodes))
	}
	if barcodes["1"] != "E-1234" {
		t.Errorf("erwartet 'E-1234' für Buchungsnummer 1, erhalten %q", barcodes["1"])
	}
	if barcodes["2"] != "E-5678" {
		t.Errorf("erwartet 'E-5678' für Buchungsnummer 2, erhalten %q", barcodes["2"])
	}
	if _, ok := barcodes["3"]; ok {
		t.Errorf("Buchungsnummer 3 ohne Exemplarnummer sollte ignoriert werden")
	}
}

func TestLeseFremdbarcodesLesefehler(t *testing.T) {
	r := &fehlerReader{}
	_, err := LeseFremdbarcodes(r)
	if err == nil {
		t.Fatal("erwartet Fehler bei Lesefehler")
	}
}

type fehlerReader struct{}

func (f *fehlerReader) Read(p []byte) (n int, err error) {
	return 0, errors.New("simulierter Lesefehler")
}
