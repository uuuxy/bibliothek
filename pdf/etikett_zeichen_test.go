package pdf

import (
	"bytes"
	"strings"
	"testing"
)

// Die Etiketten der Bücher laufen durch die Zeichenersetzung (pdfzeichen) wie jeder andere
// Erzeuger; ohne sie käme ein Titel wie „Şafak und Łukasz" als „.afak und .ukasz" vom Drucker.
// Geprüft wird am fertigen PDF, nicht am Aufruf (drei Wege, ein Inhaltsstrom).
func TestBuchEtikettenDruckenZeichenAusserhalbVonCp1252(t *testing.T) {
	etiketten := []BuchEtikett{{
		Schulname: "Philipp-Reis-Schule, Friedrichsdorf",
		BarcodeID: "100000000001", Titel: "Şafak, Łukasz und Wiśniewski",
		Signatur: "Lit 5", AnschaffungsJahr: "2026", Eigentumsvermerk: "Eigentum des Landes Hessen",
	}}

	klein, err := ErzeugeBuchEtiketten("zweckform_l4760", 1, false, etiketten)
	if err != nil {
		t.Fatalf("Buchetiketten: %v", err)
	}
	var puffer bytes.Buffer
	if err := klein.Output(&puffer); err != nil {
		t.Fatalf("PDF-Ausgabe: %v", err)
	}
	gross, err := ErzeugeLernmittelEtiketten(etiketten)
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
