package imageutil

import (
	"bytes"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"

	"github.com/chai2010/webp"
)

// MaxImagePixels begrenzt die Gesamtpixelzahl eines Bildes, BEVOR es dekodiert wird.
// Schutz vor "Decompression Bombs": eine kleine, stark komprimierte Datei kann zu
// gewaltigen Dimensionen dekodieren (z. B. 30000×30000 px ≈ 3,6 GB RGBA) und den
// Server-Speicher erschöpfen. 50 MP sind großzügig für Cover/Passfotos, liegen aber
// weit unterhalb OOM-kritischer Allokationen.
const MaxImagePixels = 50_000_000

// GuardImageDimensions liest ausschließlich den Bild-Header (image.DecodeConfig allokiert
// KEINE Pixeldaten) und lehnt Bilder ab, deren Pixelzahl das Limit überschreitet. So wird
// die teure, speicherintensive Dekodierung bei Decompression-Bombs gar nicht erst gestartet.
func GuardImageDimensions(imgBytes []byte) error {
	cfg, _, err := image.DecodeConfig(bytes.NewReader(imgBytes))
	if err != nil {
		return fmt.Errorf("fehler beim Lesen des Bild-Headers: %w", err)
	}
	if cfg.Width <= 0 || cfg.Height <= 0 {
		return fmt.Errorf("ungültige Bilddimensionen")
	}
	if int64(cfg.Width)*int64(cfg.Height) > MaxImagePixels {
		return fmt.Errorf("bild zu groß: %dx%d Pixel überschreitet das Limit von %d Megapixeln",
			cfg.Width, cfg.Height, MaxImagePixels/1_000_000)
	}
	return nil
}

// ConvertToWebP dekodiert die übergebenen Bild-Bytes (JPEG/PNG) und
// enkodiert sie in das WebP-Format mit der angegebenen Qualität.
func ConvertToWebP(imgBytes []byte, quality float32) ([]byte, error) {
	if err := GuardImageDimensions(imgBytes); err != nil {
		return nil, err
	}
	img, _, err := image.Decode(bytes.NewReader(imgBytes))
	if err != nil {
		return nil, fmt.Errorf("fehler beim Dekodieren des Bildes (muss JPG oder PNG sein): %w", err)
	}
	return EncodeImageWebP(img, quality)
}

// EncodeImageWebP enkodiert ein bereits dekodiertes Bild verlustbehaftet als WebP.
func EncodeImageWebP(img image.Image, quality float32) ([]byte, error) {
	var out bytes.Buffer
	if err := webp.Encode(&out, img, &webp.Options{Lossless: false, Quality: quality}); err != nil {
		return nil, fmt.Errorf("fehler beim WebP-Enkodieren: %w", err)
	}
	return out.Bytes(), nil
}
