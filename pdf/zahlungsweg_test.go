package pdf

import (
	"strings"
	"testing"
	"time"

	"bibliothek/internal/pdftest"
)

// Gemessen wird am FERTIGEN PDF, nicht am Struct: Beide Etiketten-Bugs von 2026 waren an
// den Zwischenschichten grün — das Feld stand im Struct, gezeichnet wurde es nicht. Hier
// geht es um Geld, das an die richtige Kasse muss.
//
// Der Satz, der weg musste: „bar in der Bibliothek". Die Arbeitshilfe zum Erlass vom
// 17.12.2014 sagt für die Lernmittel des Landes, es dürfe „keine Bargeldannahme vorgesehen
// werden" — und beide Altbriefe sahen genau das vor.

const barsatz = "bar in der Bibliothek"

func blattText(t *testing.T, roh []byte) string {
	t.Helper()
	return strings.Join(pdftest.Texte(t, roh), " ")
}

func TestRechnungLernmittelNenntDasKontoDesLandes(t *testing.T) {
	items := []RechnungItem{
		{Titel: "Deutschbuch 7", Barcode: "B-1", Ausleihdatum: time.Now(), Ersatzpreis: 19.90, IstLernmittel: true},
	}
	got, err := GenerateRechnung(Schueler{Vorname: "Lena", Nachname: "Land"}, items, testSchule(), testZahlung())
	if err != nil {
		t.Fatalf("GenerateRechnung: %v", err)
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
	items := []RechnungItem{
		{Titel: "Gregs Tagebuch", Barcode: "B-2", Ausleihdatum: time.Now(), Ersatzpreis: 9.95},
	}
	got, err := GenerateRechnung(Schueler{Vorname: "Bea", Nachname: "Buecherei"}, items, testSchule(), testZahlung())
	if err != nil {
		t.Fatalf("GenerateRechnung: %v", err)
	}
	blatt := blattText(t, got)

	// Der Zahlungsweg des Trägers ist offen (E5) — der Brief darf sich keinen ausdenken.
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

// Eine Rechnung listet ALLE offenen Forderungen eines Schülers und kann sich ihre
// Positionen nicht aussuchen. Stehen beide Töpfe darauf, müssen beide Wege dastehen —
// mit ihrer jeweiligen Teilsumme. Eine gemeinsame Summe auf ein Konto wäre Geld des
// Landes in der Kasse des Trägers.
func TestRechnungMitBeidenToepfenZeigtBeideWegeMitTeilsummen(t *testing.T) {
	items := []RechnungItem{
		{Titel: "Deutschbuch 7", Barcode: "B-1", Ausleihdatum: time.Now(), Ersatzpreis: 19.90, IstLernmittel: true},
		{Titel: "Gregs Tagebuch", Barcode: "B-2", Ausleihdatum: time.Now(), Ersatzpreis: 9.95},
	}
	got, err := GenerateRechnung(Schueler{Vorname: "Mia", Nachname: "Gemischt"}, items, testSchule(), testZahlung())
	if err != nil {
		t.Fatalf("GenerateRechnung: %v", err)
	}
	blatt := blattText(t, got)

	for _, muss := range []string{"HCC-Schulbereich", "nicht hinterlegt", "19.90 EUR", "9.95 EUR"} {
		if !strings.Contains(blatt, muss) {
			t.Errorf("auf dem gemischten Blatt fehlt %q:\n%s", muss, blatt)
		}
	}
}

func TestElternbriefFolgtDemselbenZahlungsweg(t *testing.T) {
	basis := SchadensfallInfo{
		Beschreibung: "Einband gerissen", Betrag: 24.90, ErstelltAm: time.Now(),
		SchuelerVorname: "Jörg", SchuelerNachname: "Weiß", SchuelerKlasse: "7a",
		BuchTitel: "Mathe 8", ExemplarBarcode: "B-3",
	}

	lernmittel := basis
	lernmittel.IstLernmittel = true
	got, err := GenerateSchadensfallPDF(lernmittel, testSchule(), testZahlung())
	if err != nil {
		t.Fatalf("Elternbrief (Lernmittel): %v", err)
	}
	blatt := blattText(t, got)
	if !strings.Contains(blatt, "HCC-Schulbereich") {
		t.Errorf("der Elternbrief für ein Lernmittel nennt das Konto des Landes nicht:\n%s", blatt)
	}
	if strings.Contains(blatt, barsatz) {
		t.Errorf("der Elternbrief verlangt weiterhin Barzahlung:\n%s", blatt)
	}

	got, err = GenerateSchadensfallPDF(basis, testSchule(), testZahlung())
	if err != nil {
		t.Fatalf("Elternbrief (Bücherei): %v", err)
	}
	blatt = blattText(t, got)
	if !strings.Contains(blatt, "nicht hinterlegt") {
		t.Errorf("der Elternbrief für ein Büchereibuch sagt den fehlenden Weg nicht:\n%s", blatt)
	}
	if strings.Contains(blatt, "HCC-Schulbereich") {
		t.Errorf("Büchereibuch zeigt auf das Konto des Landes:\n%s", blatt)
	}
}
