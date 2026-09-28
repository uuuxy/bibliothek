package service

import (
	"testing"

	"bibliothek/internal/littera"
)

// Die Übernahme schreibt den EAN-13 vom Littera-Etikett nach barcode_id
// (littera.EtikettBarcode), der Scanner rechnet einen gescannten EAN-13 auf die Nummer zurück
// (dekodiereLitteraEtikett). Beide beschreiben dasselbe Etikett und müssen deshalb dieselbe
// Form kennen.
//
// Bis zum 28.09.2026 gab es diesen Test nicht, und die beiden rechneten verschieden: Die
// Übernahme füllte links auf und setzte an Stelle 12 fest eine 6, der Scanner (nach zwei
// echten Scans) rechts auf und las an Stelle 12 die Stellenzahl. Für jede Nummer unter sechs
// Stellen schrieb die Übernahme damit einen Barcode, den kein Etikett trägt. Der
// Zwilling-Test daneben band nur den Scanner an die Theke ohne Netz.
//
// Blindheit: Der Test prüft, dass die zwei Rechnungen zueinander passen, nicht, dass sie zum
// Etikett passen; das tun die Messwerte in internal/littera/barcode_test.go.
func TestLitteraEtikett_UebernahmeUndScannerRechnenGleich(t *testing.T) {
	for _, nummer := range []string{"8", "80", "808", "8080", "58968", "124117", "1234567"} {
		ean, ok := littera.EtikettBarcode(nummer, "395")
		if !ok {
			t.Errorf("Übernahme bildet für %q keinen Barcode", nummer)
			continue
		}
		zurueck, ok := dekodiereLitteraEtikett(ean)
		if !ok || zurueck != nummer {
			t.Errorf("Übernahme schreibt für %q den Barcode %s, der Scanner liest daraus %q (%v)",
				nummer, ean, zurueck, ok)
		}
	}
}
