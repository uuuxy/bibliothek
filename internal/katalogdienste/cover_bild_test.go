package katalogdienste

import (
	"bytes"
	"image"
	"testing"
)

// leseAbgelegtesCover dekodiert die gespeicherten Cover-Bytes wieder zu einem Bild.
func leseAbgelegtesCover(t *testing.T, b []byte) image.Image {
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
		t.Fatalf("bereiteBildVor(…) = Endung %s, erwartet .webp", ext)
	}
	dekodierer := leseAbgelegtesCover(t, gespeichert)
	if dekodierer.Bounds().Dx() != 300 || dekodierer.Bounds().Dy() != 400 {
		t.Fatalf("bereiteBildVor(300×400) = %d×%d Pixel, erwartet 300×400 unverändert", dekodierer.Bounds().Dx(), dekodierer.Bounds().Dy())
	}
}

func TestPrepareImageForStorageResizesOversized(t *testing.T) {
	bild := image.NewRGBA(image.Rect(0, 0, 1200, 1800))

	gespeichert, ext, err := bereiteBildVor(bild, 600, 900, 80)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ext != ".webp" {
		t.Fatalf("bereiteBildVor(großes Bild) = Endung %s, erwartet .webp", ext)
	}
	dekodierer := leseAbgelegtesCover(t, gespeichert)
	if dekodierer.Bounds().Dx() > 600 || dekodierer.Bounds().Dy() > 900 {
		t.Fatalf("bereiteBildVor(großes Bild) = %d×%d Pixel, erwartet höchstens 600×900", dekodierer.Bounds().Dx(), dekodierer.Bounds().Dy())
	}
}

func TestPrepareImageForStorageResizesTallImage(t *testing.T) {
	bild := image.NewRGBA(image.Rect(0, 0, 1000, 5000))

	gespeichert, _, err := bereiteBildVor(bild, 600, 900, 80)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	dekodierer := leseAbgelegtesCover(t, gespeichert)
	if dekodierer.Bounds().Dy() != 900 {
		t.Fatalf("bereiteBildVor(…) = Höhe %d, erwartet genau 900", dekodierer.Bounds().Dy())
	}
}
