package inventur

import (
	"image"

	"bibliothek/pkg/imageutil"

	"golang.org/x/image/draw"
)

// prepareImageForStorage re-enkodiert ein Cover-Bild einheitlich als WebP und skaliert
// übergroße Bilder auf maxWidth x maxHeight herunter. Das Ergebnis ist immer WebP
// (".webp"), damit der lokale Cover-Cache kompakt und einheitlich bleibt.
func prepareImageForStorage(img image.Image, maxWidth, maxHeight int, quality float32) ([]byte, string, error) {
	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()

	src := img
	if width > maxWidth || height > maxHeight {
		ratio := float64(width) / float64(height)
		newWidth, newHeight := width, height

		if newWidth > maxWidth {
			newWidth = maxWidth
			newHeight = int(float64(newWidth) / ratio)
		}
		if newHeight > maxHeight {
			newHeight = maxHeight
			newWidth = int(float64(newHeight) * ratio)
		}

		dst := image.NewRGBA(image.Rect(0, 0, newWidth, newHeight))
		draw.ApproxBiLinear.Scale(dst, dst.Bounds(), img, bounds, draw.Over, nil)
		src = dst
	}

	out, err := imageutil.EncodeImageWebP(src, quality)
	if err != nil {
		return nil, "", err
	}
	return out, ".webp", nil
}
