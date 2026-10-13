package imageutil

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"testing"
)

// rauschbild liefert ein Bild mit wechselnden Farben. Einfarbige Flächen packen so stark, dass
// ein Vergleich der Größen nichts aussagte.
func rauschbild(w, h int) *image.RGBA {
	bild := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			bild.Set(x, y, color.RGBA{R: uint8(x * 7), G: uint8(y * 5), B: uint8(x + y), A: 255})
		}
	}
	return bild
}

// Cover liegen als WebP auf der Platte, und gofpdf kennt WebP nicht: Diesen Weg nimmt jedes
// PDF mit Cover.
func TestAlsJPEG_NimmtWebPAn(t *testing.T) {
	webpDaten, err := ErzeugeWebP(rauschbild(60, 90), 80)
	if err != nil {
		t.Fatalf("Vorbereitung: WebP erzeugen: %v", err)
	}

	jpegDaten, err := AlsJPEG(webpDaten, JPEGQualitaet)
	if err != nil {
		t.Fatalf("WebP konnte nicht nach JPEG gewandelt werden: %v", err)
	}

	kopf, format, err := image.DecodeConfig(bytes.NewReader(jpegDaten))
	if err != nil {
		t.Fatalf("Ergebnis ist kein dekodierbares Bild: %v", err)
	}
	if format != "jpeg" {
		t.Errorf("Format = %q, erwartet jpeg", format)
	}
	if kopf.Width != 60 || kopf.Height != 90 {
		t.Errorf("Dimensionen = %dx%d, erwartet 60x90", kopf.Width, kopf.Height)
	}
}

func TestAlsJPEG_NimmtPNGUndJPEGAn(t *testing.T) {
	var pngPuffer bytes.Buffer
	if err := png.Encode(&pngPuffer, rauschbild(20, 20)); err != nil {
		t.Fatalf("Vorbereitung: PNG erzeugen: %v", err)
	}
	var jpegPuffer bytes.Buffer
	if err := jpeg.Encode(&jpegPuffer, rauschbild(20, 20), nil); err != nil {
		t.Fatalf("Vorbereitung: JPEG erzeugen: %v", err)
	}

	for name, eingabe := range map[string][]byte{"png": pngPuffer.Bytes(), "jpeg": jpegPuffer.Bytes()} {
		t.Run(name, func(t *testing.T) {
			if _, err := AlsJPEG(eingabe, JPEGQualitaet); err != nil {
				t.Fatalf("%s wurde abgelehnt: %v", name, err)
			}
		})
	}
}

// JPEG hat keinen Alphakanal. Ohne den weißen Grund würden durchsichtige Flächen schwarz,
// und kein Fehler meldete es.
func TestAlsJPEG_LegtTransparenzAufWeiss(t *testing.T) {
	bild := image.NewRGBA(image.Rect(0, 0, 10, 10))
	// Ein neues Bild ist ganz durchsichtig (Alpha 0); es muss nichts gesetzt werden.
	var puffer bytes.Buffer
	if err := png.Encode(&puffer, bild); err != nil {
		t.Fatalf("Vorbereitung: PNG erzeugen: %v", err)
	}

	jpegDaten, err := AlsJPEG(puffer.Bytes(), JPEGQualitaet)
	if err != nil {
		t.Fatalf("AlsJPEG: %v", err)
	}
	ergebnis, err := jpeg.Decode(bytes.NewReader(jpegDaten))
	if err != nil {
		t.Fatalf("Ergebnis nicht dekodierbar: %v", err)
	}

	r, g, b, _ := ergebnis.At(5, 5).RGBA()
	// JPEG ist verlustbehaftet, deshalb eine Schwelle statt Gleichheit; Schwarz (0) läge weit
	// darunter.
	const hell = 0xF000
	if r < hell || g < hell || b < hell {
		t.Errorf("transparenter Bereich wurde zu (%d,%d,%d), erwartet nahezu weiß", r, g, b)
	}
}

func TestAlsJPEG_WeistUnbrauchbareEingabeAb(t *testing.T) {
	if _, err := AlsJPEG([]byte("kein bild"), JPEGQualitaet); err == nil {
		t.Fatal("erwartete Fehler bei ungültigen Bilddaten, bekam nil")
	}
	// Die Grenze der Pixelzahl gilt auch auf diesem Weg, nicht nur beim Hochladen.
	if _, err := AlsJPEG(pngKopf(60000, 60000), JPEGQualitaet); err == nil {
		t.Fatal("erwartete Ablehnung eines übergroßen Bildes, bekam nil")
	}
}
