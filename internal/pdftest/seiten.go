package pdftest

import (
	"regexp"
	"strconv"
	"testing"
)

// mediaBox findet die Seitengröße in Punkt. gofpdf schreibt sie je Seite als
// `/MediaBox [0 0 595.28 841.89]`.
var mediaBox = regexp.MustCompile(`/MediaBox\s*\[\s*0\s+0\s+([\d.]+)\s+([\d.]+)\s*\]`)

// seitenzahl liest den /Count-Eintrag des Seitenbaums.
var seitenzahl = regexp.MustCompile(`/Count\s+(\d+)`)

// Seiten liefert Seitenzahl und Seitenmaß (Breite, Höhe in mm, gerundet).
func Seiten(t *testing.T, roh []byte) (seiten int, breiteMM, hoeheMM int) {
	t.Helper()

	m := seitenzahl.FindSubmatch(roh)
	if m == nil {
		t.Fatalf("kein /Count im PDF (%d Bytes) — Seitenzahl nicht ermittelbar", len(roh))
	}
	seiten, err := strconv.Atoi(string(m[1]))
	if err != nil {
		t.Fatalf("/Count ist keine Zahl: %v", err)
	}

	b := mediaBox.FindSubmatch(roh)
	if b == nil {
		t.Fatalf("keine /MediaBox im PDF — Seitengröße nicht ermittelbar")
	}
	breite, err := strconv.ParseFloat(string(b[1]), 64)
	if err != nil {
		t.Fatalf("/MediaBox-Breite ist keine Zahl: %v", err)
	}
	hoehe, err := strconv.ParseFloat(string(b[2]), 64)
	if err != nil {
		t.Fatalf("/MediaBox-Höhe ist keine Zahl: %v", err)
	}
	// PDF rechnet in Punkt (1 pt = 1/72 Zoll), gofpdf hat mm bekommen.
	const mmProPunkt = 25.4 / 72.0
	return seiten, int(breite*mmProPunkt + 0.5), int(hoehe*mmProPunkt + 0.5)
}
