package pdf

import (
	"strings"
	"testing"
	"time"

	"bibliothek/internal/pdftest"
)

// Gemessen wird am fertigen PDF, nicht am Struct: Ein Feld kann im Struct stehen und trotzdem
// nicht gezeichnet werden. Hier geht es um Geld, das an die richtige Kasse muss.
//
// Kein Brief verlangt „bar in der Bibliothek": Die Arbeitshilfe zum Erlass vom 17.12.2014 sagt
// für die Lernmittel des Landes, es dürfe „keine Bargeldannahme vorgesehen werden".

const barsatz = "bar in der Bibliothek"

func blattText(t *testing.T, roh []byte) string {
	t.Helper()
	return strings.Join(pdftest.Texte(t, roh), " ")
}

func TestRechnungLernmittelNenntDasKontoDesLandes(t *testing.T) {
	positionen := []RechnungsPosition{
		{Titel: "Deutschbuch 7", Barcode: "B-1", Ausleihdatum: time.Now(), Ersatzpreis: 1990, Land: true},
	}
	got, err := ErzeugeRechnung(Schueler{Vorname: "Lena", Nachname: "Land"}, positionen, testSchule(), testZahlung())
	if err != nil {
		t.Fatalf("ErzeugeRechnung: %v", err)
	}
	blatt := blattText(t, got)

	if !strings.Contains(blatt, "HCC-Schulbereich") {
		t.Errorf("auf dem Blatt steht die Zahlstelle des Landes nicht:\n%s", blatt)
	}
	if !strings.Contains(blatt, "DE86500500000001002401") {
		t.Errorf("auf dem Blatt steht die Bankverbindung nicht:\n%s", blatt)
	}
	if strings.Contains(blatt, barsatz) {
		t.Errorf("die Rechnung verlangt weiterhin Barzahlung — genau das untersagt die Arbeitshilfe:\n%s", blatt)
	}
	// Kein Wort vom Schulträger: Der Brief trägt nur Lernmittel.
	if strings.Contains(blatt, "Schultr") {
		t.Errorf("reiner Lernmittel-Brief nennt den Schulträger:\n%s", blatt)
	}
}

func TestRechnungBuechereiSagtDassDerWegFehlt(t *testing.T) {
	positionen := []RechnungsPosition{
		{Titel: "Gregs Tagebuch", Barcode: "B-2", Ausleihdatum: time.Now(), Ersatzpreis: 995},
	}
	got, err := ErzeugeRechnung(Schueler{Vorname: "Bea", Nachname: "Buecherei"}, positionen, testSchule(), testZahlung())
	if err != nil {
		t.Fatalf("ErzeugeRechnung: %v", err)
	}
	blatt := blattText(t, got)

	// Der Zahlungsweg des Trägers ist offen; der Brief darf sich keinen ausdenken.
	if !strings.Contains(blatt, "nicht hinterlegt") {
		t.Errorf("der Brief sagt nicht, dass die Bankverbindung des Trägers fehlt:\n%s", blatt)
	}
	if strings.Contains(blatt, "HCC-Schulbereich") {
		t.Errorf("ein Buch der Schülerbücherei zeigt auf das Konto des LANDES — das ist die "+
			"falsche Kasse:\n%s", blatt)
	}
	if strings.Contains(blatt, barsatz) {
		t.Errorf("Barzahlung steht weiterhin auf dem Blatt:\n%s", blatt)
	}
}

// Eine Rechnung listet alle offenen Forderungen eines Schülers und kann sich ihre Positionen
// nicht aussuchen. Stehen beide Töpfe darauf, stehen beide Wege da, mit ihrer jeweiligen
// Teilsumme. Eine gemeinsame Summe auf ein Konto wäre Geld des
// Landes in der Kasse des Trägers.
func TestRechnungMitBeidenToepfenZeigtBeideWegeMitTeilsummen(t *testing.T) {
	positionen := []RechnungsPosition{
		{Titel: "Deutschbuch 7", Barcode: "B-1", Ausleihdatum: time.Now(), Ersatzpreis: 1990, Land: true},
		{Titel: "Gregs Tagebuch", Barcode: "B-2", Ausleihdatum: time.Now(), Ersatzpreis: 995},
	}
	got, err := ErzeugeRechnung(Schueler{Vorname: "Mia", Nachname: "Gemischt"}, positionen, testSchule(), testZahlung())
	if err != nil {
		t.Fatalf("ErzeugeRechnung: %v", err)
	}
	blatt := blattText(t, got)

	for _, muss := range []string{"HCC-Schulbereich", "nicht hinterlegt", "19,90 EUR", "9,95 EUR"} {
		if !strings.Contains(blatt, muss) {
			t.Errorf("auf dem gemischten Blatt fehlt %q:\n%s", muss, blatt)
		}
	}
}
