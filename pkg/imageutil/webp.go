package imageutil

import (
	"bytes"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"

	"github.com/chai2010/webp"
)

// AlsWebP wandelt ein Bild in WebP mit der genannten Qualität.
func AlsWebP(daten []byte, qualitaet float32) ([]byte, error) {
	if err := PruefeBildgroesse(daten); err != nil {
		return nil, err
	}
	bild, _, err := image.Decode(bytes.NewReader(daten))
	if err != nil {
		return nil, fmt.Errorf("fehler beim Dekodieren des Bildes (muss JPG oder PNG sein): %w", err)
	}
	return ErzeugeWebP(bild, qualitaet)
}

// ErzeugeWebP liefert ein schon dekodiertes Bild als WebP, verlustbehaftet in der genannten
// Qualität.
func ErzeugeWebP(bild image.Image, qualitaet float32) ([]byte, error) {
	var puffer bytes.Buffer
	if err := webp.Encode(&puffer, bild, &webp.Options{Lossless: false, Quality: qualitaet}); err != nil {
		return nil, fmt.Errorf("fehler beim WebP-Enkodieren: %w", err)
	}
	return puffer.Bytes(), nil
}
