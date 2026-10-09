package api

import (
	"strings"
	"testing"

	"bibliothek/pdf"
	"bibliothek/repository"
)

func TestPDFGeneration(t *testing.T) {
	items := []OrderedItem{
		{Titel: "Test Buch 1", Autor: "Autor 1", ISBN: "123-456", Menge: 5},
		{Titel: "Test Buch 2", Autor: "Autor 2", ISBN: "789-012", Menge: 2},
	}

	summaryPDF, err := GenerateOrderSummaryPDF(items, pdf.SchuleInfo{Name: "Testbibliothek"}, bogenLiegtBei, repository.MittelLand)
	if err != nil {
		t.Fatalf("Failed to generate summary PDF: %v", err)
	}
	if len(summaryPDF) == 0 {
		t.Error("Generated summary PDF is empty")
	}
}

// Das Anschreiben versprach dem Lieferanten bedingungslos einen "beigefügten Bogen" mit
// Barcode-Aufklebern — auch dann, wenn der E-Mail gar keiner beilag. Der Lieferant kann
// eine solche Anweisung nur ignorieren oder nachfragen; beides kostet die Lieferung Zeit.
func TestBestellAnschreibenNenntBarcodebogenNurWennErBeiliegt(t *testing.T) {
	land := mittelTexte[repository.MittelLand]
	mit := bestellAnschreibenText(bogenLiegtBei, land)
	if !strings.Contains(mit, barcodebogenSatz) {
		t.Error("Mit Bogen: Der Hinweis auf die Aufkleber fehlt im Anschreiben")
	}

	ohne := bestellAnschreibenText(ohneEtiketten, land)
	if strings.Contains(ohne, barcodebogenSatz) {
		t.Error("Ohne Bogen: Das Anschreiben verweist auf eine Anlage, die nicht existiert")
	}

	// Dritter Fall, seit die Etiketten hinter dem Bestätigungs-Link liegen: Der Brief muss
	// dann den LINK nennen. Bliebe hier der Satz vom "beigefügten Bogen" stehen, suchte der
	// Händler eine Anlage, die es nicht gibt — und der Link, der die Bestätigung trägt,
	// bliebe ungeklickt.
	ueberLink := bestellAnschreibenText(bogenHinterLink, land)
	if strings.Contains(ueberLink, barcodebogenSatz) {
		t.Error("Bogen hinter dem Link: Das Anschreiben verweist trotzdem auf eine beigefügte Anlage")
	}
	if !strings.Contains(ueberLink, "Link in dieser E-Mail") {
		t.Error("Bogen hinter dem Link: Das Anschreiben sagt nicht, wo der Händler die Etiketten bekommt")
	}

	// Der Rest des Briefes bleibt in allen Fällen gleich — es faellt genau ein Satz weg.
	for _, satz := range []string{"Sehr geehrte Damen und Herren", "Die Rechnung senden Sie bitte", "Bestellte Titel:"} {
		if !strings.Contains(ohne, satz) {
			t.Errorf("Ohne Bogen: %q fehlt im Anschreiben", satz)
		}
	}
}
