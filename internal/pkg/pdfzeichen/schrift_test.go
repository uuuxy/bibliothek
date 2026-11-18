package pdfzeichen_test

import (
	"bytes"
	"io"
	"strings"
	"testing"
	"unicode/utf8"

	"bibliothek/internal/pdftest"
	"bibliothek/internal/pkg/pdfzeichen"
)

// Jeder Schnitt druckt die Buchstaben, die die Standardschriften des PDF nicht kennen. Dass keine
// Standardschrift im Dokument steht, prüft der Leser in pdftest bei jedem PDF.
func TestNewDokument_DrucktJedenBuchstaben(t *testing.T) {
	const namen = "Ayşe Łukasz Doǧan Nguyễn Дмитрий Ελένη"
	// Jeder Schnitt, den ein Erzeuger setzt; fehlt einer, bricht das Dokument mit „undefined font" ab.
	schnitte := []struct{ familie, stil string }{
		{pdfzeichen.Schrift, ""}, {pdfzeichen.Schrift, "B"}, {pdfzeichen.Schrift, "I"},
		{pdfzeichen.SchriftSerif, ""}, {pdfzeichen.SchriftSerif, "B"},
		{pdfzeichen.SchriftFest, ""}, {pdfzeichen.SchriftFest, "B"},
	}
	p := pdfzeichen.NewDokument()
	p.AddPage()
	for _, s := range schnitte {
		p.SetFont(s.familie, s.stil, 10)
		p.CellFormat(0, 6, pdfzeichen.Ersetze(namen), "", 1, "L", false, 0, "")
	}
	texte := strings.Join(pdftest.Texte(t, ausgabe(t, p.Output)), "\n")
	if n, soll := strings.Count(texte, namen), len(schnitte); n != soll {
		t.Errorf("%q steht %d-mal auf dem Blatt, erwartet %d-mal (je Schnitt einmal):\n%s", namen, n, soll, texte)
	}
}

// Ein Emoji im Titel kostet ein Zeichen, nicht das Dokument.
func TestNewDokument_EmojiBrichtNichtsAb(t *testing.T) {
	p := pdfzeichen.NewDokument()
	p.AddPage()
	p.SetFont(pdfzeichen.Schrift, "", 10)
	p.MultiCell(60, 5, pdfzeichen.Ersetze("Buch "+string(rune(0x1F600))+" Titel"), "", "L", false)
	texte := pdftest.Texte(t, ausgabe(t, p.Output))
	if soll := "Buch " + string(utf8.RuneError) + " Titel"; len(texte) != 1 || texte[0] != soll {
		t.Errorf("gedruckt %q, erwartet %q", texte, soll)
	}
}

func ausgabe(t *testing.T, schreibe func(io.Writer) error) []byte {
	t.Helper()
	var roh bytes.Buffer
	if err := schreibe(&roh); err != nil {
		t.Fatalf("Dokument nicht geschrieben: %v", err)
	}
	return roh.Bytes()
}
