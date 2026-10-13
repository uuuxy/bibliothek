package imageutil

import (
	"bytes"
	"fmt"
	"image"
)

// maxPixel begrenzt die Pixelzahl eines Bilds. Eine kleine, stark gepackte Datei kann zu
// 30000 × 30000 Pixeln auspacken, das sind rund 3,6 GB im Speicher. 50 Megapixel reichen für
// Cover und Ausweisfotos und liegen weit unter dem, was den Server gefährdet.
const maxPixel = 50_000_000

// PruefeBildgroesse weist ein Bild ab, dessen Pixelzahl über der Grenze liegt. Sie liest dafür
// nur den Kopf der Datei, sodass ein übergroßes Bild keinen Speicher belegt.
func PruefeBildgroesse(daten []byte) error {
	kopf, _, err := image.DecodeConfig(bytes.NewReader(daten))
	if err != nil {
		return fmt.Errorf("fehler beim Lesen des Bild-Headers: %w", err)
	}
	if kopf.Width <= 0 || kopf.Height <= 0 {
		return fmt.Errorf("ungültige Bilddimensionen")
	}
	if int64(kopf.Width)*int64(kopf.Height) > maxPixel {
		return fmt.Errorf("bild zu groß: %dx%d Pixel überschreitet das Limit von %d Megapixeln",
			kopf.Width, kopf.Height, maxPixel/1_000_000)
	}
	return nil
}
