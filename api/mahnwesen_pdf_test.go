package api

import (
	"bytes"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"testing"

	"bibliothek/pkg/imageutil"
	"bibliothek/repository"
)

// mahnPDFTestUmgebung legt ein temporäres Arbeitsverzeichnis mit uploads/-Ordner an.
// Die Cover-Auflösung arbeitet relativ zum Arbeitsverzeichnis des Servers, deshalb der Chdir.
func mahnPDFTestUmgebung(t *testing.T) {
	t.Helper()
	t.Chdir(t.TempDir())
	if err := os.Mkdir(uploadsVerzeichnis, 0o750); err != nil {
		t.Fatalf("setup: uploads-Verzeichnis: %v", err)
	}
}

// schreibeCoverDatei legt ein Cover im WebP-Format ab — genau so, wie es der Cover-Download
// in inventur/cover_storage.go tut. Rückgabe ist die CoverURL, wie sie in der DB steht.
func schreibeCoverDatei(t *testing.T, name string) string {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 60, 90))
	for y := 0; y < 90; y++ {
		for x := 0; x < 60; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x * 4), G: uint8(y * 2), B: uint8(x + y), A: 255})
		}
	}
	daten, err := imageutil.EncodeImageWebP(img, 80)
	if err != nil {
		t.Fatalf("setup: webp encode: %v", err)
	}
	if err := os.WriteFile(filepath.Join(uploadsVerzeichnis, name), daten, 0o600); err != nil {
		t.Fatalf("setup: cover schreiben: %v", err)
	}
	return "/" + uploadsVerzeichnis + "/" + name
}

// mahnKlassenMit baut eine minimale Mahnwesen-Struktur mit einem überfälligen Medium.
func mahnKlassenMit(coverURL string) []repository.MahnwesenKlasse {
	return []repository.MahnwesenKlasse{{
		Klasse: "07B",
		Schueler: []repository.UeberfaelligerSchueler{{
			SchuelerID: "s-1",
			Name:       "Test Schüler",
			Klasse:     "07B",
			Medien: []repository.UeberfaelligesMedium{{
				AusleiheID:       "a-1",
				Titel:            "Das überfällige Buch",
				Autor:            "A. Utor",
				Barcode:          "B-0001",
				CoverURL:         coverURL,
				FaelligAm:        "01.06.2026",
				TageUeberfaellig: 21,
			}},
		}},
	}}
}

// TestGenerateMahnPDF_WebPCoverBrichtDasDokumentNicht ist der Regressionstest zum Bug:
// gofpdf erkennt den Bildtyp an der Dateiendung und kann kein WebP. Der dabei gesetzte
// Fehlerzustand ist klebrig und schlug in pdf.Output() durch — ein einziges WebP-Cover
// ließ die komplette Mahnliste mit HTTP 500 scheitern.
func TestGenerateMahnPDF_WebPCoverBrichtDasDokumentNicht(t *testing.T) {
	mahnPDFTestUmgebung(t)
	coverURL := schreibeCoverDatei(t, "cover_auto_9780000000163.webp")

	mitCover, err := generateMahnPDF(mahnKlassenMit(coverURL))
	if err != nil {
		t.Fatalf("PDF-Erzeugung mit WebP-Cover schlug fehl: %v", err)
	}
	if !bytes.HasPrefix(mitCover, []byte("%PDF")) {
		t.Fatal("Ausgabe ist kein PDF")
	}

	// Das Cover muss auch tatsächlich drin sein — sonst würde ein stilles Weglassen
	// (der Fehlerpfad) diesen Test ebenfalls bestehen.
	ohneCover, err := generateMahnPDF(mahnKlassenMit(""))
	if err != nil {
		t.Fatalf("PDF-Erzeugung ohne Cover schlug fehl: %v", err)
	}
	if !bytes.Contains(mitCover, []byte("/Subtype /Image")) {
		t.Error("PDF enthält kein eingebettetes Bild — Cover wurde stillschweigend verworfen")
	}
	if len(mitCover) <= len(ohneCover) {
		t.Errorf("PDF mit Cover (%d B) ist nicht größer als ohne (%d B)", len(mitCover), len(ohneCover))
	}
}

