package imageutil

import (
	"bytes"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"

	"github.com/chai2010/webp"
)

// ConvertToWebP dekodiert die übergebenen Bild-Bytes (JPEG/PNG) und 
// enkodiert sie in das WebP-Format mit der angegebenen Qualität.
func ConvertToWebP(imgBytes []byte, quality float32) ([]byte, error) {
	img, _, err := image.Decode(bytes.NewReader(imgBytes))
	if err != nil {
		return nil, fmt.Errorf("fehler beim Dekodieren des Bildes (muss JPG oder PNG sein): %w", err)
	}

	var out bytes.Buffer
	err = webp.Encode(&out, img, &webp.Options{Lossless: false, Quality: quality})
	if err != nil {
		return nil, fmt.Errorf("fehler beim WebP-Enkodieren: %w", err)
	}

	return out.Bytes(), nil
}
