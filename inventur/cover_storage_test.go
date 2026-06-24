package inventur

import (
	"bytes"
	"image"
	"testing"
)

// decodeStored dekodiert die gespeicherten Cover-Bytes wieder zu einem Bild.
func decodeStored(t *testing.T, b []byte) image.Image {
	t.Helper()
	img, _, err := image.Decode(bytes.NewReader(b))
	if err != nil {
		t.Fatalf("gespeichertes Cover ist kein dekodierbares Bild: %v", err)
	}
	return img
}

func TestPrepareImageForStorageAlwaysWebP(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 300, 400))

	stored, ext, err := prepareImageForStorage(img, 600, 900, 80)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ext != ".webp" {
		t.Fatalf("expected .webp extension, got %s", ext)
	}
	dec := decodeStored(t, stored)
	if dec.Bounds().Dx() != 300 || dec.Bounds().Dy() != 400 {
		t.Fatalf("expected 300x400 preserved, got %dx%d", dec.Bounds().Dx(), dec.Bounds().Dy())
	}
}

func TestPrepareImageForStorageResizesOversized(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 1200, 1800))

	stored, ext, err := prepareImageForStorage(img, 600, 900, 80)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ext != ".webp" {
		t.Fatalf("expected .webp extension after resize, got %s", ext)
	}
	dec := decodeStored(t, stored)
	if dec.Bounds().Dx() > 600 || dec.Bounds().Dy() > 900 {
		t.Fatalf("expected resized image within 600x900, got %dx%d", dec.Bounds().Dx(), dec.Bounds().Dy())
	}
}

func TestPrepareImageForStorageResizesTallImage(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 1000, 5000))

	stored, _, err := prepareImageForStorage(img, 600, 900, 80)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	dec := decodeStored(t, stored)
	if dec.Bounds().Dy() != 900 {
		t.Fatalf("expected resized image height to be exactly 900, got %d", dec.Bounds().Dy())
	}
}
