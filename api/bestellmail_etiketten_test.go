package api

// Eigentumsvermerk auf den gemailten Etikettenbögen (Sweep 01.09.2026, Fund P2):
// Bis dahin nagelte der Mailweg den Vermerk auf die Werksvorgabe „Eigentum des
// Landes Hessen" fest, während Selbstdruck (s.etikettKopf) und Lieferanten-Link
// die Einstellung etikett_eigentumsvermerk lasen — zwei Wege zum selben Buch,
// zwei verschiedene Aufkleber. Geprüft am fertigen PDF-Inhaltsstrom (pdfText),
// wie alle Etiketten-Gates.

import (
	"strings"
	"testing"

	"bibliothek/pdf"
	"bibliothek/repository"
)

func testEtikettLabels() []BarcodeLabelDetail {
	return []BarcodeLabelDetail{{
		BarcodeID: "BM-TEST-1", Titel: "Bestellmail-Testband", Autor: "Prüfer", ISBN: "9783123456789",
	}}
}

func TestBestellmailEtikettenTragenKonfiguriertenVermerk(t *testing.T) {
	boegen, err := etikettenboegen(testEtikettLabels(), pdf.SchuleInfo{Name: "Testschule"}, false,
		"Eigentum des Kreises Wetterau")
	if err != nil {
		t.Fatalf("etikettenboegen: %v", err)
	}
	if len(boegen) == 0 {
		t.Fatal("kein Bogen erzeugt")
	}
	for _, b := range boegen {
		text := pdfText(t, b.Data)
		if !strings.Contains(text, "Eigentum des Kreises Wetterau") {
			t.Errorf("%s: konfigurierter Eigentumsvermerk fehlt auf dem Bogen", b.Name)
		}
		if strings.Contains(text, repository.StandardEigentumsvermerk) {
			t.Errorf("%s: Werksvorgabe steht trotz konfiguriertem Vermerk auf dem Bogen", b.Name)
		}
	}
}

func TestBestellmailEtikettenFallenOhneKonfigurationAufWerksvorgabe(t *testing.T) {
	boegen, err := etikettenboegen(testEtikettLabels(), pdf.SchuleInfo{Name: "Testschule"}, false, "")
	if err != nil {
		t.Fatalf("etikettenboegen: %v", err)
	}
	for _, b := range boegen {
		if !strings.Contains(pdfText(t, b.Data), repository.StandardEigentumsvermerk) {
			t.Errorf("%s: ohne Konfiguration muss die Werksvorgabe drucken", b.Name)
		}
	}
}
