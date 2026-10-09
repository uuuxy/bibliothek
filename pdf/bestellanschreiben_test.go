package pdf

// Das Anschreiben an den Lieferanten, geprüft am gedruckten Blatt. Der Erzeuger kennt weder
// Töpfe noch die Anlagen der Mail: Er druckt Betreff und Vermerk seiner Eingabe und den Satz
// zum Weg der Etiketten. Dass die Tür sie richtig füllt, prüft api/bestellanschreiben_test.go.

import (
	"strings"
	"testing"

	"bibliothek/internal/pdftest"
	"bibliothek/pkg/schulzeit"
)

// Das Anschreiben versprach dem Lieferanten früher immer einen beigefügten Bogen mit
// Aufklebern, auch wenn der Mail keiner beilag. Eine solche Anweisung kann er nur übergehen
// oder nachfragen; beides kostet die Lieferung Zeit.
func TestBestellanschreibenNenntBarcodebogenNurWennErBeiliegt(t *testing.T) {
	const vermerk = "Vermerk-Probe."
	mit := bestellanschreibenText(BogenLiegtBei, vermerk)
	if !strings.Contains(mit, barcodebogenSatz) {
		t.Error("Mit Bogen: Der Hinweis auf die Aufkleber fehlt im Anschreiben")
	}

	ohne := bestellanschreibenText(OhneEtiketten, vermerk)
	if strings.Contains(ohne, barcodebogenSatz) {
		t.Error("Ohne Bogen: Das Anschreiben verweist auf eine Anlage, die nicht existiert")
	}

	// Liegen die Etiketten hinter dem Link der Mail, nennt der Brief den Link. Bliebe der Satz
	// vom beigefügten Bogen stehen, suchte der Händler eine Anlage, die es nicht gibt, und der
	// Link, der die Bestätigung trägt, bliebe ungeklickt.
	ueberLink := bestellanschreibenText(BogenHinterLink, vermerk)
	if strings.Contains(ueberLink, barcodebogenSatz) {
		t.Error("Bogen hinter dem Link: Das Anschreiben verweist trotzdem auf eine beigefügte Anlage")
	}
	if !strings.Contains(ueberLink, "Link in dieser E-Mail") {
		t.Error("Bogen hinter dem Link: Das Anschreiben sagt nicht, wo der Händler die Etiketten bekommt")
	}

	// Der Rest des Briefs ist in allen Fällen gleich: Es fällt genau ein Satz weg.
	for name, text := range map[string]string{"mit Bogen": mit, "ohne Bogen": ohne, "hinter dem Link": ueberLink} {
		for _, satz := range []string{"Sehr geehrte Damen und Herren", vermerk, "Die Rechnung senden Sie bitte", "Bestellte Titel:"} {
			if !strings.Contains(text, satz) {
				t.Errorf("%s: %q fehlt im Anschreiben", name, satz)
			}
		}
	}
}

