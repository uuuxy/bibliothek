package portal

import (
	"strings"
	"testing"

	"bibliothek/internal/repository"
)

// Die Bereit-Mail an die Lehrkraft und die Meldung „Nur … im Bestand" schrieben die Mehrzahl
// fest; bei einem einzelnen Exemplar stand „1 Exemplare" da.
func TestKlassensatz_EinExemplarStehtInDerEinzahl(t *testing.T) {
	text := klassensatzBereitText(&repository.ErledigteReservierung{TitelName: "Tschick", Klasse: "08G1", Anzahl: 1})
	if !strings.Contains(text, "Anzahl: 1 Exemplar\n") {
		t.Errorf("Bereit-Mail für ein Exemplar: %q", text)
	}
	text = klassensatzBereitText(&repository.ErledigteReservierung{TitelName: "Tschick", Klasse: "08G1", Anzahl: 28})
	if !strings.Contains(text, "Anzahl: 28 Exemplare\n") {
		t.Errorf("Bereit-Mail für 28 Exemplare: %q", text)
	}
	for n, soll := range map[int]string{0: "0 Exemplare", 1: "1 Exemplar", 2: "2 Exemplare"} {
		if ist := exemplarText(n); ist != soll {
			t.Errorf("exemplarText(%d) = %q, erwartet %q", n, ist, soll)
		}
	}
}

// Der Fuß der Bereit-Mail schreibt „Abschließen“ mit ß, wie jeder Text an die Lehrkräfte.
func TestKlassensatz_BereitMailNenntDenAnlass(t *testing.T) {
	text := klassensatzBereitText(&repository.ErledigteReservierung{TitelName: "Tschick", Klasse: "08G1", Anzahl: 28})
	if soll := "Diese Mail wurde automatisch beim Abschließen der Reservierung verschickt."; !strings.HasSuffix(text, soll) {
		t.Errorf("Bereit-Mail endet mit %q, erwartet %q", text[strings.LastIndex(text, "\n")+1:], soll)
	}
}
