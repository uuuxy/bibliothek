package imageutil

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"testing"
)

// rauschbild erzeugt ein Bild mit wechselnden Farben. Einfarbige Flächen komprimieren so
// extrem, dass Größenvergleiche in den Tests nichts mehr aussagen würden.
func rauschbild(w, h int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x * 7), G: uint8(y * 5), B: uint8(x + y), A: 255})
		}
	}
	return img
}

// TestConvertToJPEG_AkzeptiertWebP ist der Kern der Sache: WebP ist unser Speicherformat für
// Cover, und genau dieses Format konnte die PDF-Erzeugung vorher nicht verarbeiten.
func TestConvertToJPEG_AkzeptiertWebP(t *testing.T) {
	webpBytes, err := EncodeImageWebP(rauschbild(60, 90), 80)
	if err != nil {
		t.Fatalf("setup: webp encode: %v", err)
	}

	jpgBytes, err := ConvertToJPEG(webpBytes, DefaultJPEGQuality)
	if err != nil {
		t.Fatalf("WebP konnte nicht nach JPEG gewandelt werden: %v", err)
	}

	cfg, format, err := image.DecodeConfig(bytes.NewReader(jpgBytes))
	if err != nil {
		t.Fatalf("Ergebnis ist kein dekodierbares Bild: %v", err)
	}
	if format != "jpeg" {
		t.Errorf("Format = %q, erwartet jpeg", format)
	}
	if cfg.Width != 60 || cfg.Height != 90 {
		t.Errorf("Dimensionen = %dx%d, erwartet 60x90", cfg.Width, cfg.Height)
	}
}

func TestConvertToJPEG_AkzeptiertPNGundJPEG(t *testing.T) {
	var pngBuf bytes.Buffer
	if err := png.Encode(&pngBuf, rauschbild(20, 20)); err != nil {
		t.Fatalf("setup: png encode: %v", err)
	}
	var jpgBuf bytes.Buffer
	if err := jpeg.Encode(&jpgBuf, rauschbild(20, 20), nil); err != nil {
		t.Fatalf("setup: jpeg encode: %v", err)
	}

	for name, eingabe := range map[string][]byte{"png": pngBuf.Bytes(), "jpeg": jpgBuf.Bytes()} {
		t.Run(name, func(t *testing.T) {
			if _, err := ConvertToJPEG(eingabe, DefaultJPEGQuality); err != nil {
				t.Fatalf("%s wurde abgelehnt: %v", name, err)
			}
		})
	}
}

// TestConvertToJPEG_FlattetTransparenzAufWeiss sichert das Verhalten ab, das ohne Flatten
// still kaputtginge: JPEG hat keinen Alphakanal, transparente Flächen würden schwarz.
func TestConvertToJPEG_FlattetTransparenzAufWeiss(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 10, 10))
	// vollständig transparent (Alpha 0) — nichts setzen genügt
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("setup: png encode: %v", err)
	}

	jpgBytes, err := ConvertToJPEG(buf.Bytes(), DefaultJPEGQuality)
	if err != nil {
		t.Fatalf("ConvertToJPEG: %v", err)
	}
	ergebnis, err := jpeg.Decode(bytes.NewReader(jpgBytes))
	if err != nil {
		t.Fatalf("Ergebnis nicht dekodierbar: %v", err)
	}

	r, g, b, _ := ergebnis.At(5, 5).RGBA()
	// JPEG ist verlustbehaftet, deshalb Schwelle statt Gleichheit — Schwarz (0) läge weit darunter.
	const hell = 0xF000
	if r < hell || g < hell || b < hell {
		t.Errorf("transparenter Bereich wurde zu (%d,%d,%d), erwartet nahezu weiß", r, g, b)
	}
}

func TestConvertToJPEG_LehntUnbrauchbareEingabeAb(t *testing.T) {
	if _, err := ConvertToJPEG([]byte("kein bild"), DefaultJPEGQuality); err == nil {
		t.Fatal("erwartete Fehler bei ungültigen Bilddaten, bekam nil")
	}
	// Der Bomb-Schutz muss auch auf diesem Pfad greifen, nicht nur beim WebP-Upload.
	if _, err := ConvertToJPEG(makePNGHeader(60000, 60000), DefaultJPEGQuality); err == nil {
		t.Fatal("erwartete Ablehnung einer Decompression-Bomb, bekam nil")
	}
}
