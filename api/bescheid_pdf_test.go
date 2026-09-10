package api

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"bibliothek/internal/pdftest"
	"bibliothek/pdf"
)

// Der Bescheid wird am ENTPACKTEN Inhaltsstrom geprüft, nicht an den Eingabedaten.
//
// Das Blatt ist ein Bescheid: Fehlt ein Satz, ist es ein anderer Brief. Am Bildschirm
// sieht man das nicht — das PDF geht direkt in den Drucker und dann in ein Kuvert.
// Dieselbe Technik wie beim Lernmittel-Etikett und beim Bestellanschreiben.

func testBescheid() BescheidBrief {
	return BescheidBrief{
		Schule: pdf.SchuleInfo{
			Name: "Philipp-Reis-Schule", Strasse: "Färberstraße 4", PLZ: "61381", Ort: "Friedrichsdorf",
		},
		Empfaenger: BescheidEmpfaenger{
			Anrede:  "Sehr geehrte Erziehungsberechtigte,",
			AnZeile: bescheidAnAnErzieher,
			Name:    "Ayşe Demir",
			Strasse: "Musterweg 12",
			PLZ:     "61381",
			Ort:     "Friedrichsdorf",
		},
		Geschaeftszeichen: "LMF-2026-014",
		Bearbeiter:        "Frau Naacher",
		Durchwahl:         "06172 5905-23",
		BriefDatum:        time.Date(2026, 9, 10, 0, 0, 0, 0, time.Local),
		FristBis:          time.Date(2026, 10, 8, 0, 0, 0, 0, time.Local),
		NichtZurueckgegeben: []BescheidPosition{
			{SchuelerName: "Ayşe Demir", Titel: "Mathematik 7", ISBN: "978-3-12-733071-6", Betrag: 24.90},
			{SchuelerName: "Ayşe Demir", Titel: "Deutschbuch 8 — Sprach- und Lesebuch", ISBN: "978-3-06-062477-3", Betrag: 18.50},
		},
		Beschaedigt: []BescheidPosition{
			{SchuelerName: "Ayşe Demir", Titel: "Englisch Green Line 7", ISBN: "978-3-12-834271-0", Betrag: 12.00},
		},
		Gesamtbetrag:   55.40,
		Zahlstelle:     "HCC-Schulbereich bei der Landesbank Hessen-Thüringen",
		Bankverbindung: "Konto-Nr. 1002401\nBankleitzahl 500 500 00\nIBAN DE86500500000001002401\nBIC HELADEFFXXX",
		Referenznummer: "5830 2026 1234 0001",
		Aufsicht:       "Staatlichen Schulamt für den Hochtaunuskreis und den Wetteraukreis, Konrad-Adenauer-Allee 1-11, 61118 Bad Vilbel",
		Schulleitung:   "Dr. M. Beispiel, Schulleiterin",
	}
}

// Der Wortlaut der Vorlage, Satz für Satz. Geprüft werden ANKER — Wortgruppen, die im
// Strom am Stück stehen; ganze Absätze bricht gofpdf um.
func TestBescheidTraegtDenWortlautDerVorlage(t *testing.T) {
	roh, err := GenerateBescheidPDF(testBescheid())
	if err != nil {
		t.Fatalf("Bescheid erzeugen: %v", err)
	}
	text := pdfText(t, roh)

	anker := []string{
		// Betreff
		"Nutzungsverh", "Schadenersatzforderung",
		// Die beiden einleitenden Absätze
		"Lernmittelfreiheit", "Eigentum des Landes",
		"pfleglich zu behandeln",
		"Bei Verlust oder Besch",
		// Beide Fallgruppen samt Einleitung
		"Sie haben/Ihr Kind hat die Lernmittel",
		"nicht ordnungsgem", "so stark besch",
		"Folgende Lernmittel sind betroffen",
		// Die Bitte-Sätze mit dem Fristdatum
		"Ich bitte um R", "Wiederbeschaffungspreises", "08.10.2026",
		// Zahlung
		"Gesamtbetrag", "55,40", "HCC-Schulbereich",
		"IBAN DE86500500000001002401", "BIC HELADEFFXXX",
		"Referenz-Nr", "5830 2026 1234 0001",
		// Androhung und Schluss
		"Verwaltungsvollstreckungsverfahrens",
		"Mit freundlichen Gr", "Schulleiterin",
		"Rechtsbehelfsbelehrung", "innerhalb eines Monats", "Widerspruch",
		// Tabellenkopf und Inhalt
		"Titel", "ISBN", "Preis", "Mathematik 7", "978-3-12-733071-6", "24,90",
		// Kopf und Anschrift
		"Philipp-Reis-Schule", "Musterweg 12", "61381 Friedrichsdorf",
		"An die Erziehungsberechtigten",
		// Infoblock
		"Gesch", "Bearbeiter", "Durchwahl", "Datum", "10.09.2026",
	}
	for _, a := range anker {
		if !strings.Contains(text, a) {
			t.Errorf("%q steht nicht im Bescheid", a)
		}
	}

	// Die Ausfüllhilfe der Vorlage gehört NICHT auf den fertigen Brief.
	for _, unerwuenscht := range []string{"4stellig", "E N T W U R F", "Summe Tabelle"} {
		if strings.Contains(text, unerwuenscht) {
			t.Errorf("%q ist ein Feld der Vorlage und darf im Brief nicht stehen", unerwuenscht)
		}
	}
}

