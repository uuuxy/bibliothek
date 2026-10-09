package pdf

import (
	"bytes"
	"strings"
	"testing"

	"bibliothek/internal/pdftest"
)

func TestGenerateLabelsPDF(t *testing.T) {
	labels := []BuchEtikett{
		{Schulname: "Testbibliothek", BarcodeID: "B-10001", Titel: "Test Buch 1", Autor: "Autor 1", Signatur: "LMF-Deutsch 5", Eigentumsvermerk: "Eigentum des Landes Hessen"},
		{Schulname: "Testbibliothek", BarcodeID: "B-10002", Titel: "Test Buch 2", Autor: "Autor 2", Eigentumsvermerk: "Eigentum des Landes Hessen"},
	}

	labelDoc, err := GenerateLabelsPDF("zweckform_l4760", 1, false, labels)
	if err != nil {
		t.Fatalf("Failed to generate label PDF: %v", err)
	}
	var buf bytes.Buffer
	if err := labelDoc.Output(&buf); err != nil {
		t.Fatalf("Failed to output label PDF: %v", err)
	}
	if buf.Len() == 0 {
		t.Error("Generated label PDF is empty")
	}
}

// Naacher & Co. bekommen zusätzlich zum kleinen das große Lernmittel-Etikett
// mitgeschickt, damit sie selbst wählen können, welches sie drucken.
func TestGenerateLernmittelEtikettenPDF(t *testing.T) {
	items := []BuchEtikett{
		{Schulname: "Testbibliothek", BarcodeID: "B-10001", Titel: "Test Buch 1", AnschaffungsJahr: "2024", Signatur: "LMF-Deutsch 5", Eigentumsvermerk: "Eigentum des Landes Hessen"},
		{Schulname: "Testbibliothek", BarcodeID: "B-10002", Titel: "Test Buch 2", Eigentumsvermerk: "Eigentum des Landes Hessen"},
	}

	pdfBytes, err := GenerateLernmittelEtikettenPDF(items)
	if err != nil {
		t.Fatalf("Failed to generate Lernmittel-Etikett PDF: %v", err)
	}
	if len(pdfBytes) == 0 {
		t.Error("Generated Lernmittel-Etikett PDF is empty")
	}
}

func TestZweiteZeile(t *testing.T) {
	cases := []struct {
		jahr, signatur, want string
	}{
		{"2016", "LMF-Deutsch 5", "Ansch.J. 2016 · LMF-Deutsch 5"},
		{"2016", "", "Ansch.J. 2016"},
		{"", "LMF-Deutsch 5", "LMF-Deutsch 5"},
		{"", "", ""},
	}
	for _, c := range cases {
		if got := zweiteZeile(c.jahr, c.signatur); got != c.want {
			t.Errorf("zweiteZeile(%q, %q) = %q, want %q", c.jahr, c.signatur, got, c.want)
		}
	}
}

// gedruckteZeilen liefert die gedruckten Textstücke, die das Wort tragen, in der Reihenfolge
// des Blatts.
func gedruckteZeilen(t *testing.T, roh []byte, wort string) []string {
	t.Helper()
	var zeilen []string
	for _, text := range pdftest.TexteInReihenfolge(t, roh) {
		if strings.Contains(text, wort) {
			zeilen = append(zeilen, text)
		}
	}
	return zeilen
}

// Der Erzeuger kennt weder Einstellungen noch den Topf: Er druckt Schulname und Vermerk, die
// das Etikett trägt, und ohne Vermerk keinen. Auf dem kleinen Format unter 30 mm hat die
// Zeile des Vermerks keinen Platz.
func TestBuchEtikett_DrucktSchulnameUndVermerkDesEtiketts(t *testing.T) {
	items := []BuchEtikett{
		{Schulname: "Schule Eins", BarcodeID: "B-1", Titel: "Eins", Eigentumsvermerk: "Vermerk Eins"},
		{Schulname: "Schule Zwei", BarcodeID: "B-2", Titel: "Zwei", Eigentumsvermerk: "Vermerk Zwei"},
		{Schulname: "Schule Drei", BarcodeID: "B-3", Titel: "Drei"},
	}
	bogen := func(format string) []byte {
		t.Helper()
		doc, err := GenerateLabelsPDF(format, 1, false, items)
		if err != nil {
			t.Fatalf("Bogen %s: %v", format, err)
		}
		var buf bytes.Buffer
		if err := doc.Output(&buf); err != nil {
			t.Fatalf("Bogen %s ausgeben: %v", format, err)
		}
		return buf.Bytes()
	}
	gross, err := GenerateLernmittelEtikettenPDF(items)
	if err != nil {
		t.Fatalf("großes Etikett: %v", err)
	}

	schulen := []string{"Schule Eins", "Schule Zwei", "Schule Drei"}
	vermerke := []string{"Vermerk Eins", "Vermerk Zwei"}
	for name, roh := range map[string][]byte{"Bogen ab 30 mm": bogen("zweckform_l4760"), "großes Etikett": gross} {
		if ist := gedruckteZeilen(t, roh, "Schule"); strings.Join(ist, "|") != strings.Join(schulen, "|") {
			t.Errorf("%s: gedruckte Schulnamen %q, erwartet %q", name, ist, schulen)
		}
		if ist := gedruckteZeilen(t, roh, "Vermerk"); strings.Join(ist, "|") != strings.Join(vermerke, "|") {
			t.Errorf("%s: gedruckte Vermerke %q, erwartet %q", name, ist, vermerke)
		}
	}
	klein := bogen("standard_52")
	if ist := gedruckteZeilen(t, klein, "Schule"); strings.Join(ist, "|") != strings.Join(schulen, "|") {
		t.Errorf("kleines Format: gedruckte Schulnamen %q, erwartet %q", ist, schulen)
	}
	if ist := gedruckteZeilen(t, klein, "Vermerk"); len(ist) != 0 {
		t.Errorf("kleines Format: gedruckte Vermerke %q, erwartet keinen", ist)
	}
}
