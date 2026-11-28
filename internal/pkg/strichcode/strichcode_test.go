package strichcode

import (
	"bytes"
	"errors"
	"image/png"
	"strings"
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

// Das Bild hat eine Grenze: Ohne sie legte eine Anfrage an /api/barcode mit 100000 × 100000 ein
// Bild von 40 GB an, und der Prozess endete am Speichermangel. Der Fehler kommt vor dem Anlegen.
func TestPNG_UebergrossesBildWirdAbgewiesen(t *testing.T) {
	for name, fall := range map[string]struct {
		breite, hoehe int
		qr            bool
	}{
		"zu breit":   {maxSeite + 1, 100, false},
		"zu hoch":    {300, maxSeite + 1, false},
		"QR zu groß": {maxSeite + 1, maxSeite + 1, true},
	} {
		if _, err := PNG("B-10001", fall.qr, fall.breite, fall.hoehe); !errors.Is(err, ErrZuGross) {
			t.Errorf("%s: Fehler %v, erwartet ErrZuGross", name, err)
		}
	}
	if _, err := PNG("B-10001", false, maxSeite, maxSeite); err != nil {
		t.Errorf("Bild genau an der Grenze: %v", err)
	}
}

// Ein Inhalt über maxInhalt ist ein Fehler der Anfrage, kein Fehler des Servers.
func TestPNG_ZuLangerInhaltIstErrZuGross(t *testing.T) {
	if _, err := PNG(strings.Repeat("9", maxInhalt+1), true, 200, 200); !errors.Is(err, ErrZuGross) {
		t.Errorf("Fehler %v, erwartet ErrZuGross", err)
	}
	if _, err := PNG(strings.Repeat("9", maxInhalt), false, 300, 100); err != nil {
		t.Errorf("Inhalt genau an der Grenze: %v", err)
	}
}

// messeStriche liest die mittlere Pixelzeile eines Strichcodes: die Breite des Bilds, den
// schmalsten Strich und den hellen Rand links und rechts.
func messeStriche(t *testing.T, daten []byte) (breite, schmalster, randLinks, randRechts int) {
	t.Helper()
	bild, err := png.Decode(bytes.NewReader(daten))
	if err != nil {
		t.Fatalf("Ausgabe ist kein gültiges PNG: %v", err)
	}
	grenzen := bild.Bounds()
	y := grenzen.Min.Y + grenzen.Dy()/2
	erster, letzter, lauf := -1, -1, 0
	schmalster = grenzen.Dx()
	for x := grenzen.Min.X; x < grenzen.Max.X; x++ {
		rot, _, _, _ := bild.At(x, y).RGBA()
		if rot < 0x8000 {
			if erster < 0 {
				erster = x - grenzen.Min.X
			}
			letzter = x - grenzen.Min.X
			lauf++
			continue
		}
		if lauf > 0 {
			schmalster = min(schmalster, lauf)
		}
		lauf = 0
	}
	return grenzen.Dx(), schmalster, erster, grenzen.Dx() - 1 - letzter
}

// Für ein Feld auf einem Blatt bleibt eine Nummer, deren Striche in der Wunschbreite zwei Pixel
// bekommen, das Bild von PNG. Eine längere füllt das Bild ohne Rand mit zwei Pixeln je Strich; in
// PNG fiele sie auf einen Pixel und gedruckt auf die halbe Strichbreite.
func TestPNGFuerBlatt_LangeNummerFuelltDasBild(t *testing.T) {
	kurz, err := PNGFuerBlatt("0000000123456", 300, 80)
	if err != nil {
		t.Fatalf("PNGFuerBlatt(13 Ziffern): %v", err)
	}
	wiePNG, err := PNG("0000000123456", false, 300, 80)
	if err != nil {
		t.Fatalf("PNG(13 Ziffern): %v", err)
	}
	if !bytes.Equal(kurz, wiePNG) {
		t.Error("PNGFuerBlatt(13 Ziffern, 300) weicht vom Bild aus PNG ab, erwartet dasselbe Bild")
	}

	lang, err := PNGFuerBlatt("ABCDEFGHIJKLM", 300, 80)
	if err != nil {
		t.Fatalf("PNGFuerBlatt(13 Buchstaben): %v", err)
	}
	breite, schmalster, links, rechts := messeStriche(t, lang)
	if breite != 356 || schmalster != 2 || links != 0 || rechts != 0 {
		t.Errorf("PNGFuerBlatt(13 Buchstaben, 300) = %d Pixel breit, schmalster Strich %d, Rand %d und %d, erwartet 356, 2, 0 und 0",
			breite, schmalster, links, rechts)
	}
}
