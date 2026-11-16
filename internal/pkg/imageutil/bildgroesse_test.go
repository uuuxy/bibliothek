package imageutil

import (
	"bytes"
	"encoding/binary"
	"hash/crc32"
	"image"
	"image/color"
	"image/png"
	"testing"
)

// pngKopf baut Signatur und IHDR-Abschnitt eines PNG mit den genannten Maßen. Daraus liest
// image.DecodeConfig Breite und Höhe; ein übergroßes Bild lässt sich so prüfen, ohne dass
// seine Pixel im Speicher stehen.
func pngKopf(breite, hoehe uint32) []byte {
	var b bytes.Buffer
	b.Write([]byte{0x89, 'P', 'N', 'G', 0x0d, 0x0a, 0x1a, 0x0a}) // Signatur

	ihdr := make([]byte, 13)
	binary.BigEndian.PutUint32(ihdr[0:4], breite)
	binary.BigEndian.PutUint32(ihdr[4:8], hoehe)
	ihdr[8] = 8 // Bittiefe
	ihdr[9] = 2 // Farbtyp: RGB ohne Palette
	// ihdr[10] bis ihdr[12] bleiben 0: Kompression, Filter, Interlace.

	var laenge [4]byte
	binary.BigEndian.PutUint32(laenge[:], uint32(len(ihdr)))
	b.Write(laenge[:])
	b.WriteString("IHDR")
	b.Write(ihdr)

	crc := crc32.NewIEEE()
	crc.Write([]byte("IHDR"))
	crc.Write(ihdr)
	var pruefsumme [4]byte
	binary.BigEndian.PutUint32(pruefsumme[:], crc.Sum32())
	b.Write(pruefsumme[:])

	return b.Bytes()
}

func TestPruefeBildgroesse_WeistUebergrossesBildAb(t *testing.T) {
	// 60000 × 60000 sind 3,6 Milliarden Pixel, weit über maxPixel.
	uebergross := pngKopf(60000, 60000)
	if err := PruefeBildgroesse(uebergross); err == nil {
		t.Fatal("erwartete Ablehnung eines übergroßen Bildes, bekam nil")
	}
}

func TestPruefeBildgroesse_NimmtGewoehnlichesBildAn(t *testing.T) {
	var puffer bytes.Buffer
	bild := image.NewRGBA(image.Rect(0, 0, 100, 100))
	bild.Set(0, 0, color.RGBA{R: 1, G: 2, B: 3, A: 255})
	if err := png.Encode(&puffer, bild); err != nil {
		t.Fatalf("Vorbereitung: PNG erzeugen: %v", err)
	}
	if err := PruefeBildgroesse(puffer.Bytes()); err != nil {
		t.Fatalf("normales Bild wurde fälschlich abgelehnt: %v", err)
	}
}

func TestPruefeBildgroesse_WeistUnlesbarenKopfAb(t *testing.T) {
	if err := PruefeBildgroesse([]byte("kein bild")); err == nil {
		t.Fatal("erwartete Fehler bei ungültigem Header, bekam nil")
	}
}
