package inventur

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"

	"golang.org/x/image/draw"
)

// prepareImageForStorage keeps original bytes for already-small images and
// only re-encodes oversized images as JPEG after resizing.
func prepareImageForStorage(originalBytes []byte, img image.Image, format string, maxWidth, maxHeight, jpegQuality int) ([]byte, string, error) {
	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()

	saveExt := ".jpg"
	switch format {
	case "jpeg":
		saveExt = ".jpg"
	case "png":
		saveExt = ".png"
	case "webp":
		saveExt = ".webp"
	}

	if width <= maxWidth && height <= maxHeight {
		return originalBytes, saveExt, nil
	}

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

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, dst, &jpeg.Options{Quality: jpegQuality}); err != nil {
		return nil, "", fmt.Errorf("jpeg encode failed: %w", err)
	}

	return buf.Bytes(), ".jpg", nil
}
