package api

import (
	"testing"
)

func TestPDFGeneration(t *testing.T) {
	items := []OrderedItem{
		{Titel: "Test Buch 1", Autor: "Autor 1", ISBN: "123-456", Menge: 5},
		{Titel: "Test Buch 2", Autor: "Autor 2", ISBN: "789-012", Menge: 2},
	}

	summaryPDF, err := GenerateOrderSummaryPDF(items)
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