// Eine Fallgruppe ohne Positionen erscheint gar nicht: Ein Kästchen mit leerer Tabelle
// wäre ein unausgefülltes Formular, kein Bescheid.
func TestBescheidZeigtNurBelegteFallgruppen(t *testing.T) {
	b := testBescheid()
	b.Beschaedigt = nil
	text := pdfText(t, mussErzeugen(t, b))
	if !strings.Contains(text, "nicht ordnungsgem") {
		t.Error("die belegte Gruppe fehlt")
	}
	if strings.Contains(text, "so stark besch") {
		t.Error("die leere Gruppe „beschädigt\" steht trotzdem im Brief")
	}

	b = testBescheid()
	b.NichtZurueckgegeben = nil
	text = pdfText(t, mussErzeugen(t, b))
	if strings.Contains(text, "nicht ordnungsgem") {
		t.Error("die leere Gruppe „nicht zurückgegeben\" steht trotzdem im Brief")
	}
	if !strings.Contains(text, "so stark besch") {
		t.Error("die belegte Gruppe fehlt")
	}
}

// Fehlt die Anschrift, sagt der Brief es an der Stelle, an der sie stehen müsste —
// dieselbe Regel wie beim Eltern-Mahnbrief. Ein Bescheid ohne Empfänger darf nicht
// aussehen wie ein vollständiger.
func TestBescheidOhneAnschriftSagtEs(t *testing.T) {
	b := testBescheid()
	b.Empfaenger.Strasse = ""
	b.Empfaenger.Ort = ""
	text := pdfText(t, mussErzeugen(t, b))
	if !strings.Contains(text, "keine Adresse hinterlegt") {
		t.Error("der Brief verschweigt die fehlende Anschrift")
	}
}

// Leere freiwillige Angaben lassen ihre Zeile weg, statt ein leeres Etikett zu drucken.
func TestBescheidLaesstLeereInfozeilenWeg(t *testing.T) {
	b := testBescheid()
	b.Geschaeftszeichen = ""
	b.Bearbeiter = ""
	b.Durchwahl = ""
	text := pdfText(t, mussErzeugen(t, b))
	for _, weg := range []string{"Bearbeiter", "Durchwahl"} {
		if strings.Contains(text, weg) {
			t.Errorf("%q steht im Brief, obwohl die Angabe leer ist", weg)
		}
	}
	// Das Datum steht immer.
	if !strings.Contains(text, "Datum") {
		t.Error("das Briefdatum fehlt")
	}
}

func mussErzeugen(t *testing.T, b BescheidBrief) []byte {
	t.Helper()
	roh, err := GenerateBescheidPDF(b)
	if err != nil {
		t.Fatalf("Bescheid erzeugen: %v", err)
	}
	return roh
}

// Zum Ansehen mit eigenen Augen: BESCHEID_PDF_AUSGABE=/tmp/bescheid.pdf go test -run Muster
func TestBescheidMusterSchreiben(t *testing.T) {
	ziel := os.Getenv("BESCHEID_PDF_AUSGABE")
	if ziel == "" {
		t.Skip("BESCHEID_PDF_AUSGABE nicht gesetzt")
	}
	if err := os.WriteFile(ziel, mussErzeugen(t, testBescheid()), 0o600); err != nil {
		t.Fatalf("schreiben: %v", err)
	}
	t.Logf("Muster geschrieben: %s", ziel)
}

