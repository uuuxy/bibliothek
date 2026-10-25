package pdf

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"bibliothek/internal/pdftest"
)

// Das Paket pdf erzeugt jedes Schriftstück, das die Schule aus der Hand gibt: Ersatzforderung,
// Mahnliste, Kontoauszug. Ein Panic oder ein leeres Dokument fiele sonst erst dem Sekretariat
// auf, im Zweifel gegenüber Eltern.
//
// Diese Tests prüfen nur, was ohne Parser für PDFs belastbar ist: dass ein gültiges Dokument
// entsteht, dass Sonderfälle es nicht umbringen und dass Beträge stimmen. Wie es aussieht,
// entscheidet das Auge, nicht ein Test.

// istPDF prüft die Signatur und den Abschluss-Marker. Ein abgeschnittenes Dokument
// beginnt zwar mit %PDF, hat aber kein %%EOF; so sähe ein Fehler bei der Ausgabe aus.
func istPDF(t *testing.T, b []byte, was string) {
	t.Helper()

	if len(b) == 0 {
		t.Fatalf("%s: leeres Dokument", was)
	}
	if !bytes.HasPrefix(b, []byte("%PDF-")) {
		t.Fatalf("%s: fehlende PDF-Signatur, Anfang: %q", was, string(b[:min(16, len(b))]))
	}
	if !bytes.Contains(b[max(0, len(b)-1024):], []byte("%%EOF")) {
		t.Fatalf("%s: kein %%%%EOF am Ende — Dokument abgeschnitten (%d Bytes)", was, len(b))
	}
}

// umlautText liefert einen Titel, der jeden byteweisen Schnitt auffliegen ließe.
const umlautText = "Die Bärenhöhle im Grünen Wäldchen — Fünf Freunde lösen jeden Fall über Ostern"

func testSchule() SchuleInfo {
	return SchuleInfo{
		Name:    "Philipp-Reis-Schule",
		Strasse: "Schulstraße 1",
		PLZ:     "61440",
		Ort:     "Oberursel",
	}
}

// testZahlung sind die Vorgaben des Musterschreibens, wie sie repository.BescheidAngabenAus
// einsetzt, wenn die Schule nichts eingetragen hat.
func testZahlung() Zahlungsangaben {
	return Zahlungsangaben{
		Zahlstelle:     "HCC-Schulbereich bei der Landesbank Hessen-Thüringen",
		Bankverbindung: "Konto-Nr. 1002401\nIBAN DE86500500000001002401",
	}
}

func TestErzeugeRechnung(t *testing.T) {
	schueler := Schueler{
		Vorname: "Änne", Nachname: "Müller-Lüdenscheidt",
		Strasse: "Übergasse", Hausnummer: "3a", PLZ: "61440", Ort: "Oberursel",
	}
	positionen := []RechnungsPosition{
		{Titel: umlautText, Barcode: "B-00001", Ausleihdatum: time.Now().AddDate(0, -3, 0), Ersatzpreis: 1299},
		{Titel: "Zweites Buch", Barcode: "B-00002", Ausleihdatum: time.Now().AddDate(0, -2, 0), Ersatzpreis: 750},
	}

	got, err := ErzeugeRechnung(schueler, positionen, testSchule(), testZahlung())
	if err != nil {
		t.Fatalf("ErzeugeRechnung: %v", err)
	}
	istPDF(t, got, "Rechnung")
}

// Drei Posten zu 0,10 € ergeben auf der Rechnung 0,30 €. Als float64 summiert wären es
// 0.30000000000000004 gewesen; in Cent ist die Summe genau, und das Blatt zeigt sie so.
func TestRechnungSummeRundetAufCent(t *testing.T) {
	positionen := make([]RechnungsPosition, 3)
	for i := range positionen {
		positionen[i] = RechnungsPosition{Titel: "Posten", Barcode: "B-1", Ausleihdatum: time.Now(), Ersatzpreis: 10}
	}

	got, err := ErzeugeRechnung(Schueler{Vorname: "A", Nachname: "B"}, positionen, testSchule(), testZahlung())
	if err != nil {
		t.Fatalf("ErzeugeRechnung: %v", err)
	}
	istPDF(t, got, "Rechnung mit Rundung")
	if blatt := strings.Join(pdftest.Texte(t, got), " "); !strings.Contains(blatt, "0,30 EUR") {
		t.Errorf("die Summe der Rechnung steht nicht als 0,30 EUR auf dem Blatt: %q", blatt)
	}
}

