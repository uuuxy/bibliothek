// Package strichcode erzeugt den Strichcode eines Ausweises, Etiketts oder Briefs als Bild.
package strichcode

import (
	"bytes"
	"fmt"
	"image"
	"image/draw"
	"image/png"

	"github.com/boombuler/barcode"
	"github.com/boombuler/barcode/code128"
	"github.com/boombuler/barcode/qr"
)

// PNG liefert den Strichcode als PNG in der gewünschten Größe: Code 128, oder QR, wenn alsQR
// gesetzt ist.
//
// Code 128 trägt kein Prüfzeichen im Text, braucht für dieselbe Nummer rund die halbe Breite
// von Code 39 und kennt Klein- wie Großbuchstaben: Der Aufdruck ist Zeichen für Zeichen, was
// in der Datenbank steht. Ältere Aufdrucke in Code 39 tragen ein Prüfzeichen in den Daten;
// der Scan schlägt sie als zweiten Versuch ohne dieses Zeichen nach (pkg/code39).
func PNG(inhalt string, alsQR bool, breite, hoehe int) ([]byte, error) {
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

	// barcode.Scale verkleinert nicht unter die Breite der Striche selbst, und ein langer
	// Inhalt ist breiter als die üblichen 200 Pixel der Ausweiskarte. Das Bild wird dann
	// größer geliefert, statt die Anfrage scheitern zu lassen.
	if minBreite := code.Bounds().Dx(); breite < minBreite {
		breite = minBreite
	}
	if minHoehe := code.Bounds().Dy(); hoehe < minHoehe {
		hoehe = minHoehe
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
