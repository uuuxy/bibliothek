package api

import (
	"bytes"
	"compress/zlib"
	"io"
	"strings"
	"testing"
)

// Geprüft wird am ENTPACKTEN PDF-Inhaltsstrom, nicht an den Eingabedaten.
//
// Ein Test, der nur „die Funktion bekam Max Mustermann" behauptet, sagt nichts darüber,
// ob der Name auf dem Papier landet — genau daran ist beim Lernmittel-Etikett schon
// einmal ein Weg vorbeigelaufen, der nur drei von sechs Feldern mitgeschickt hat.
// Dieselbe Technik wie in etiketten_pdf_paritaet_pg_test.go.

// pdfText packt die FlateDecode-Ströme eines PDFs aus und liefert ihren Inhalt.
func pdfText(t *testing.T, roh []byte) string {
	t.Helper()
	var text strings.Builder
	text.Write(roh) // unkomprimierte Anteile (kleine Ströme lässt gofpdf offen)

	rest := roh
	for {
		i := bytes.Index(rest, []byte("stream"))
		if i < 0 {
			break
		}
		nach := rest[i+len("stream"):]
		nach = bytes.TrimLeft(nach, "\r\n")
		j := bytes.Index(nach, []byte("endstream"))
		if j < 0 {
			break
		}
		if zr, err := zlib.NewReader(bytes.NewReader(nach[:j])); err == nil {
			if entpackt, err := io.ReadAll(zr); err == nil {
				text.Write(entpackt)
			}
			zr.Close() //nolint:errcheck // Lesende zlib-Ströme im Test
		}
		rest = nach[j:]
	}
	return text.String()
}

func erzeugeBogen(t *testing.T, format string, start int, etiketten []SchuelerEtikett) string {
	t.Helper()
	pdf, err := GenerateSchuelerEtikettenPDF(format, start, etiketten)
	if err != nil {
		t.Fatalf("PDF-Erzeugung fehlgeschlagen: %v", err)
	}
	var puffer bytes.Buffer
	if err := pdf.Output(&puffer); err != nil {
		t.Fatalf("PDF-Ausgabe fehlgeschlagen: %v", err)
	}
	if !bytes.HasPrefix(puffer.Bytes(), []byte("%PDF-")) {
		t.Fatal("Ergebnis ist kein PDF")
	}
	return pdfText(t, puffer.Bytes())
}

func TestSchuelerEtikettTraegtNameKlasseUndBarcode(t *testing.T) {
	// Die drei Angaben, auf die der Betreiber das Etikett festgelegt hat (24.08.2026).
	// Fällt eine davon beim Zeichnen weg, ist der Bogen wertlos — und am Bildschirm
	// sieht man es nicht, weil das PDF erst im Druckdialog aufgeht.
	for _, format := range labelFormatReihenfolge {
		t.Run(format, func(t *testing.T) {
			text := erzeugeBogen(t, format, 1, []SchuelerEtikett{
				{BarcodeID: "S-000123", Vorname: "Max", Nachname: "Mustermann", Klasse: "8G2"},
			})

			for _, erwartet := range []string{"Mustermann, Max", "8G2", "S-000123"} {
				if !strings.Contains(text, erwartet) {
					t.Errorf("%q steht nicht auf dem Etikett", erwartet)
				}
			}
		})
	}
}

func TestSchuelerEtikettDrucktNamenAusserhalbVonCp1252(t *testing.T) {
	// Der PDF-Zeichensatz ist cp1252; „ş" wurde darin zu einem Punkt, „Ayşe" kam als
	// „Ay.e" aus dem Drucker. An einer hessischen Schule ist das kein Randfall.
	// Ein Häkchen weniger ist hinnehmbar, ein entstellter Name nicht.
	text := erzeugeBogen(t, "zweckform_l4760", 1, []SchuelerEtikett{
		{BarcodeID: "S-000200", Vorname: "Ayşe", Nachname: "Öztürk", Klasse: "5a"},
		{BarcodeID: "S-000201", Vorname: "Łukasz", Nachname: "Wiśniewski", Klasse: "9R2"},
	})

	// Verglichen wird nur mit ASCII-Ausschnitten: Im PDF-Strom stehen ä/ö/ü/Ö als
	// EINZELNE cp1252-Bytes, ein UTF-8-Literal wie "Öztürk" würde dort nie passen und
	// der Test wäre immer rot — aus dem falschen Grund. Die Aussage steckt ohnehin in
	// den Buchstaben, die ersetzt werden mussten.
	for _, erwartet := range []string{", Ayse", "Wisniewski, Lukasz"} {
		if !strings.Contains(text, erwartet) {
			t.Errorf("%q steht nicht auf dem Etikett", erwartet)
		}
	}
	// Der Punkt ist das Zeichen, das cp1252 für Unbekanntes einsetzt.
	for _, verboten := range []string{"Ay.e", "W.sniewski", ".ukasz"} {
		if strings.Contains(text, verboten) {
			t.Errorf("entstellter Name auf dem Etikett: %q", verboten)
		}
	}
}

