package inventur

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"testing"

	"bibliothek/internal/pdftest"
)

// Bildplatzierungen im Inhaltsstrom: `q 10.00 0 0 15.00 17.00 260.00 cm /I1 Do Q`.
// Die vierte Zahl ist die Höhe, die sechste die y-Position VON UNTEN (PDF-Koordinaten).
var bildMatrix = regexp.MustCompile(`([0-9.]+) 0 0 ([0-9.]+) ([0-9.]+) ([0-9.]+) cm`)

// TestSchulbuecherAlsPDF_CoverBleibtBeiSeinerZeile hält den Seitenumbruch fest.
//
// gofpdf prüft den Umbruch erst in der ersten Tabellenzelle. Das Cover wird aber mit
// fester Position gezeichnet und ging deshalb noch auf die ALTE Seite, während seine
// Zeile schon auf der neuen stand: ab der 14. Zeile klebte auf jedem Blattende ein
// herrenloses Buchcover im Fußsteg, und die zugehörige Zeile hatte oben ein leeres
// Bildkästchen — was aussieht wie „für dieses Buch ist kein Cover hinterlegt"
// (Befund Rasterdurchgang 03.09.2026). Der PG-Test sah es nicht: drei Titel, eine Seite.
func TestSchulbuecherAlsPDF_CoverBleibtBeiSeinerZeile(t *testing.T) {
	// coverdatei liest relativ zum Arbeitsverzeichnis; ein eigenes uploads/ mit einem
	// echten Bild darin macht den Cover-Pfad im Test begehbar.
	verzeichnis := t.TempDir()
	t.Chdir(verzeichnis)
	if err := os.MkdirAll(filepath.Join(verzeichnis, "uploads"), 0o750); err != nil {
		t.Fatal(err)
	}
	bild := image.NewRGBA(image.Rect(0, 0, 40, 60))
	for y := 0; y < 60; y++ {
		for x := 0; x < 40; x++ {
			bild.Set(x, y, color.RGBA{R: 20, G: 60, B: 160, A: 255})
		}
	}
	var puffer bytes.Buffer
	if err := png.Encode(&puffer, bild); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(verzeichnis, "uploads", "cover.png"), puffer.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}

	titel := make([]LernmittelTitel, 0, 30)
	for i := 1; i <= 30; i++ {
		titel = append(titel, LernmittelTitel{
			ID: strconv.Itoa(i), Title: fmt.Sprintf("Titel %02d", i), Autor: "Verlag",
			CoverURL: "/uploads/cover.png", Gesamt: i, Verfuegbar: i,
		})
	}

	doc, err := SchulbuecherAlsPDF(titel, "Biologie", "")
	if err != nil {
		t.Fatal(err)
	}
	pdftest.IstPDF(t, doc, "Schulbuch-Export über mehrere Seiten")

	treffer := bildMatrix.FindAllStringSubmatch(string(pdftest.Inhalt(t, doc)), -1)
	if len(treffer) < 20 {
		t.Fatalf("erwartet ein Cover je Zeile, gefunden %d Bildplatzierungen — "+
			"der Detektor greift nicht mehr", len(treffer))
	}
	// Kein Bild darf in den Fußsteg ragen: Unterkante = y (von unten) muss mindestens
	// dem unteren Rand entsprechen. Mit dem alten Fehler lag ein Bild je Seite bei y≈5.
	for _, m := range treffer {
		y, err := strconv.ParseFloat(m[4], 64)
		if err != nil {
			t.Fatalf("Bildmatrix unlesbar: %v", m)
		}
		if y < 15 {
			t.Errorf("Cover ragt in den Fußsteg (y=%.1f von unten) — es gehört zur Zeile "+
				"auf der nächsten Seite, steht aber noch auf dieser", y)
		}
	}
}
