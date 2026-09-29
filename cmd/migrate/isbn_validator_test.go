package main

import "testing"

// TestValidateBarcode: insertExemplare prüft jeden erzeugten Barcode gegen die Form
// B-<Ziffern>, bevor er in die Datenbank geht. highestBarcodeSeq zählt beim nächsten Lauf
// nur Barcodes dieser Form (bis neun Ziffern); einen anders geschriebenen sähe der Zähler
// nicht, und die Übernahme erzeugte ihn noch einmal.
func TestValidateBarcode(t *testing.T) {
	faelle := []struct {
		name     string
		barcode  string
		erwartet bool
	}{
		{"üblich", "B-12345", true},
		{"eine Ziffer", "B-1", true},
		{"neun Ziffern", "B-999999999", true},
		{"führende Nullen", "B-00123", true},
		{"leer", "", false},
		{"ohne Ziffern", "B-", false},
		{"Buchstabe in der Nummer", "B-123A", false},
		{"kleines b", "b-123", false},
		{"ohne Vorsilbe", "12345", false},
		{"nur Strich", "-", false},
		{"Leerzeichen", "B- 123", false},
		{"Zeichen davor", "AB-123", false},
	}

	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			if got := validateBarcode(f.barcode); got != f.erwartet {
				t.Errorf("validateBarcode(%q) = %v, erwartet %v", f.barcode, got, f.erwartet)
			}
		})
	}
}
