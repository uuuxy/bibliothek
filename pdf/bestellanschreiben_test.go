package pdf

// Das Anschreiben an den Lieferanten, geprüft am gedruckten Blatt. Der Erzeuger kennt weder
// Töpfe noch die Anlagen der Mail: Er druckt Betreff und Vermerk seiner Eingabe und den Satz
// zum Weg der Etiketten. Dass die Tür sie richtig füllt, prüft api/bestellanschreiben_test.go.

import (
	"fmt"
	"math"
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

// Titel, Autor und ISBN stehen gekürzt mit Auslassungszeichen da, sobald sie gedruckt breiter
// wären als ihre Spalte; gofpdf druckt Überlanges über die Nachbarzelle.
func TestBestellanschreiben_TitelAutorUndISBNBleibenInIhrerSpalte(t *testing.T) {
	positionen := []BestellPosition{
		{Titel: strings.Repeat("W", 60), Autor: strings.Repeat("M", 30), ISBN: strings.Repeat("9", 40), Menge: 1001},
		{Titel: strings.Repeat("i", 80), Autor: strings.Repeat("l", 40), ISBN: "978-3-12-733771-9", Menge: 1002},
		{Titel: "Seydlitz – Geographie Gymnasium Hessen, Schülerband für die Klassen 5 und 6, Ausgabe 2024",
			Autor: "Annegret Müller-Lüdenscheidt und Kollegen", ISBN: "9783141004267", Menge: 1003},
		{Titel: "Kurz", Autor: "A. Utor", ISBN: "978", Menge: 1004},
	}
	roh, err := GenerateBestellanschreibenPDF(Bestellanschreiben{Betreff: "B", Vermerk: "V.", Positionen: positionen}, SchuleInfo{})
	if err != nil {
		t.Fatalf("Anschreiben drucken: %v", err)
	}
	texte := pdftest.TexteInReihenfolge(t, roh)
	gekuerzt := map[string]bool{}
	for stelle, text := range texte {
		for _, pos := range positionen {
			if text != fmt.Sprint(pos.Menge) {
				continue
			}
			// Vor der Menge stehen Titel, Autor und ISBN der Zeile.
			titel, autor, isbn := texte[stelle-3], texte[stelle-2], texte[stelle-1]
			pruefeInSpalte(t, titel, pos.Titel, 9, 73)
			pruefeInSpalte(t, autor, pos.Autor, 9, 38)
			pruefeInSpalte(t, isbn, pos.ISBN, 9, 33)
			gekuerzt[fmt.Sprint(pos.Menge)] = titel != pos.Titel || autor != pos.Autor || isbn != pos.ISBN
		}
	}
	if erwartet := map[string]bool{"1001": true, "1002": false, "1003": true, "1004": false}; fmt.Sprint(gekuerzt) != fmt.Sprint(erwartet) {
		t.Errorf("gekürzt: %v\nerwartet: %v", gekuerzt, erwartet)
	}
}

// Der Name der Schule steht am selben linken Rand wie der übrige Brief: 20 mm von der Kante,
// dazu der Innenabstand einer Zelle von 1 mm.
func TestBestellanschreiben_KopfStehtAmRandDesBriefs(t *testing.T) {
	roh, err := GenerateBestellanschreibenPDF(Bestellanschreiben{Betreff: "Betreff-Probe", Vermerk: "V."},
		SchuleInfo{Name: "Testschule", Strasse: "Schulweg 1", PLZ: "61381", Ort: "Friedrichsdorf"})
	if err != nil {
		t.Fatalf("Anschreiben drucken: %v", err)
	}
	seite := seitenMitOrten(t, roh)[0]
	const rand = (20.0 + 1.0) / 25.4 * 72
	for _, text := range []string{"Testschule", "An den Buchlieferanten", "Betreff-Probe", "Mit freundlichen Gr\xfc\xdfen,"} {
		links, da := seite.links[text]
		if !da {
			t.Errorf("%q steht nicht auf dem Blatt", text)
		} else if math.Abs(links-rand) > 0.5 {
			t.Errorf("%q beginnt %.1f mm von der Kante, erwartet 21", text, links/72*25.4)
		}
	}
}

// anschreibenTabelleAuf zählt die Positionen einer Seite und sagt, ob sie die Spaltenköpfe trägt.
// Verglichen wird das ganze Textstück: „Buchtitel" steht auch im Satz über der Tabelle.
func anschreibenTabelleAuf(seite []string) (zeilen int, koepfe bool) {
	for _, text := range seite {
		if strings.HasPrefix(text, "Titel ") {
			zeilen++
		}
		if text == "Buchtitel" {
			koepfe = true
		}
	}
	return zeilen, koepfe
}

// Eine Bestellung über mehrere Seiten: Jede Seite mit Positionen trägt die Spaltenköpfe, jede
// Position steht einmal da, und Gruß und Unterschrift stehen zusammen auf einer Seite.
func TestBestellanschreiben_LangeBestellungWiederholtDieKoepfe(t *testing.T) {
	// Jede Zahl bis über die dritte Seite: Irgendwo darunter liegt die, bei der der Gruß noch
	// passt und die Unterschrift nicht mehr.
	for anzahl := 1; anzahl <= 130; anzahl++ {
		positionen := make([]BestellPosition, 0, anzahl)
		for i := 1; i <= anzahl; i++ {
			positionen = append(positionen, BestellPosition{Titel: fmt.Sprintf("Titel %03d", i), Autor: "Autor", ISBN: "978", Menge: i})
		}
		roh, err := GenerateBestellanschreibenPDF(Bestellanschreiben{Betreff: "B", Vermerk: "V.", Positionen: positionen}, SchuleInfo{Name: "Testschule"})
		if err != nil {
			t.Fatalf("%d Positionen: %v", anzahl, err)
		}
		gedruckt, gruss := 0, 0
		for nr, seite := range pdftest.TexteJeSeite(t, roh) {
			text := strings.Join(seite, "\n")
			zeilen, koepfe := anschreibenTabelleAuf(seite)
			gedruckt += zeilen
			if koepfe != (zeilen > 0) {
				t.Errorf("%d Positionen, Seite %d: Spaltenköpfe = %v bei %d Zeilen", anzahl, nr+1, koepfe, zeilen)
			}
			hatGruss, hatTeam := strings.Contains(text, "Mit freundlichen Grüßen,"), strings.Contains(text, "Das Bibliotheksteam")
			if hatGruss != hatTeam {
				t.Errorf("%d Positionen, Seite %d: Gruß = %v, Unterschrift = %v — sie gehören zusammen", anzahl, nr+1, hatGruss, hatTeam)
			}
			if hatGruss {
				gruss++
			}
		}
		if gedruckt != anzahl || gruss != 1 {
			t.Errorf("%d Positionen: %d Zeilen gedruckt, Gruß %d-mal", anzahl, gedruckt, gruss)
		}
	}
}

// Schiebt ein langer Vermerk die Tabelle an den Fuß der Seite, beginnt sie auf der nächsten:
// Köpfe ohne Zeile darunter nennen Spalten ohne Inhalt.
func TestBestellanschreiben_KoepfeStehenNichtAlleinAmSeitenfuss(t *testing.T) {
	positionen := []BestellPosition{{Titel: "Titel 001", Autor: "Autor", ISBN: "978", Menge: 1}, {Titel: "Titel 002", Autor: "Autor", ISBN: "978", Menge: 2}}
	verschoben := 0
	for zeilen := 20; zeilen <= 45; zeilen++ {
		roh, err := GenerateBestellanschreibenPDF(Bestellanschreiben{Betreff: "B", Vermerk: strings.Repeat("Zeile des Vermerks\n", zeilen), Positionen: positionen}, SchuleInfo{})
		if err != nil {
			t.Fatalf("%d Zeilen: %v", zeilen, err)
		}
		seiten := pdftest.TexteJeSeite(t, roh)
		for nr, seite := range seiten {
			if positionenAuf, koepfe := anschreibenTabelleAuf(seite); koepfe != (positionenAuf > 0) {
				t.Errorf("Vermerk mit %d Zeilen, Seite %d: Spaltenköpfe = %v bei %d Positionen", zeilen, nr+1, koepfe, positionenAuf)
			}
		}
		if _, koepfeAufDerErsten := anschreibenTabelleAuf(seiten[0]); len(seiten) > 1 && !koepfeAufDerErsten {
			verschoben++
		}
	}
	// Ohne einen Fall, in dem die Tabelle auf die zweite Seite rückt, prüfte der Test nichts.
	if verschoben == 0 {
		t.Error("in keinem Fall begann die Tabelle auf der zweiten Seite")
	}
}
