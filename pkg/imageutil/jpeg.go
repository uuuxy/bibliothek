package imageutil

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"

	// Registriert den WebP-Decoder prozessweit. WebP ist unser Speicherformat für Cover
	// (siehe inventur/cover_storage.go), aber weder image/jpeg noch image/png kennen es —
	// ohne diesen Import scheitert image.Decode auf genau den Dateien, die wir selbst schreiben.
	_ "golang.org/x/image/webp"
)

// DefaultJPEGQuality ist die Standard-Qualitätsstufe für erzeugte JPEGs. 80 ist bei den
// hier üblichen Bildgrößen optisch unauffällig und deutlich kompakter als verlustfreies PNG.
const DefaultJPEGQuality = 80

// ConvertToJPEG dekodiert beliebige unterstützte Bild-Bytes (JPEG, PNG, GIF, WebP) und
// enkodiert sie als Baseline-JPEG.
//
// Zweck ist die Einbettung in PDFs: gofpdf beherrscht ausschließlich JPG, PNG und GIF und
// erkennt den Typ an der Dateiendung. Unsere Cover liegen aber als WebP auf der Platte, sind
// dort also nicht direkt einbettbar. JPEG statt PNG als Zielformat, weil Cover Fotos sind —
// ein PNG-Reencode eines 600×900-Covers ist rund zehnmal so groß, und eine Mahnliste enthält
// pro überfälligem Buch eines.
func ConvertToJPEG(imgBytes []byte, quality int) ([]byte, error) {
	// Gleicher Bomb-Schutz wie beim WebP-Pfad: die Datei stammt zwar aus unserem eigenen
	// Upload-Verzeichnis, wird hier aber ungeprüft dekodiert.
	if err := GuardImageDimensions(imgBytes); err != nil {
		return nil, err
	}
	img, _, err := image.Decode(bytes.NewReader(imgBytes))
	if err != nil {
		return nil, fmt.Errorf("fehler beim Dekodieren des Bildes: %w", err)
	}
	var out bytes.Buffer
	if err := jpeg.Encode(&out, flattenOnWhite(img), &jpeg.Options{Quality: quality}); err != nil {
		return nil, fmt.Errorf("fehler beim JPEG-Enkodieren: %w", err)
	}
	return out.Bytes(), nil
}

// flattenOnWhite legt ein Bild mit Alphakanal auf weißen Grund. JPEG kennt keine Transparenz:
// ohne dieses Flatten liefe der Alphakanal beim Enkodieren einfach weg und transparente
// Flächen würden schwarz — auf einem weißen Ausdruck ein auffälliger Fehler.
func flattenOnWhite(img image.Image) image.Image {
	// Alle Bildtypen des Standardpakets bieten Opaque(); ist es bereits deckend, sparen wir
	// uns die Kopie. Fehlt die Methode, wird sicherheitshalber geflattet.
	if o, ok := img.(interface{ Opaque() bool }); ok && o.Opaque() {
		return img
	}
	bounds := img.Bounds()
	dst := image.NewRGBA(bounds)
	draw.Draw(dst, bounds, image.NewUniform(color.White), image.Point{}, draw.Src)
	draw.Draw(dst, bounds, img, bounds.Min, draw.Over)
	return dst
}
