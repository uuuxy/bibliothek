// Package strichcode erzeugt den Strichcode eines Ausweises, Etiketts oder Briefs als Bild.
package strichcode

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"unicode/utf8"

	"github.com/boombuler/barcode"
	"github.com/boombuler/barcode/code128"
	"github.com/boombuler/barcode/qr"
)

// maxSeite ist die größte Breite und Höhe eines Bilds in Pixeln. Die Oberfläche bittet höchstens
// um einen Strichcode über die ganze Ausweiskarte in 300 dpi (1011 × 638); ohne Grenze legte eine
// Anfrage mit 100000 × 100000 ein Bild von 40 GB an, und der Prozess endete am Speichermangel.
const maxSeite = 2000

// maxInhalt ist die längste Nummer in einem Strichcode, die Grenze von Code 128 in
// github.com/boombuler/barcode/code128.
const maxInhalt = 80

// ErrZuGross meldet einen Inhalt über maxInhalt oder ein Bild über maxSeite.
var ErrZuGross = errors.New("strichcode zu groß")

// PNG liefert den Strichcode als PNG in der gewünschten Größe: Code 128, oder QR, wenn alsQR
// gesetzt ist.
//
// Code 128 trägt kein Prüfzeichen im Text, braucht für dieselbe Nummer rund die halbe Breite
// von Code 39 und kennt Klein- wie Großbuchstaben: Der Aufdruck ist Zeichen für Zeichen, was
// in der Datenbank steht. Ältere Aufdrucke in Code 39 tragen ein Prüfzeichen in den Daten;
// der Scan schlägt sie als zweiten Versuch ohne dieses Zeichen nach (internal/pkg/code39).
func PNG(inhalt string, alsQR bool, breite, hoehe int) ([]byte, error) {
	code, err := kodiere(inhalt, alsQR)
	if err != nil {
		return nil, err
	}
	return alsPNG(code, breite, hoehe)
}

// strichPixelAufBlatt sind die Pixel, die ein Strich im Bild für ein Blatt mindestens bekommt.
const strichPixelAufBlatt = 2

// PNGFuerBlatt liefert den Strichcode (Code 128) für ein Feld fester Breite auf einem Blatt.
// Solange jeder Strich in breite Pixeln zwei Pixel bekommt, ist es das Bild von PNG. Eine längere
// Nummer füllt das Bild ohne Rand mit zwei Pixeln je Strich: Gedruckt wird der Strich so breit,
// wie das Feld es zulässt, statt auf die Hälfte zu fallen. Die Ruhezone lässt dann der Rand um
// das Feld; ein Etikett, dessen Kante das Feld begrenzt, nimmt deshalb PNG.
func PNGFuerBlatt(inhalt string, breite, hoehe int) ([]byte, error) {
	code, err := kodiere(inhalt, false)
	if err != nil {
		return nil, err
	}
	if fuellend := strichPixelAufBlatt * code.Bounds().Dx(); breite < fuellend {
		breite = fuellend
	}
	return alsPNG(code, breite, hoehe)
}

// kodiere liefert die Striche des Inhalts in ihrer kleinsten Breite: Code 128, oder QR.
func kodiere(inhalt string, alsQR bool) (barcode.Barcode, error) {
	if n := utf8.RuneCountInString(inhalt); n > maxInhalt {
		return nil, fmt.Errorf("%w: %d zeichen, höchstens %d", ErrZuGross, n, maxInhalt)
	}

	var code barcode.Barcode
	var err error

	if alsQR {
		code, err = qr.Encode(inhalt, qr.M, qr.Auto)
	} else {
		code, err = code128.Encode(inhalt)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to encode barcode: %w", err)
	}
	return code, nil
}

// alsPNG skaliert die Striche auf die gewünschte Größe und liefert sie als PNG.
func alsPNG(code barcode.Barcode, breite, hoehe int) ([]byte, error) {
	// barcode.Scale verkleinert nicht unter die Breite der Striche selbst, und ein langer
	// Inhalt ist breiter als die üblichen 200 Pixel der Ausweiskarte. Das Bild wird dann
	// größer geliefert, statt die Anfrage scheitern zu lassen.
	if minBreite := code.Bounds().Dx(); breite < minBreite {
		breite = minBreite
	}
	if minHoehe := code.Bounds().Dy(); hoehe < minHoehe {
		hoehe = minHoehe
	}
	if breite > maxSeite || hoehe > maxSeite {
		return nil, fmt.Errorf("%w: %d × %d pixel, höchstens %d je seite", ErrZuGross, breite, hoehe, maxSeite)
	}

	skaliert, err := barcode.Scale(code, breite, hoehe)
	if err != nil {
		return nil, fmt.Errorf("failed to scale barcode: %w", err)
	}

	// Als RGBA mit 8 Bit je Kanal, weil gofpdf kein PNG mit 16 Bit Tiefe liest.
	grenzen := skaliert.Bounds()
	bild := image.NewRGBA(grenzen)
	draw.Draw(bild, grenzen, skaliert, grenzen.Min, draw.Src)

	var puffer bytes.Buffer
	if err := png.Encode(&puffer, bild); err != nil {
		return nil, fmt.Errorf("failed to encode PNG: %w", err)
	}

	return puffer.Bytes(), nil
}
