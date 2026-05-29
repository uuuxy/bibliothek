package inventur

import (
	"bytes"
	"image"
	"image/jpeg"
	"testing"
)

func TestPrepareImageForStorageKeepsSmallWebP(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 300, 400))
	orig := []byte("webp-bytes")

	stored, ext, err := prepareImageForStorage(orig, img, "webp", 600, 900, 82)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ext != ".webp" {
		t.Fatalf("expected .webp extension, got %s", ext)
	}
	if !bytes.Equal(stored, orig) {
		t.Fatalf("expected original bytes to be kept for small webp")
	}
}

func TestPrepareImageForStorageResizesOversizedToJPEG(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 1200, 1800))
	orig := []byte("placeholder")

	stored, ext, err := prepareImageForStorage(orig, img, "webp", 600, 900, 82)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ext != ".jpg" {
		t.Fatalf("expected .jpg extension after resize, got %s", ext)
	}

	decoded, err := jpeg.Decode(bytes.NewReader(stored))
	if err != nil {
		t.Fatalf("expected resized bytes to be valid jpeg: %v", err)
	}
	if decoded.Bounds().Dx() > 600 || decoded.Bounds().Dy() > 900 {
		t.Fatalf("expected resized image within 600x900, got %dx%d", decoded.Bounds().Dx(), decoded.Bounds().Dy())
	}
}
