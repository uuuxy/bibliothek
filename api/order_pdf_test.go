package api

import (
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
		{BarcodeID: "B-10001", Titel: "Test Buch 1", Autor: "Autor 1"},
		{BarcodeID: "B-10002", Titel: "Test Buch 2", Autor: "Autor 2"},
	}

	barcodePDF, err := GenerateBarcodeSheetPDF(labels)
	if err != nil {
		t.Fatalf("Failed to generate barcode PDF: %v", err)
	}
	if len(barcodePDF) == 0 {
		t.Error("Generated barcode PDF is empty")
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
