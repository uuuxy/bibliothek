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

// makePNGHeader baut die Bytes eines gültigen PNG-Signatur- + IHDR-Chunks mit den
// angegebenen Dimensionen. image.DecodeConfig liest daraus Breite/Höhe, ohne dass die
// (ggf. riesige) Pixelmatrix alloziert werden muss — ideal zum Testen des Bomb-Schutzes.
func makePNGHeader(width, height uint32) []byte {
	var b bytes.Buffer
	b.Write([]byte{0x89, 'P', 'N', 'G', 0x0d, 0x0a, 0x1a, 0x0a}) // PNG-Signatur

	ihdr := make([]byte, 13)
	binary.BigEndian.PutUint32(ihdr[0:4], width)
	binary.BigEndian.PutUint32(ihdr[4:8], height)
	ihdr[8] = 8 // bit depth
	ihdr[9] = 2 // color type: truecolor (RGB)
	// ihdr[10..12] = compression/filter/interlace = 0

	var lenBuf [4]byte
	binary.BigEndian.PutUint32(lenBuf[:], uint32(len(ihdr)))
	b.Write(lenBuf[:])
	b.WriteString("IHDR")
	b.Write(ihdr)

	crc := crc32.NewIEEE()
	crc.Write([]byte("IHDR"))
	crc.Write(ihdr)
	var crcBuf [4]byte
	binary.BigEndian.PutUint32(crcBuf[:], crc.Sum32())
	b.Write(crcBuf[:])

	return b.Bytes()
}

func TestGuardImageDimensions_RejectsDecompressionBomb(t *testing.T) {
	// 60000 × 60000 = 3,6 Mrd. Pixel → weit über MaxImagePixels.
	bomb := makePNGHeader(60000, 60000)
	if err := GuardImageDimensions(bomb); err == nil {
		t.Fatal("erwartete Ablehnung eines übergroßen Bildes, bekam nil")
	}
}

func TestGuardImageDimensions_AcceptsNormalImage(t *testing.T) {
	var buf bytes.Buffer
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	img.Set(0, 0, color.RGBA{R: 1, G: 2, B: 3, A: 255})
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("setup: png encode: %v", err)
	}
	if err := GuardImageDimensions(buf.Bytes()); err != nil {
		t.Fatalf("normales Bild wurde fälschlich abgelehnt: %v", err)
	}
}

func TestGuardImageDimensions_RejectsGarbage(t *testing.T) {
	if err := GuardImageDimensions([]byte("kein bild")); err == nil {
		t.Fatal("erwartete Fehler bei ungültigem Header, bekam nil")
	}
}
