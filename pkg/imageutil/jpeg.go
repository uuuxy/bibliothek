// Package imageutil wandelt Bilder in JPEG und WebP und weist ein Bild ab, dessen Pixelzahl die
// Grenze überschreitet, bevor es dekodiert wird.
package imageutil

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"

	// Cover liegen als WebP auf der Platte. Ohne diesen Decoder scheitert image.Decode an den
	// Dateien, die das Programm selbst schreibt.
	_ "golang.org/x/image/webp"
)

// JPEGQualitaet ist die Qualitätsstufe für erzeugte JPEG. Bei den hier üblichen Bildgrößen
// fällt 80 nicht auf und braucht deutlich weniger Platz als ein verlustfreies PNG.
const JPEGQualitaet = 80

// AlsJPEG wandelt ein Bild in ein Baseline-JPEG; das Paket bringt die Decoder für JPEG, PNG
// und WebP mit.
//
// Gebraucht wird es zum Einbetten in ein PDF: gofpdf kennt nur JPG, PNG und GIF, die Cover
// liegen aber als WebP vor. Das Ziel ist JPEG, weil Cover Fotos sind und als PNG ein Vielfaches
// an Platz brauchen; eine Mahnliste trägt je überfälligem Titel eines.
func AlsJPEG(daten []byte, qualitaet int) ([]byte, error) {
	// Die Datei kommt aus dem eigenen Upload-Verzeichnis, wird hier aber ungeprüft dekodiert.
	if err := PruefeBildgroesse(daten); err != nil {
		return nil, err
	}
	bild, _, err := image.Decode(bytes.NewReader(daten))
	if err != nil {
		return nil, fmt.Errorf("fehler beim Dekodieren des Bildes: %w", err)
	}
	var puffer bytes.Buffer
	if err := jpeg.Encode(&puffer, aufWeiss(bild), &jpeg.Options{Quality: qualitaet}); err != nil {
		return nil, fmt.Errorf("fehler beim JPEG-Enkodieren: %w", err)
	}
	return puffer.Bytes(), nil
}

// aufWeiss legt ein Bild mit Alphakanal auf weißen Grund. JPEG kennt keine Transparenz; ohne
// diesen Schritt würden durchsichtige Flächen schwarz, auf einem weißen Blatt ein sichtbarer
// Fehler.
func aufWeiss(bild image.Image) image.Image {
	// Ein schon deckendes Bild braucht keine Kopie. Fehlt die Methode Opaque, wird es
	// vorsichtshalber kopiert.
	if o, ok := bild.(interface{ Opaque() bool }); ok && o.Opaque() {
		return bild
	}
	grenzen := bild.Bounds()
	ziel := image.NewRGBA(grenzen)
	draw.Draw(ziel, grenzen, image.NewUniform(color.White), image.Point{}, draw.Src)
	draw.Draw(ziel, grenzen, bild, grenzen.Min, draw.Over)
	return ziel
}
