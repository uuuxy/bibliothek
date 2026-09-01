package api

// Tests am fertigen PDF-Inhaltsstrom (Technik wie die Etiketten-Gates): Der
// Eltern-Mahnbrief ist ein DIN-5008-Fensterkuvert-Brief — seit dem 01.09.2026
// steht die Anschrift aus der Schülerdatei im Fensterfeld (vorher hartkodiert
// „Adresse unbekannt", obwohl das Layout von Anfang an für den Postversand
// gebaut war). Den bestückten Fall über den Live-Pfad prüft das PII-Antwort-Gate
// (Positiv-Kontrolle auf /api/reports/overdue-pdf); hier steht der Gegenfall,
// der dort nicht abbildbar ist: OHNE Anschrift muss der Brief das ausdrücklich
// sagen — eine leere Zeile im Fenster sähe aus wie ein Druckfehler, so sieht
// die Sekretärin sofort, welcher Brief über die Klassenleitung gehen muss.

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/jung-kurt/gofpdf"
)

func renderElternMahnbrief(t *testing.T, student *OverdueStudent) string {
	t.Helper()
	doc := gofpdf.New("P", "mm", "A4", "")
	tr := doc.UnicodeTranslatorFromDescriptor("")
	zeichneElternMahnbrief(doc, tr, student, "Mahnung", "Bitte zurückgeben:\n{{.BuchListe}}", "Testschule")
	var buf bytes.Buffer
	if err := doc.Output(&buf); err != nil {
		t.Fatalf("PDF erzeugen: %v", err)
	}
	return pdfText(t, buf.Bytes())
}

func testOverdueStudent() *OverdueStudent {
	return &OverdueStudent{
		Vorname: "Mia", Nachname: "Musterkind",
		Books: []OverdueBook{{
			Titel: "Testband", BarcodeID: "BC-1",
			AusgeliehenAm: time.Now().AddDate(0, -2, 0),
			Frist:         time.Now().AddDate(0, -1, 0),
			DaysOverdue:   30,
		}},
	}
}

func TestElternMahnbriefDrucktAnschriftInsFensterfeld(t *testing.T) {
	student := testOverdueStudent()
	student.Strasse, student.Hausnummer = "Blumenweg", "7"
	student.PLZ, student.Ort = "61169", "Friedberg"

	text := renderElternMahnbrief(t, student)
	for _, soll := range []string{"Blumenweg 7", "61169 Friedberg", "Eltern von Mia Musterkind"} {
		if !strings.Contains(text, soll) {
			t.Errorf("Brief ohne %q — das Fensterkuvert bliebe leer", soll)
		}
	}
	if strings.Contains(text, "Adresse unbekannt") || strings.Contains(text, "keine Adresse hinterlegt") {
		t.Error("Brief trägt trotz vollständiger Anschrift einen Fehlt-Vermerk")
	}
}

func TestElternMahnbriefOhneAnschriftSagtEsAusdruecklich(t *testing.T) {
	text := renderElternMahnbrief(t, testOverdueStudent())
	// Ohne Klammern gesucht: Im PDF-Inhaltsstrom stehen Klammern escaped
	// (`\(…\)`), der Wortlaut dazwischen bleibt unverändert.
	if !strings.Contains(text, "keine Adresse hinterlegt") {
		t.Error("Brief ohne Anschrift muss '(keine Adresse hinterlegt)' ins Fensterfeld drucken — " +
			"eine leere Zeile ist von einem Druckfehler nicht zu unterscheiden")
	}
}
