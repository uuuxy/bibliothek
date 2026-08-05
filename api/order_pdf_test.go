package api

import (
	"bytes"
	"strings"
	"testing"

	"bibliothek/pdf"
)

func TestPDFGeneration(t *testing.T) {
	items := []OrderedItem{
		{Titel: "Test Buch 1", Autor: "Autor 1", ISBN: "123-456", Menge: 5},
		{Titel: "Test Buch 2", Autor: "Autor 2", ISBN: "789-012", Menge: 2},
	}

	summaryPDF, err := GenerateOrderSummaryPDF(items, pdf.SchuleInfo{Name: "Testbibliothek"}, true)
	if err != nil {
		t.Fatalf("Failed to generate summary PDF: %v", err)
	}
	if len(summaryPDF) == 0 {
		t.Error("Generated summary PDF is empty")
	}

	labels := []BarcodeLabelDetail{
		{BarcodeID: "B-10001", Titel: "Test Buch 1", Autor: "Autor 1", Signatur: "LMF-Deutsch 5"},
		{BarcodeID: "B-10002", Titel: "Test Buch 2", Autor: "Autor 2"},
	}

	kopf := EtikettKopf{Schulname: "Testbibliothek", Eigentumsvermerk: "Eigentum des Landes Hessen"}
	labelDoc, err := GenerateLabelsPDF("zweckform_l4760", 1, false, labels, kopf)
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
	items := []BarcodeLabelDetail{
		{BarcodeID: "B-10001", Titel: "Test Buch 1", AnschaffungsJahr: "2024", Signatur: "LMF-Deutsch 5"},
		{BarcodeID: "B-10002", Titel: "Test Buch 2"},
	}
	kopf := EtikettKopf{Schulname: "Testbibliothek", Eigentumsvermerk: "Eigentum des Landes Hessen"}

	pdfBytes, err := GenerateLernmittelEtikettenPDF(items, kopf)
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

// Das Anschreiben versprach dem Lieferanten bedingungslos einen "beigefügten Bogen" mit
// Barcode-Aufklebern — auch dann, wenn der E-Mail gar keiner beilag. Der Lieferant kann
// eine solche Anweisung nur ignorieren oder nachfragen; beides kostet die Lieferung Zeit.
func TestBestellAnschreibenNenntBarcodebogenNurWennErBeiliegt(t *testing.T) {
	mit := bestellAnschreibenText(true)
	if !strings.Contains(mit, barcodebogenSatz) {
		t.Error("Mit Bogen: Der Hinweis auf die Aufkleber fehlt im Anschreiben")
	}

	ohne := bestellAnschreibenText(false)
	if strings.Contains(ohne, barcodebogenSatz) {
		t.Error("Ohne Bogen: Das Anschreiben verweist auf eine Anlage, die nicht existiert")
	}

	// Der Rest des Briefes bleibt in beiden Fällen gleich — es faellt genau ein Satz weg.
	for _, satz := range []string{"Sehr geehrte Damen und Herren", "Die Rechnung senden Sie bitte", "Bestellte Titel:"} {
		if !strings.Contains(ohne, satz) {
			t.Errorf("Ohne Bogen: %q fehlt im Anschreiben", satz)
		}
	}
}
