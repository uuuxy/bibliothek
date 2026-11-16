package pdf

import (
	"bytes"
	"strings"
	"testing"
	"time"
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

// Die Erzeuger mit maroto (Kontoauszug und Laufzettel, Rechnung, LMF-Plan) übergeben ihren Text
// dem cp1252-Übersetzer von gofpdf, den maroto selbst aufruft; ohne die Ersetzung käme
// „Şafak Wiśniewski" als „.afak Wi.niewski" auf das Blatt. Geprüft am fertigen PDF.
func TestMarotoErzeugerDruckenZeichenAusserhalbVonCp1252(t *testing.T) {
	jetzt := time.Date(2026, 11, 11, 9, 0, 0, 0, time.UTC)
	auszug, err := ErzeugeKontoauszuege([]KontoauszugEintrag{{
		Schueler: KontoauszugSchueler{Vorname: "Şafak", Nachname: "Wiśniewski", Klasse: "07G1"},
		Buecher:  []KontoauszugBuch{{Titel: "Łukasz und die Brücke", Barcode: "B-1", Ausleihdatum: jetzt, Rueckgabedatum: jetzt}},
	}}, testSchule(), true)
	if err != nil {
		t.Fatalf("Kontoauszug: %v", err)
	}
	rechnung, err := ErzeugeRechnung(Schueler{Vorname: "Şafak", Nachname: "Wiśniewski", Strasse: "Łąkowa", Hausnummer: "3",
		PLZ: "61381", Ort: "Friedrichsdorf"},
		[]RechnungsPosition{{Titel: "Łukasz und die Brücke", Barcode: "B-2", Ausleihdatum: jetzt, Ersatzpreis: 1995, Land: true}},
		testSchule(), testZahlung())
	if err != nil {
		t.Fatalf("Rechnung: %v", err)
	}
	plan, err := ErzeugeLMFPlan(lmfTestPlan(3, "Frau Yılmaz holt ab"), jetzt)
	if err != nil {
		t.Fatalf("LMF-Plan: %v", err)
	}

	for _, fall := range []struct {
		name     string
		roh      []byte
		erwartet []string
		verboten []string
	}{
		{"Kontoauszug", auszug, []string{"Safak Wisniewski", "Lukasz und die"}, []string{".afak", "Wi.niewski", ".ukasz"}},
		{"Rechnung", rechnung, []string{"Safak Wisniewski", "Lakowa 3", "Lukasz und die"}, []string{".afak", "Wi.niewski", ".ukasz", ".akowa"}},
		{"LMF-Plan", plan, []string{"Yilmaz"}, []string{"Y.lmaz"}},
	} {
		text := pdfText(t, fall.roh)
		for _, stueck := range fall.erwartet {
			if !strings.Contains(text, stueck) {
				t.Errorf("%s: %q steht nicht auf dem Blatt", fall.name, stueck)
			}
		}
		for _, stueck := range fall.verboten {
			if strings.Contains(text, stueck) {
				t.Errorf("%s: entstellter Text auf dem Blatt: %q", fall.name, stueck)
			}
		}
	}
}