// Was der Brief setzt, in der Folge von oben nach unten: Kopf der Schule, Ort und Tag, Betreff,
// Text mit Vermerk und dem Satz zu den Etiketten, die Tabelle mit ihren Spalten, der Gruß.
func TestBestellanschreiben_SetztKopfBetreffTextUndTabelleInIhrerFolge(t *testing.T) {
	roh, err := GenerateBestellanschreibenPDF(Bestellanschreiben{
		Betreff: "Betreff-Probe", Vermerk: "Vermerk-Probe.", Etiketten: OhneEtiketten,
		Positionen: []BestellPosition{
			{Titel: "Mathematik 7", Autor: "Lambacher", ISBN: "978-3-12-733771-9", Menge: 30},
			{Titel: "Atlas", Autor: "Diercke", ISBN: "978-3-14-100800-5", Menge: 2},
		},
	}, SchuleInfo{Name: "Testschule", Strasse: "Schulweg 1", PLZ: "61381", Ort: "Friedrichsdorf"})
	if err != nil {
		t.Fatalf("Anschreiben drucken: %v", err)
	}
	pdftest.IstPDF(t, roh, "Bestellanschreiben")

	gesetzt := strings.Join(pdftest.TexteInReihenfolge(t, roh), " | ")
	folge := strings.Join([]string{
		"Testschule", "Testschule · Schulweg 1 · 61381 Friedrichsdorf",
		"Friedrichsdorf, den " + schulzeit.Jetzt().Format(dateFormatDE),
		"An den Buchlieferanten",
		"Betreff-Probe",
		"Sehr geehrte Damen und Herren,",
		"hiermit bestellen wir die nachfolgend aufgeführten Buchtitel zur Lieferung.",
		"Vermerk-Probe.",
		// Der Satz ist länger als die Zeile und bricht vor seinem letzten Wort um.
		"Die Rechnung senden Sie bitte an die oben angegebene Anschrift und führen Sie darauf denselben",
		"Vermerk.",
		"Bestellte Titel:",
		"Buchtitel", "Autor", "ISBN", "Menge",
		"Mathematik 7", "Lambacher", "978-3-12-733771-9", "30",
		"Atlas", "Diercke", "978-3-14-100800-5", "2",
		"Mit freundlichen Grüßen,", "Das Bibliotheksteam",
	}, " | ")
	if gesetzt != folge {
		t.Errorf("gesetzt:\n%s\nerwartet:\n%s", gesetzt, folge)
	}
}

// Jeder Weg der Etiketten steht mit seinem Satz auf dem Blatt, der Weg ohne Etiketten mit keinem.
func TestBestellanschreiben_DerSatzZuDenEtikettenStehtAufDemBlatt(t *testing.T) {
	for _, f := range []struct {
		weg      EtikettenWeg
		muss     string
		darfNich []string
	}{
		{OhneEtiketten, "", []string{"Aufklebern"}},
		{BogenLiegtBei, "aus dem beigefügten Bogen", []string{"Link in dieser E-Mail"}},
		{BogenHinterLink, "Link in dieser E-Mail", []string{"aus dem beigefügten Bogen"}},
	} {
		roh, err := GenerateBestellanschreibenPDF(Bestellanschreiben{Betreff: "B", Vermerk: "V.", Etiketten: f.weg}, SchuleInfo{})
		if err != nil {
			t.Fatalf("Weg %d: %v", f.weg, err)
		}
		// Der Satz ist länger als die Zeile; verglichen wird ohne die Umbrüche des Blatts.
		blatt := strings.Join(pdftest.TexteInReihenfolge(t, roh), " ")
		if f.muss != "" && !strings.Contains(blatt, f.muss) {
			t.Errorf("Weg %d: auf dem Blatt fehlt %q:\n%s", f.weg, f.muss, blatt)
		}
		for _, d := range f.darfNich {
			if strings.Contains(blatt, d) {
				t.Errorf("Weg %d: auf dem Blatt steht %q:\n%s", f.weg, d, blatt)
			}
		}
	}
}

// Der Ort der Schule steht in der Datumszeile, wie er heißt. gofpdf druckt in cp1252: Ein Ort mit
// Umlaut, der nicht durch die Zeichenersetzung geht, stünde verstümmelt auf dem Brief.
func TestBestellanschreiben_OrtMitUmlautStehtLesbarInDerDatumszeile(t *testing.T) {
	roh, err := GenerateBestellanschreibenPDF(Bestellanschreiben{Betreff: "B", Vermerk: "V."},
		SchuleInfo{Name: "Schule", Strasse: "Weg 1", PLZ: "50667", Ort: "Köln"})
	if err != nil {
		t.Fatalf("Anschreiben drucken: %v", err)
	}
	pruefeBlatt(t, strings.Join(pdftest.Texte(t, roh), "\n"),
		[]string{"Köln, den " + schulzeit.Jetzt().Format(dateFormatDE)}, []string{"Ã"})
}