// TestGenerateMahnPDF_DefektesCoverKostetNurDasBild hält die eigentliche Härtung fest:
// Egal was im uploads-Ordner liegt, die Mahnliste muss ausgeliefert werden.
func TestGenerateMahnPDF_DefektesCoverKostetNurDasBild(t *testing.T) {
	mahnPDFTestUmgebung(t)
	kaputt := filepath.Join(uploadsVerzeichnis, "cover_kaputt.webp")
	if err := os.WriteFile(kaputt, []byte("das ist kein bild"), 0o600); err != nil {
		t.Fatalf("setup: %v", err)
	}

	pdfBytes, err := generateMahnPDF(mahnKlassenMit("/" + uploadsVerzeichnis + "/cover_kaputt.webp"))
	if err != nil {
		t.Fatalf("defektes Cover ließ die PDF-Erzeugung scheitern: %v", err)
	}
	if !bytes.HasPrefix(pdfBytes, []byte("%PDF")) {
		t.Fatal("Ausgabe ist kein PDF")
	}
}

// TestGenerateMahnPDF_MehrfachesCoverWirdEinmalEingebettet: Bei mehreren überfälligen
// Exemplaren desselben Titels darf das Cover nicht je Zeile erneut im PDF landen.
func TestGenerateMahnPDF_MehrfachesCoverWirdEinmalEingebettet(t *testing.T) {
	mahnPDFTestUmgebung(t)
	coverURL := schreibeCoverDatei(t, "cover_doppelt.webp")

	klassen := mahnKlassenMit(coverURL)
	medium := klassen[0].Schueler[0].Medien[0]
	medium.Barcode = "B-0002"
	klassen[0].Schueler[0].Medien = append(klassen[0].Schueler[0].Medien, medium)

	zweiZeilen, err := generateMahnPDF(klassen)
	if err != nil {
		t.Fatalf("PDF-Erzeugung: %v", err)
	}
	eineZeile, err := generateMahnPDF(mahnKlassenMit(coverURL))
	if err != nil {
		t.Fatalf("PDF-Erzeugung: %v", err)
	}

	// Die zweite Zeile bringt Text und einen Barcode mit; das Cover selbst darf sie nicht
	// noch einmal kosten. Ein erneut eingebettetes Cover wäre in dieser Größenordnung
	// deutlich sichtbar, daher großzügige, aber wirksame Schranke.
	zuwachs := len(zweiZeilen) - len(eineZeile)
	if zuwachs > len(eineZeile)/2 {
		t.Errorf("zweite Zeile kostet %d B — Cover wurde vermutlich erneut eingebettet", zuwachs)
	}
}

func TestCoverDateiPfad(t *testing.T) {
	mahnPDFTestUmgebung(t)
	vorhanden := schreibeCoverDatei(t, "cover_ok.webp")

	// Eine Datei ausserhalb von uploads/, auf die ein Traversal zielen könnte.
	if err := os.WriteFile("geheim.txt", []byte("streng geheim"), 0o600); err != nil {
		t.Fatalf("setup: %v", err)
	}

	faelle := []struct {
		name     string
		coverURL string
		wantPfad string
	}{
		{"vorhandenes Cover", vorhanden, filepath.Join(uploadsVerzeichnis, "cover_ok.webp")},
		{"leere URL", "", ""},
		{"externe URL", "https://example.invalid/cover.jpg", ""},
		{"anderes Verzeichnis", "/etc/passwd", ""},
		{"Datei existiert nicht", "/uploads/gibtesnicht.webp", ""},
		{"Verzeichnis statt Datei", "/uploads/", ""},
		{"Traversal", "/uploads/../geheim.txt", ""},
		{"Traversal mit Umweg", "/uploads/unter/../../geheim.txt", ""},
	}
	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			if got := coverDateiPfad(f.coverURL); got != f.wantPfad {
				t.Errorf("coverDateiPfad(%q) = %q, erwartet %q", f.coverURL, got, f.wantPfad)
			}
		})
	}
}
