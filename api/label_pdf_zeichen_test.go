package api

import (
	"bytes"
	"strings"
	"testing"
)

// Die Buchetiketten kannten die Zeichenersetzung bis zum 21.09.2026 nicht: Ein Titel
// wie „Şafak und Łukasz" kam als „.afak und .ukasz" vom Drucker, während das Schüler-
// Etikett daneben längst richtig druckte (OFFEN.md 5.5). Seit pdfzeichen läuft jeder
// Renderer durch dieselbe Ersetzung; dieser Test hält das für die beiden Buchetiketten
// fest — am fertigen PDF, nicht am Aufruf (drei Wege, ein Inhaltsstrom).
func TestBuchEtikettenDruckenZeichenAusserhalbVonCp1252(t *testing.T) {
	items := []BarcodeLabelDetail{{
		BarcodeID: "100000000001", Titel: "Şafak, Łukasz und Wiśniewski",
		Signatur: "Lit 5", AnschaffungsJahr: "2026",
	}}

	klein, err := GenerateLabelsPDF("zweckform_l4760", 1, false, items, layoutKopf)
	if err != nil {
		t.Fatalf("Buchetiketten: %v", err)
	}
	var puffer bytes.Buffer
	if err := klein.Output(&puffer); err != nil {
		t.Fatalf("PDF-Ausgabe: %v", err)
	}
	gross, err := GenerateLernmittelEtikettenPDF(items, layoutKopf)
	if err != nil {
		t.Fatalf("Lernmittel-Etiketten: %v", err)
	}

	for name, roh := range map[string][]byte{"Buchetikett": puffer.Bytes(), "Lernmittel-Etikett": gross} {
		text := pdfText(t, roh)
		// Der Titel ist nach der Ersetzung reines ASCII — so ist der Vergleich unabhängig
		// davon, wie der Strom Umlaute kodiert.
		if !strings.Contains(text, "Safak, Lukasz und Wisniewski") {
			t.Errorf("%s: der ersetzte Titel steht nicht auf dem Etikett", name)
		}
		// Der Punkt ist das Zeichen, das cp1252 für Unbekanntes einsetzt.
		for _, verboten := range []string{".afak", ".ukasz", "W.sniewski"} {
			if strings.Contains(text, verboten) {
				t.Errorf("%s: entstellter Titel auf dem Etikett: %q", name, verboten)
			}
		}
	}
}