func TestSchuelerEtikettOhneKlasseLaesstDieZeileWeg(t *testing.T) {
	// Handanlage ohne Klassenangabe und Abgänger haben keine Klasse. „Klasse " ins
	// Nichts zu schreiben wäre schlechter als die Zeile wegzulassen.
	text := erzeugeBogen(t, "zweckform_l4760", 1, []SchuelerEtikett{
		{BarcodeID: "S-000999", Vorname: "Erika", Nachname: "Ohneklasse"},
	})

	if !strings.Contains(text, "Ohneklasse, Erika") {
		t.Error("der Name fehlt auf dem Etikett")
	}
	if strings.Contains(text, "Klasse ") {
		t.Error("ohne Klasse darf keine Klassenzeile gedruckt werden")
	}
}

func TestSchuelerEtikettKuerztNachBreiteUndVerschontDieKlasse(t *testing.T) {
	// Gekürzt wird nach gemessener Breite, nicht nach Zeichenzahl. Belegt an zwei
	// Fällen, die eine Zeichengrenze beide falsch behandelt:
	//
	//  1. Ein kurzer Name muss VOLLSTÄNDIG dastehen — auch auf dem kleinen Format,
	//     wo vorher stur bei 16 Zeichen abgeschnitten wurde.
	//  2. Die Klasse überlebt die Kürzung. Sie stand auf dem kleinen Format als
	//     "LIT-A…" da, obwohl rechts Platz frei war; eine halbe Klasse ist wertlos.
	for _, format := range []string{"zweckform_l4760", "standard_52"} {
		t.Run(format, func(t *testing.T) {
			text := erzeugeBogen(t, format, 1, []SchuelerEtikett{
				{BarcodeID: "S-000300", Vorname: "Amal", Nachname: "Abardouch", Klasse: "LIT-ALT"},
			})
			if !strings.Contains(text, "Abardouch, Amal") {
				t.Error("der kurze Name steht nicht vollständig auf dem Etikett")
			}
			if !strings.Contains(text, "LIT-ALT") {
				t.Error("die Klasse wurde gekürzt, obwohl sie hinpasst")
			}
		})
	}

	// Gegenprobe: Ein Name, der NICHT passt, muss gekürzt werden — sonst liefe er
	// über den Rand des Klebefelds, und die Breitenmessung waere wirkungslos.
	lang := erzeugeBogen(t, "standard_52", 1, []SchuelerEtikett{
		{BarcodeID: "S-000301", Vorname: "Maximiliane-Charlotte", Nachname: "Schneider-Weisshaupt", Klasse: "10R1"},
	})
	if strings.Contains(lang, "Schneider-Weisshaupt, Maximiliane-Charlotte") {
		t.Error("der ueberlange Name wurde nicht gekuerzt")
	}
	if !strings.Contains(lang, "10R1") {
		t.Error("die Klasse fiel der Kuerzung zum Opfer")
	}
}

func TestSchuelerEtikettenBogenBrichtNachEinerSeiteUm(t *testing.T) {
	// zweckform_l4760 fasst 21 Etiketten. Mit Startposition 20 passen genau zwei auf
	// die erste Seite, der Rest gehört auf die zweite. Die Rechnung dazu steht in
	// zeichneRaster und wird von Buch- UND Schüler-Etiketten benutzt — ein Fehler
	// darin verdruckt beide Bogensorten.
	etiketten := make([]SchuelerEtikett, 0, 5)
	for _, name := range []string{"Anders", "Berger", "Conrad", "Dorn", "Engel"} {
		etiketten = append(etiketten, SchuelerEtikett{BarcodeID: "S-" + name, Nachname: name, Vorname: "Test", Klasse: "5a"})
	}

	pdf, err := GenerateSchuelerEtikettenPDF("zweckform_l4760", 20, etiketten)
	if err != nil {
		t.Fatalf("PDF-Erzeugung fehlgeschlagen: %v", err)
	}
	if seiten := pdf.PageCount(); seiten != 2 {
		t.Errorf("erwartet 2 Seiten (2 auf der ersten, 3 auf der zweiten), erzeugt: %d", seiten)
	}
}

func TestMusterEtikettIstDasselbeEtikettWieDasEchte(t *testing.T) {
	// Der Testdruck des Designers darf kein zweiter Renderer sein. Genau daran sind im
	// Ausweis-Designer schon einmal Leinwand und Druck auseinandergelaufen.
	text := erzeugeBogen(t, "zweckform_l4760", 1, []SchuelerEtikett{MusterSchuelerEtikett})

	for _, erwartet := range []string{"Mustermann, Max", "8G2", "DEMO-S-001"} {
		if !strings.Contains(text, erwartet) {
			t.Errorf("%q fehlt auf dem Muster-Etikett", erwartet)
		}
	}
}
