package inventur

import (
	"bytes"
	"image"
	"testing"
)

// decodeStored dekodiert die gespeicherten Cover-Bytes wieder zu einem Bild.
func decodeStored(t *testing.T, b []byte) image.Image {
	t.Helper()
	bild, _, err := image.Decode(bytes.NewReader(b))
	if err != nil {
		t.Fatalf("gespeichertes Cover ist kein dekodierbares Bild: %v", err)
	}
	return bild
}

func TestPrepareImageForStorageAlwaysWebP(t *testing.T) {
	bild := image.NewRGBA(image.Rect(0, 0, 300, 400))

	gespeichert, ext, err := bereiteBildVor(bild, 600, 900, 80)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ext != ".webp" {
		t.Fatalf("expected .webp extension, got %s", ext)
	}
	dekodierer := decodeStored(t, gespeichert)
	if dekodierer.Bounds().Dx() != 300 || dekodierer.Bounds().Dy() != 400 {
		t.Fatalf("expected 300x400 preserved, got %dx%d", dekodierer.Bounds().Dx(), dekodierer.Bounds().Dy())
	}
}

func TestPrepareImageForStorageResizesOversized(t *testing.T) {
	bild := image.NewRGBA(image.Rect(0, 0, 1200, 1800))

	gespeichert, ext, err := bereiteBildVor(bild, 600, 900, 80)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ext != ".webp" {
		t.Fatalf("expected .webp extension after resize, got %s", ext)
	}
	dekodierer := decodeStored(t, gespeichert)
	if dekodierer.Bounds().Dx() > 600 || dekodierer.Bounds().Dy() > 900 {
		t.Fatalf("expected resized image within 600x900, got %dx%d", dekodierer.Bounds().Dx(), dekodierer.Bounds().Dy())
	}
}

func TestPrepareImageForStorageResizesTallImage(t *testing.T) {
	bild := image.NewRGBA(image.Rect(0, 0, 1000, 5000))

	gespeichert, _, err := bereiteBildVor(bild, 600, 900, 80)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	dekodierer := decodeStored(t, gespeichert)
	if dekodierer.Bounds().Dy() != 900 {
		t.Fatalf("expected resized image height to be exactly 900, got %d", dekodierer.Bounds().Dy())
	}
}
