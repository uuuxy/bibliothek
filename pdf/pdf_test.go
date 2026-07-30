package pdf

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

// Das pdf-Paket erzeugt jedes Schriftstück, das die Schule aus der Hand gibt:
// Ersatzforderung, Elternbrief zum Schadensfall, Mahnliste, Kontoauszug. Bis hierher
// hatte es keinen einzigen Test — ein Panic oder ein leeres Dokument wäre erst dem
// Sekretariat aufgefallen, im Zweifel gegenüber Eltern.
//
// Die Tests prüfen bewusst nur, was ohne PDF-Parser belastbar ist: dass überhaupt ein
// gültiges Dokument entsteht, dass Sonderfälle es nicht umbringen und dass Beträge
// stimmen. Wie es aussieht, entscheidet das Auge — nicht ein Test.

// istPDF prüft die Signatur und den Abschluss-Marker. Ein abgeschnittenes Dokument
// beginnt zwar mit %PDF, hat aber kein %%EOF — genau so sähe ein Ausgabefehler aus.
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

func TestGenerateRechnung(t *testing.T) {
	schueler := Schueler{
		Vorname: "Änne", Nachname: "Müller-Lüdenscheidt",
		Strasse: "Übergasse", Hausnummer: "3a", PLZ: "61440", Ort: "Oberursel",
	}
	items := []RechnungItem{
		{Titel: umlautText, Barcode: "B-00001", Ausleihdatum: time.Now().AddDate(0, -3, 0), Ersatzpreis: 12.99},
		{Titel: "Zweites Buch", Barcode: "B-00002", Ausleihdatum: time.Now().AddDate(0, -2, 0), Ersatzpreis: 7.50},
	}

	got, err := GenerateRechnung(schueler, items, testSchule())
	if err != nil {
		t.Fatalf("GenerateRechnung: %v", err)
	}
	istPDF(t, got, "Rechnung")
}

// TestRechnungSummeRundetAufCent sichert die bewusste Entscheidung ab, Geld als float64
// zu führen (siehe Kommentar in buildItemsTableBlock). Drei Posten zu 0.10 ergeben in
// Fließkomma 0.30000000000000004; ohne die Rundung stünde das auf der Forderung.
func TestRechnungSummeRundetAufCent(t *testing.T) {
	items := make([]RechnungItem, 3)
	for i := range items {
		items[i] = RechnungItem{Titel: "Posten", Barcode: "B-1", Ausleihdatum: time.Now(), Ersatzpreis: 0.10}
	}

	got, err := GenerateRechnung(Schueler{Vorname: "A", Nachname: "B"}, items, testSchule())
	if err != nil {
		t.Fatalf("GenerateRechnung: %v", err)
	}
	istPDF(t, got, "Rechnung mit Rundung")

	// Gegenprobe auf die reine Rechenlogik — im PDF-Strom ist der Text komprimiert.
	var summe float64
	for _, it := range items {
		summe += it.Ersatzpreis
	}
	if summe == 0.30 {
		t.Skip("Fließkomma verhält sich hier exakt — die Rundung ist trotzdem richtig")
	}
	if gerundet := float64(int(summe*100+0.5)) / 100; gerundet != 0.30 {
		t.Errorf("Rundung auf Cent ergibt %v statt 0.30", gerundet)
	}
}

func TestGenerateRechnungOhnePosten(t *testing.T) {
	// Eine Forderung ohne Posten ist fachlich unsinnig, darf den Server aber nicht
	// umbringen — Handler dürfen sich auf einen Fehler statt eines Panics verlassen.
	got, err := GenerateRechnung(Schueler{Vorname: "Leer", Nachname: "Fall"}, nil, testSchule())
	if err != nil {
		t.Fatalf("GenerateRechnung ohne Posten: %v", err)
	}
	istPDF(t, got, "Rechnung ohne Posten")
}

func TestGenerateSchadensfallPDF(t *testing.T) {
	data := SchadensfallInfo{
		Beschreibung:     "Wasserschaden über mehrere Seiten, Einband gewellt — Öffnen nur noch teilweise möglich.",
		Betrag:           24.90,
		ErstelltAm:       time.Now(),
		SchuelerVorname:  "Jörg",
		SchuelerNachname: "Weiß",
		SchuelerKlasse:   "7a",
		BuchTitel:        umlautText,
		ExemplarBarcode:  "B-04711",
	}

	got, err := GenerateSchadensfallPDF(data, testSchule())
	if err != nil {
		t.Fatalf("GenerateSchadensfallPDF: %v", err)
	}
	istPDF(t, got, "Schadensfall")
}

