package api

import (
	"bytes"
	"image/png"
	"testing"
)

// Lange Inhalte sprengen die native Code39-Modulbreite der üblichen
// 200px-Ausweisbreite — statt eines Fehlers ("can not scale barcode to an
// image smaller than ...") muss das PNG dann eben breiter ausfallen.
func TestGenerateBarcodePNG_LangerInhaltUnter200pxFaelltNichtAus(t *testing.T) {
	data, err := GenerateBarcodePNG("S-GESPERRT-MRE1234567", false, 200, 50)
	if err != nil {
		t.Fatalf("langer Inhalt bei width=200 darf nicht scheitern: %v", err)
	}

	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("Ausgabe ist kein valides PNG: %v", err)
	}
	if img.Bounds().Dx() < 200 {
		t.Errorf("Bild muss mindestens native Breite haben, hat %dpx", img.Bounds().Dx())
	}
}

// Kurze Inhalte respektieren weiterhin die Wunschgröße.
func TestGenerateBarcodePNG_KurzerInhaltBehaeltWunschbreite(t *testing.T) {
	data, err := GenerateBarcodePNG("B-1", false, 300, 100)
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("Ausgabe ist kein valides PNG: %v", err)
	}
	if img.Bounds().Dx() != 300 || img.Bounds().Dy() != 100 {
		t.Errorf("erwartet 300x100, bekam %dx%d", img.Bounds().Dx(), img.Bounds().Dy())
	}
}