// Kein Seitenumbruch mitten in der Tabelle — geprüft SEITE FÜR SEITE.
//
// Der Fund, der dieses Gate erzwungen hat (10.09.2026): Der erste Entwurf riss seine
// Tabellenkopfzeile entzwei. „Name des/der" stand unten auf Seite 1,
// „Schülers/Schülerin" oben auf Seite 2, dahinter die Spaltenüberschriften am Fuß der
// Folgeseite, dazwischen eine halbleere Seite. Im GESAMTTEXT war alles vorhanden — das
// alte Gate blieb grün, und nur das Ansehen des Papiers zeigte den Schaden. Seitdem
// liest pdftest.TexteJeSeite die Seiten getrennt.
func TestBescheidZerreisstDieTabelleNicht(t *testing.T) {
	b := testBescheid()
	// So viele Positionen, dass die Tabelle sicher über den Seitenrand hinausgeht.
	b.NichtZurueckgegeben = nil
	for i := 0; i < 25; i++ {
		b.NichtZurueckgegeben = append(b.NichtZurueckgegeben, BescheidPosition{
			SchuelerName: "Ayşe Demir",
			Titel:        fmt.Sprintf("Lernmittel Nummer %d", i+1),
			ISBN:         fmt.Sprintf("978-3-12-7330%02d-6", i),
			Betrag:       9.90,
		})
	}
	b.Beschaedigt = nil

	seiten := pdftest.TexteJeSeite(t, mussErzeugen(t, b))
	if len(seiten) < 2 {
		t.Fatalf("nur %d Seite(n) — der Fall „Tabelle über den Seitenrand\" wird nicht geprüft", len(seiten))
	}

	for nr, texte := range seiten {
		seite := strings.Join(texte, "\n")
		kopfOben := strings.Contains(seite, "Name des/der")
		kopfUnten := strings.Contains(seite, "lers/Sch")
		// 1. Die zweizeilige Kopfzeile gehört zusammen.
		if kopfOben != kopfUnten {
			t.Errorf("Seite %d: die Tabellenkopfzeile ist zerrissen (obere Zeile %v, untere %v)",
				nr+1, kopfOben, kopfUnten)
		}
		// 2. Datenzeilen ohne Kopfzeile sind Zahlen ohne Überschrift.
		if strings.Contains(seite, "978-3-12-7330") && !kopfOben {
			t.Errorf("Seite %d trägt Tabellenzeilen, aber keine Kopfzeile", nr+1)
		}
		// 3. Keine Seite ohne Inhalt außer der Seitenzahl.
		if len(texte) <= 1 {
			t.Errorf("Seite %d ist leer (nur %d Textstück(e)) — der Umbruch hat eine Lücke gerissen", nr+1, len(texte))
		}
	}
}

// Namen mit Zeichen außerhalb von cp1252 kommen richtig aufs Papier.
//
// Der Fund (10.09.2026, am fertigen PDF gesehen): „Ayşe" wurde zu „Ay.e". Dieselbe Falle
// wie beim Schüler-Etikett — der Zeichensatz des PDFs kennt das ş nicht, und ein Bescheid
// mit entstelltem Namen ist ein fehlerhafter Bescheid.
func TestBescheidErsetztZeichenAusserhalbDesZeichensatzes(t *testing.T) {
	b := testBescheid()
	b.Empfaenger.Name = "Ayşe Öztürk-Ćurić"
	b.NichtZurueckgegeben = []BescheidPosition{
		{SchuelerName: "Ayşe Öztürk-Ćurić", Titel: "Mathematik 7", ISBN: "978-3-12-733071-6", Betrag: 24.90},
	}
	b.Beschaedigt = nil

	text := pdfText(t, mussErzeugen(t, b))
	if strings.Contains(text, "Ay.e") || strings.Contains(text, ".uri") {
		t.Errorf("Zeichen wurden zu Punkten verstümmelt statt ersetzt:\n%s", text[:min(600, len(text))])
	}
	// Ersetzt, nicht verschluckt: „Ayse" und „Curic" müssen lesbar sein. Ö und ü kennt
	// cp1252 und bleiben, wie sie sind.
	for _, erwartet := range []string{"Ayse", "Curic"} {
		if !strings.Contains(text, erwartet) {
			t.Errorf("%q steht nicht im Brief — die Ersetzung greift nicht", erwartet)
		}
	}
}
