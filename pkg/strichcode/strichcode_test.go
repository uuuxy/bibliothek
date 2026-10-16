package strichcode

import (
	"bytes"
	"image/png"
	"testing"
)

// Ein langer Inhalt ist breiter als die üblichen 200 Pixel der Ausweiskarte. Statt des Fehlers
// der Bibliothek („can not scale barcode to an image smaller than …") kommt das PNG dann
// breiter zurück.
func TestPNG_LangerInhaltUnter200pxFaelltNichtAus(t *testing.T) {
	daten, err := PNG("S-GESPERRT-MRE1234567", false, 200, 50)
	if err != nil {
		t.Fatalf("langer Inhalt bei Breite 200 darf nicht scheitern: %v", err)
	}

	bild, err := png.Decode(bytes.NewReader(daten))
	if err != nil {
		t.Fatalf("Ausgabe ist kein gültiges PNG: %v", err)
	}
	if bild.Bounds().Dx() < 200 {
		t.Errorf("Bild muss mindestens so breit sein wie die Striche, hat %dpx", bild.Bounds().Dx())
	}
}

// Ein kurzer Inhalt behält die gewünschte Größe.
func TestPNG_KurzerInhaltBehaeltWunschbreite(t *testing.T) {
	daten, err := PNG("B-1", false, 300, 100)
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}
	bild, err := png.Decode(bytes.NewReader(daten))
	if err != nil {
		t.Fatalf("Ausgabe ist kein gültiges PNG: %v", err)
	}
	if bild.Bounds().Dx() != 300 || bild.Bounds().Dy() != 100 {
		t.Errorf("Bild = %dx%d, erwartet 300x100", bild.Bounds().Dx(), bild.Bounds().Dy())
	}
}
