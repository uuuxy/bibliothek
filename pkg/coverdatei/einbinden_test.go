package coverdatei

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"testing"

	"bibliothek/internal/pdftest"

	"github.com/jung-kurt/gofpdf"
)

// Bildplatzierung im Inhaltsstrom, in Punkt und von unten gemessen:
// `q 19.84 0 0 29.76 51.02 562.68 cm /I… Do Q` — Breite, Höhe, linker Rand, Unterkante.
var bildPlatz = regexp.MustCompile(`q ([0-9.]+) 0 0 ([0-9.]+) ([0-9.]+) ([0-9.]+) cm /I[0-9a-f]+ Do Q`)

// schreibeCover legt ein Bild der genannten Größe ab und liefert seine Adresse.
func schreibeCover(t *testing.T, name string, breite, hoehe int) string {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, breite, hoehe))
	for y := 0; y < hoehe; y++ {
		for x := 0; x < breite; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x), G: uint8(y), B: 90, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("PNG erzeugen: %v", err)
	}
	if err := os.WriteFile(filepath.Join(Wurzel, name), buf.Bytes(), 0o600); err != nil {
		t.Fatalf("Bild schreiben: %v", err)
	}
	return "/" + Wurzel + "/" + name
}

// blattMit legt ein Dokument an, ruft fn auf seiner ersten Seite und liefert das fertige PDF.
// Ein Fehler am Dokument bricht den Test ab: Er stünde sonst erst bei Output.
func blattMit(t *testing.T, fn func(doc *gofpdf.Fpdf)) []byte {
	t.Helper()
	doc := gofpdf.New("P", "mm", "A4", "")
	doc.AddPage()
	doc.SetFont("Arial", "", 10)
	doc.Cell(40, 5, "Blatt")
	fn(doc)
	var buf bytes.Buffer
	if err := doc.Output(&buf); err != nil {
		t.Fatalf("Dokument schreiben: %v", err)
	}
	return buf.Bytes()
}

// platzierungen liest Breite, Höhe, linken Rand und Oberkante jedes Bilds in Millimetern.
func platzierungen(t *testing.T, roh []byte) [][4]float64 {
	t.Helper()
	const mm = 72 / 25.4
	var orte [][4]float64
	for _, m := range bildPlatz.FindAllSubmatch(pdftest.Inhalt(t, roh), -1) {
		var w [4]float64
		for i := range w {
			zahl, err := strconv.ParseFloat(string(m[i+1]), 64)
			if err != nil {
				t.Fatalf("Zahl im Inhaltsstrom unlesbar: %q", m[i+1])
			}
			w[i] = zahl / mm
		}
		// Aus der Unterkante von unten wird die Oberkante von oben (A4: 297 mm).
		w[3] = 297 - w[3] - w[1]
		orte = append(orte, w)
	}
	return orte
}

// Das Cover steht im Seitenverhältnis des Bilds in seinem Platz und darin mittig: ein
// Hochformat so hoch wie der Platz, ein Querformat so breit.
func TestBindeEin_SetztDasCoverImSeitenverhaeltnisInDenPlatz(t *testing.T) {
	bereiteWurzel(t)
	hoch := schreibeCover(t, "hoch.png", 40, 60)
	quer := schreibeCover(t, "quer.png", 120, 40)
	genau := schreibeCover(t, "genau.png", 30, 45)

	for _, f := range []struct {
		name  string
		cover string
		platz CoverPlatz
		will  [4]float64 // Breite, Höhe, linker Rand, Oberkante
	}{
		{"Hochformat im Quadrat", hoch, CoverPlatz{X: 20, Y: 30, Breite: 12, Hoehe: 12}, [4]float64{8, 12, 22, 30}},
		{"Querformat im Hochformat", quer, CoverPlatz{X: 20, Y: 30, Breite: 12, Hoehe: 18}, [4]float64{12, 4, 20, 37}},
		{"Bild in der Form des Platzes", genau, CoverPlatz{X: 50, Y: 60, Breite: 10, Hoehe: 15}, [4]float64{10, 15, 50, 60}},
	} {
		roh := blattMit(t, func(doc *gofpdf.Fpdf) {
			if !BindeEin(doc, f.cover, f.platz) {
				t.Errorf("%s: kein Cover gesetzt", f.name)
			}
		})
		orte := platzierungen(t, roh)
		if len(orte) != 1 {
			t.Errorf("%s: %d Bilder auf dem Blatt, erwartet 1", f.name, len(orte))
			continue
		}
		for i, wert := range orte[0] {
			if math.Abs(wert-f.will[i]) > 0.05 {
				t.Errorf("%s: Breite, Höhe, links, oben = %.2f, erwartet %.2f", f.name, orte[0], f.will)
				break
			}
		}
	}
}

// Ohne brauchbares Cover setzt BindeEin nichts, meldet es und lässt das Dokument heil.
func TestBindeEin_OhneCoverBleibtDasBlattHeil(t *testing.T) {
	basis := bereiteWurzel(t)
	if err := os.WriteFile(filepath.Join(Wurzel, "kaputt.png"), []byte("das ist kein bild"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(basis, "geheim.txt"), []byte("geheim"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, coverURL := range []string{"", "/uploads/fehlt.png", "/uploads/kaputt.png", "https://example.org/cover.png", "/uploads/../geheim.txt"} {
		roh := blattMit(t, func(doc *gofpdf.Fpdf) {
			if BindeEin(doc, coverURL, CoverPlatz{X: 20, Y: 30, Breite: 10, Hoehe: 15}) {
				t.Errorf("%q: als Cover gesetzt", coverURL)
			}
		})
		if orte := platzierungen(t, roh); len(orte) != 0 {
			t.Errorf("%q: %d Bilder auf dem Blatt", coverURL, len(orte))
		}
	}
}

// Dasselbe Cover in mehreren Zeilen steht einmal im Dokument und an jeder Stelle auf dem Blatt.
func TestBindeEin_DasselbeCoverStehtEinmalImDokument(t *testing.T) {
	bereiteWurzel(t)
	cover := schreibeCover(t, "doppelt.png", 40, 60)
	roh := blattMit(t, func(doc *gofpdf.Fpdf) {
		for zeile := 0; zeile < 3; zeile++ {
			if !BindeEin(doc, cover, CoverPlatz{X: 20, Y: 30 + float64(zeile)*20, Breite: 10, Hoehe: 15}) {
				t.Errorf("Zeile %d: kein Cover gesetzt", zeile)
			}
		}
	})
	if n := bytes.Count(roh, []byte("/Subtype /Image")); n != 1 {
		t.Errorf("%d Bilder im Dokument, erwartet 1", n)
	}
	if orte := platzierungen(t, roh); len(orte) != 3 {
		t.Errorf("%d Platzierungen auf dem Blatt, erwartet 3", len(orte))
	}
}
