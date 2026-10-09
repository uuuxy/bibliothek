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

// PNG liefert den Strichcode als PNG in der gewünschten Größe: Code 128, oder QR, wenn isQR
// gesetzt ist.
//
// Code 128 trägt kein Prüfzeichen im Text, braucht für dieselbe Nummer rund die halbe Breite
// von Code 39 und kennt Klein- wie Großbuchstaben: Der Aufdruck ist Zeichen für Zeichen, was
// in der Datenbank steht. Ältere Aufdrucke in Code 39 tragen ein Prüfzeichen in den Daten;
// der Scan schlägt sie als zweiten Versuch ohne dieses Zeichen nach (pkg/code39).
func PNG(content string, isQR bool, width, height int) ([]byte, error) {
	var bc barcode.Barcode
	var err error

	if isQR {
		bc, err = qr.Encode(content, qr.M, qr.Auto)
	} else {
		bc, err = code128.Encode(content)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to encode barcode: %w", err)
	}

	// barcode.Scale verweigert Verkleinern unter die native Modulgröße
	// (lange Inhalte sprengen z. B. die üblichen 200px der Ausweiskarten).
	// Dann lieber größer ausliefern als mit 500 scheitern.
	if minW := bc.Bounds().Dx(); width < minW {
		width = minW
	}
	if minH := bc.Bounds().Dy(); height < minH {
		height = minH
	}

	scaled, err := barcode.Scale(bc, width, height)
	if err != nil {
		return nil, fmt.Errorf("failed to scale barcode: %w", err)
	}

	// Convert the scaled barcode to standard 8-bit RGBA image
	// to avoid 16-bit PNG depth which gofpdf PNG parser doesn't support.
	bounds := scaled.Bounds()
	rgbaImg := image.NewRGBA(bounds)
	draw.Draw(rgbaImg, bounds, scaled, bounds.Min, draw.Src)

	var buf bytes.Buffer
	if err := png.Encode(&buf, rgbaImg); err != nil {
		return nil, fmt.Errorf("failed to encode PNG: %w", err)
	}

	return buf.Bytes(), nil
}