func TestErzeugeRechnung_OhnePosten(t *testing.T) {
	// Eine Forderung ohne Posten ist fachlich unsinnig, darf den Server aber nicht umbringen;
	// eine Tür darf sich auf einen Fehler statt eines Panics verlassen.
	got, err := ErzeugeRechnung(Schueler{Vorname: "Leer", Nachname: "Fall"}, nil, testSchule(), testZahlung())
	if err != nil {
		t.Fatalf("ErzeugeRechnung ohne Posten: %v", err)
	}
	istPDF(t, got, "Rechnung ohne Posten")
}

func TestErzeugeKontoauszug(t *testing.T) {
	schueler := KontoauszugSchueler{Vorname: "Änne", Nachname: "Groß", Klasse: "8a"}
	buecher := []KontoauszugBuch{
		{Titel: umlautText, Barcode: "B-00020", Ausleihdatum: time.Now().AddDate(0, -1, 0)},
		{Titel: "Zurückgegeben", Barcode: "B-00021",
			Ausleihdatum: time.Now().AddDate(0, -2, 0), Rueckgabedatum: time.Now().AddDate(0, -1, 0)},
	}

	got, err := ErzeugeKontoauszug(schueler, buecher)
	if err != nil {
		t.Fatalf("ErzeugeKontoauszug: %v", err)
	}
	istPDF(t, got, "Kontoauszug")
}

// TestErzeugeKontoauszuege_MitUnterschrift deckt den Laufzettel der Abgänger ab: derselbe Auszug,
// aber mit Freigabezeile. Beide Varianten müssen ein Dokument liefern.
func TestErzeugeKontoauszuege_MitUnterschrift(t *testing.T) {
	eintraege := []KontoauszugEintrag{
		{Schueler: KontoauszugSchueler{Vorname: "Ali", Nachname: "Öztürk", Klasse: "10a"}},
		{Schueler: KontoauszugSchueler{Vorname: "Bea", Nachname: "Schäfer", Klasse: "10b"},
			Buecher: []KontoauszugBuch{{Titel: "Rest", Barcode: "B-9", Ausleihdatum: time.Now()}}},
	}

	for _, mitUnterschrift := range []bool{false, true} {
		got, err := ErzeugeKontoauszuege(eintraege, mitUnterschrift)
		if err != nil {
			t.Fatalf("ErzeugeKontoauszuege(mitUnterschrift=%v): %v", mitUnterschrift, err)
		}
		istPDF(t, got, "Kontoauszug-Stapel")
	}
}

// TestUeberlangeEingabenBringenKeinDokumentUm: Titel kommen aus Fremdimporten (Littera,
// DNB) und sind nicht längenbegrenzt. Ein 5000 Zeichen langer Titel darf das Dokument
// höchstens hässlich machen, nicht verhindern.
func TestUeberlangeEingabenBringenKeinDokumentUm(t *testing.T) {
	lang := strings.Repeat("Ü", 5000)

	got, err := ErzeugeRechnung(
		Schueler{Vorname: lang, Nachname: lang, Strasse: lang, Ort: lang},
		[]RechnungsPosition{{Titel: lang, Barcode: "B-1", Ausleihdatum: time.Now(), Ersatzpreis: 100}},
		SchuleInfo{Name: lang, Strasse: lang, Ort: lang},
		testZahlung(),
	)
	if err != nil {
		t.Fatalf("überlange Eingaben: %v", err)
	}
	istPDF(t, got, "Rechnung mit überlangen Feldern")
}

func TestAbsenderzeile(t *testing.T) {
	faelle := []struct {
		name     string
		schule   SchuleInfo
		erwartet string
	}{
		{"vollständig", testSchule(), "Philipp-Reis-Schule · Schulstraße 1 · 61440 Oberursel"},
		{"ohne Namen", SchuleInfo{}, "Schulbibliothek"},
		{"ohne Anschrift", SchuleInfo{Name: "Nur Name"}, "Nur Name"},
		{"ohne Ort", SchuleInfo{Name: "N", Strasse: "S"}, "N"},
	}
	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			if got := f.schule.Absenderzeile(); got != f.erwartet {
				t.Errorf("Absenderzeile() = %q, erwartet %q", got, f.erwartet)
			}
		})
	}
}

func TestOrtDatum(t *testing.T) {
	if got := testSchule().ortDatum("01.02.2026"); got != "Oberursel, den 01.02.2026" {
		t.Errorf("ortDatum() = %q", got)
	}
	if got := (SchuleInfo{Name: "X"}).ortDatum("01.02.2026"); got != "01.02.2026" {
		t.Errorf("ortDatum() ohne Ort = %q, erwartet nur das Datum", got)
	}
}
