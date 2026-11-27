package pdf

import (
	"strings"
	"testing"
	"time"
)

// Die Ersatzforderung an einen minderjährigen Schüler geht an die Erziehungsberechtigten: Über
// dem Namen steht die Zeile des Bescheids, passend zur Anrede. Einen volljährigen spricht sie
// selbst an.
func TestRechnungAnschriftAnDieErziehungsberechtigten(t *testing.T) {
	positionen := []RechnungsPosition{{Titel: "Deutschbuch 7", Barcode: "B-1", Ausleihdatum: time.Now(), Ersatzpreis: 1990, Land: true}}
	for _, volljaehrig := range []bool{false, true} {
		s := Schueler{Vorname: "Lena", Nachname: "Land", Strasse: "Blumenweg", Hausnummer: "7",
			PLZ: "61169", Ort: "Friedberg", Volljaehrig: volljaehrig}
		got, err := ErzeugeRechnung(s, positionen, testSchule(), testZahlung())
		if err != nil {
			t.Fatalf("ErzeugeRechnung: %v", err)
		}
		blatt := blattText(t, got)
		if mitZeile := strings.Contains(blatt, "Erziehungsberechtigten des"); mitZeile == volljaehrig {
			t.Errorf("volljährig=%v: Zeile an die Erziehungsberechtigten im Fensterfeld = %v:\n%s", volljaehrig, mitZeile, blatt)
		}
		if !strings.Contains(blatt, "Blumenweg 7") || !strings.Contains(blatt, "61169 Friedberg") {
			t.Errorf("volljährig=%v: die Anschrift fehlt:\n%s", volljaehrig, blatt)
		}
	}
}

// Fehlt die Straße, steht der Vermerk im Fensterfeld wie bei Mahnbrief und Bescheid.
func TestRechnungHalbeAnschriftNenntDenVermerk(t *testing.T) {
	positionen := []RechnungsPosition{{Titel: "Deutschbuch 7", Barcode: "B-1", Ausleihdatum: time.Now(), Ersatzpreis: 1990, Land: true}}
	got, err := ErzeugeRechnung(Schueler{Vorname: "Lena", Nachname: "Land", PLZ: "61169", Ort: "Friedberg"},
		positionen, testSchule(), testZahlung())
	if err != nil {
		t.Fatalf("ErzeugeRechnung: %v", err)
	}
	blatt := blattText(t, got)
	if !strings.Contains(blatt, "Keine Adresse hinterlegt") || strings.Contains(blatt, "61169 Friedberg") {
		t.Errorf("halbe Anschrift ohne Vermerk:\n%s", blatt)
	}
}