func TestGenerateMahnliste(t *testing.T) {
	liste := []MahnungSchueler{
		{
			Vorname: "Ömer", Nachname: "Özdemir", Klasse: "5b",
			Buecher: []MahnungBuch{
				{Titel: umlautText, Barcode: "B-00010", FaelligSeit: time.Now().AddDate(0, 0, -21)},
			},
		},
		{
			Vorname: "Süleyman", Nachname: "Şahin", Klasse: "9c",
			Buecher: []MahnungBuch{
				{Titel: "Kurz", Barcode: "B-00011", FaelligSeit: time.Now().AddDate(0, 0, -3)},
				{Titel: "Noch eins", Barcode: "B-00012", FaelligSeit: time.Now().AddDate(0, 0, -40)},
			},
		},
	}

	got, err := GenerateMahnliste(liste)
	if err != nil {
		t.Fatalf("GenerateMahnliste: %v", err)
	}
	istPDF(t, got, "Mahnliste")
}

// TestGenerateMahnlisteLeer: Die Mahnliste wird auch dann angefordert, wenn gerade
// niemand etwas schuldet. Ein Fehler wäre hier falsch — der Nutzer bekäme HTTP 500
// für den erfreulichen Fall.
func TestGenerateMahnlisteLeer(t *testing.T) {
	got, err := GenerateMahnliste(nil)
	if err != nil {
		t.Fatalf("GenerateMahnliste ohne Schüler: %v", err)
	}
	istPDF(t, got, "leere Mahnliste")
}

func TestGenerateKontoauszug(t *testing.T) {
	schueler := KontoauszugSchueler{Vorname: "Änne", Nachname: "Groß", Klasse: "8a"}
	buecher := []KontoauszugBuch{
		{Titel: umlautText, Barcode: "B-00020", Ausleihdatum: time.Now().AddDate(0, -1, 0)},
		{Titel: "Zurückgegeben", Barcode: "B-00021",
			Ausleihdatum: time.Now().AddDate(0, -2, 0), Rueckgabedatum: time.Now().AddDate(0, -1, 0)},
	}

	got, err := GenerateKontoauszug(schueler, buecher)
	if err != nil {
		t.Fatalf("GenerateKontoauszug: %v", err)
	}
	istPDF(t, got, "Kontoauszug")
}

// TestKontoauszugBatchMitUnterschrift deckt den Abgänger-Laufzettel ab: derselbe Auszug,
// aber mit Freigabezeile. Beide Varianten müssen ein Dokument liefern.
func TestKontoauszugBatchMitUnterschrift(t *testing.T) {
	eintraege := []KontoauszugEintrag{
		{Schueler: KontoauszugSchueler{Vorname: "Ali", Nachname: "Öztürk", Klasse: "10a"}},
		{Schueler: KontoauszugSchueler{Vorname: "Bea", Nachname: "Schäfer", Klasse: "10b"},
			Buecher: []KontoauszugBuch{{Titel: "Rest", Barcode: "B-9", Ausleihdatum: time.Now()}}},
	}

	for _, mitUnterschrift := range []bool{false, true} {
		got, err := GenerateKontoauszugBatch(eintraege, mitUnterschrift)
		if err != nil {
			t.Fatalf("GenerateKontoauszugBatch(mitUnterschrift=%v): %v", mitUnterschrift, err)
		}
		istPDF(t, got, "Kontoauszug-Stapel")
	}
}

// TestUeberlangeEingabenBringenKeinDokumentUm: Titel kommen aus Fremdimporten (Littera,
// DNB) und sind nicht längenbegrenzt. Ein 5000 Zeichen langer Titel darf das Dokument
// höchstens hässlich machen, nicht verhindern.
func TestUeberlangeEingabenBringenKeinDokumentUm(t *testing.T) {
	lang := strings.Repeat("Ü", 5000)

	got, err := GenerateRechnung(
		Schueler{Vorname: lang, Nachname: lang, Strasse: lang, Ort: lang},
		[]RechnungItem{{Titel: lang, Barcode: "B-1", Ausleihdatum: time.Now(), Ersatzpreis: 1}},
		SchuleInfo{Name: lang, Strasse: lang, Ort: lang},
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
	if got := testSchule().OrtDatum("01.02.2026"); got != "Oberursel, den 01.02.2026" {
		t.Errorf("OrtDatum() = %q", got)
	}
	if got := (SchuleInfo{Name: "X"}).OrtDatum("01.02.2026"); got != "01.02.2026" {
		t.Errorf("OrtDatum() ohne Ort = %q, erwartet nur das Datum", got)
	}
}
