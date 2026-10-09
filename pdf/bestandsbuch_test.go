package pdf

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"bibliothek/internal/pdftest"
)

// Die Blätter von Zugangs- und Abgangsbuch, geprüft am gedruckten Text. Die Erzeuger kennen
// weder Töpfe noch Abfragen: Sie drucken die Abschnitte, Überschriften und Zahlen, die ihre
// Eingabe trägt. Dass die Tür sie richtig füllt, prüft api/bestandsbuch_test.go.

var (
	buchVon = time.Date(2026, time.March, 16, 0, 0, 0, 0, time.UTC)
	buchBis = time.Date(2026, time.September, 15, 0, 0, 0, 0, time.UTC)
)

// buchText liefert den gedruckten Text eines Blatts; ein Fehler des Erzeugers bricht den Test ab.
func buchText(t *testing.T, roh []byte, err error) string {
	t.Helper()
	if err != nil {
		t.Fatalf("Blatt drucken: %v", err)
	}
	return blattText(t, roh)
}

func pruefeBlatt(t *testing.T, blatt string, muss, darfNicht []string) {
	t.Helper()
	for _, m := range muss {
		if !strings.Contains(blatt, m) {
			t.Errorf("auf dem Blatt fehlt %q:\n%s", m, blatt)
		}
	}
	for _, d := range darfNicht {
		if strings.Contains(blatt, d) {
			t.Errorf("auf dem Blatt steht %q und gehört dort nicht hin:\n%s", d, blatt)
		}
	}
}

// Die Gesamtzahl weicht mit Absicht von der Zahl der Zeilen ab: Das Blatt druckt die Zahl der
// Eingabe und zählt nicht selbst nach.
func TestAbgangsbuch_DrucktAbschnitteZahlenUndHinweise(t *testing.T) {
	tag := time.Date(2026, time.April, 12, 10, 0, 0, 0, time.UTC)
	roh, err := GenerateAbgangsbuchPDF(Abgangsbuch{
		Von: buchVon, Bis: buchBis,
		Abschnitte: []AbgangsAbschnitt{
			{Titel: "Topf Eins", Zeilen: []AbgangsZeile{
				{Datum: tag, Barcode: "B-00042", Titel: "Mathebuch 7", Signatur: "Mat 7", GrundText: "Verlust"}}},
			{Titel: "Topf Leer"},
			{Titel: "Topf Zwei", Zeilen: []AbgangsZeile{
				{Datum: tag.AddDate(0, 0, 1), Barcode: "B-00815", Titel: "Gregs Tagebuch", Signatur: "Jug Gre", GrundText: "Aussortiert"},
				{Datum: tag.AddDate(0, 0, 2), Barcode: "B-00816", Titel: "Atlas", Signatur: "Erd 1", GrundText: "Sonstiges"}}},
		},
		Gesamt: 9, OhneZeitpunkt: 7, AusKatalogGeloescht: 3,
	}, SchuleInfo{Name: "Philipp-Reis-Schule", Strasse: "Schulstr. 1", PLZ: "61440", Ort: "Oberursel"})

	pruefeBlatt(t, buchText(t, roh, err), []string{
		"Abgangsbuch", "Philipp-Reis-Schule", "Zeitraum: 16.03.2026 bis 15.09.2026",
		"Topf Eins", "12.04.2026", "B-00042", "Mathebuch 7", "Mat 7", "Verlust",
		"Summe Topf Eins: 1 Exemplare",
		"Topf Zwei", "13.04.2026", "B-00815", "Aussortiert", "14.04.2026", "B-00816", "Sonstiges",
		"Summe Topf Zwei: 2 Exemplare",
		"Abgänge im Zeitraum: 9 Exemplare",
		"7 weitere Exemplare sind ausgesondert",
		"3 Exemplare wurden in diesem Zeitraum aus dem Katalog gelöscht",
	}, []string{
		"Topf Leer", // eine Überschrift ohne Zeilen hilft auf dem abgehefteten Blatt niemandem
		"kein Exemplar aus dem Bestand gegangen",
	})
}

// Ohne Abgänge sagt das Blatt es. Die Hinweise auf das, was in keiner Liste steht, bleiben.
func TestAbgangsbuch_LeererZeitraumSagtEsUndBehaeltDieHinweise(t *testing.T) {
	roh, err := GenerateAbgangsbuchPDF(Abgangsbuch{
		Von: buchVon, Bis: buchBis,
		Abschnitte:    []AbgangsAbschnitt{{Titel: "Topf Eins"}, {Titel: "Topf Zwei"}},
		OhneZeitpunkt: 4,
	}, SchuleInfo{Name: "Philipp-Reis-Schule"})

	pruefeBlatt(t, buchText(t, roh, err), []string{
		"kein Exemplar aus dem Bestand gegangen",
		"Abgänge im Zeitraum: 0 Exemplare",
		"4 weitere Exemplare sind ausgesondert",
	}, []string{"Topf Eins", "Summe", "aus dem Katalog gelöscht"})
}

func TestZugangsbuch_DrucktAbschnitteUndZahlen(t *testing.T) {
	tag := time.Date(2026, time.May, 3, 0, 0, 0, 0, time.UTC)
	roh, err := GenerateZugangsbuchPDF(Zugangsbuch{
		Von: buchVon, Bis: buchBis,
		Abschnitte: []ZugangsAbschnitt{
			{Titel: "Topf Eins", Zeilen: []ZugangsZeile{
				{Datum: tag, Barcode: "B-00100", Titel: "Mathebuch 7", Lieferant: "Buchhandlung Land"}}},
			{Titel: "Topf Leer"},
		},
		Gesamt: 5,
	}, SchuleInfo{Name: "Philipp-Reis-Schule"})

	pruefeBlatt(t, buchText(t, roh, err), []string{
		"Zugangsbuch", "Zeitraum: 16.03.2026 bis 15.09.2026",
		"Topf Eins", "03.05.2026", "B-00100", "Mathebuch 7", "Buchhandlung Land",
		"Summe Topf Eins: 1 Exemplare",
		"Zugänge im Zeitraum: 5 Exemplare",
	}, []string{"Topf Leer", "keine Bestellung hinterlegt", "kein Exemplar in den Bestand gekommen"})
}

// Den Abschnitt der Zugänge ohne hinterlegte Bestellung erklärt das Blatt genau dann, wenn die
// Eingabe es verlangt.
func TestZugangsbuch_ErklaertOhneZuordnungNurAufVerlangen(t *testing.T) {
	buch := Zugangsbuch{
		Von: buchVon, Bis: buchBis,
		Abschnitte: []ZugangsAbschnitt{{Titel: "ohne Zuordnung", Zeilen: []ZugangsZeile{
			{Datum: buchVon, Barcode: "B-00300", Titel: "Fundstück aus dem Schrank"}}}},
		Gesamt: 1,
	}
	roh, err := GenerateZugangsbuchPDF(buch, SchuleInfo{})
	pruefeBlatt(t, buchText(t, roh, err), []string{"Summe ohne Zuordnung: 1 Exemplare"}, []string{"keine Bestellung hinterlegt"})

	buch.OhneZuordnung = true
	roh, err = GenerateZugangsbuchPDF(buch, SchuleInfo{})
	pruefeBlatt(t, buchText(t, roh, err), []string{"keine Bestellung hinterlegt", "Summe ohne Zuordnung: 1 Exemplare"}, nil)

	leer, err := GenerateZugangsbuchPDF(Zugangsbuch{Von: buchVon, Bis: buchBis}, SchuleInfo{})
	pruefeBlatt(t, buchText(t, leer, err), []string{"kein Exemplar in den Bestand gekommen", "Zugänge im Zeitraum: 0 Exemplare"}, nil)
}

// Ein Buch über mehrere Seiten: Jede Seite mit Zeilen trägt die Spaltenköpfe, und keine Zeile
// geht verloren. Ein abgeheftetes Folgeblatt ohne Köpfe wäre eine Liste von Nummern.
func TestBestandsbuch_JedeSeiteMitZeilenTraegtDieSpaltenkoepfe(t *testing.T) {
	const anzahl = 120
	abgaenge := make([]AbgangsZeile, 0, anzahl)
	zugaenge := make([]ZugangsZeile, 0, anzahl)
	for i := 0; i < anzahl; i++ {
		nummer := fmt.Sprintf("B-%05d", 70000+i)
		// Keine Zelle einer Zeile trägt den Wortlaut eines Spaltenkopfs: Sonst hielte der Test
		// die Zeile für den Kopf.
		abgaenge = append(abgaenge, AbgangsZeile{Datum: buchVon, Barcode: nummer, Titel: "Mathebuch", Signatur: "Mat 7", GrundText: "Verlust"})
		zugaenge = append(zugaenge, ZugangsZeile{Datum: buchVon, Barcode: nummer, Titel: "Mathebuch", Lieferant: "Buchhandlung"})
	}
	abgang, err := GenerateAbgangsbuchPDF(Abgangsbuch{Von: buchVon, Bis: buchBis, Gesamt: anzahl,
		Abschnitte: []AbgangsAbschnitt{{Titel: "Topf Eins", Zeilen: abgaenge}}}, SchuleInfo{})
	if err != nil {
		t.Fatalf("Abgangsbuch drucken: %v", err)
	}
	zugang, err := GenerateZugangsbuchPDF(Zugangsbuch{Von: buchVon, Bis: buchBis, Gesamt: anzahl,
		Abschnitte: []ZugangsAbschnitt{{Titel: "Topf Eins", Zeilen: zugaenge}}}, SchuleInfo{})
	if err != nil {
		t.Fatalf("Zugangsbuch drucken: %v", err)
	}

	for name, f := range map[string]struct {
		roh  []byte
		kopf string
	}{"Abgangsbuch": {abgang, "Grund"}, "Zugangsbuch": {zugang, "Lieferant"}} {
		seiten := pdftest.TexteJeSeite(t, f.roh)
		if len(seiten) < 3 {
			t.Errorf("%s: %d Seiten für %d Zeilen, erwartet mindestens 3", name, len(seiten), anzahl)
		}
		gedruckt := 0
		for nr, seite := range seiten {
			zeilen := 0
			koepfe := false
			for _, text := range seite {
				if strings.HasPrefix(text, "B-7") {
					zeilen++
				}
				if text == f.kopf {
					koepfe = true
				}
			}
			gedruckt += zeilen
			if zeilen > 0 && !koepfe {
				t.Errorf("%s, Seite %d: %d Zeilen ohne Spaltenköpfe", name, nr+1, zeilen)
			}
		}
		if gedruckt != anzahl {
			t.Errorf("%s: %d Zeilen gedruckt, erwartet %d", name, gedruckt, anzahl)
		}
	}
}

// Titel, Signatur und Lieferant stehen gekürzt mit Auslassungszeichen da, sobald sie gedruckt
// breiter wären als ihre Spalte: Gemessen wird die Breite, nicht die Zahl der Zeichen. Ein Titel
// in Großbuchstaben und die Signatur eines Lernmittels liefen sonst über die Nachbarzelle.
func TestBestandsbuecher_TitelSignaturUndLieferantBleibenInIhrerSpalte(t *testing.T) {
	const (
		gross    = "DEUTSCHBUCH GYMNASIUM – ALLGEMEINE AUSGABE 2019, 5. SCHULJAHR, SCHÜLERBUCH MIT ARBEITSHEFT"
		signatur = "LMF-Gesellschaftslehre 10"
		haendler = "BUCHHANDLUNG AM MARKT, INHABERIN ANNEGRET MÜLLER-LÜDENSCHEIDT E. K."
	)
	tag := time.Date(2026, time.April, 12, 10, 0, 0, 0, time.UTC)
	abgang, err := GenerateAbgangsbuchPDF(Abgangsbuch{Von: buchVon, Bis: buchBis, Gesamt: 2,
		Abschnitte: []AbgangsAbschnitt{{Titel: "Topf Eins", Zeilen: []AbgangsZeile{
			{Datum: tag, Barcode: "B-81", Titel: gross, Signatur: signatur, GrundText: "Verlust"},
			{Datum: tag, Barcode: "B-82", Titel: "Atlas", Signatur: "Erd 1", GrundText: "Verlust"}}}}}, SchuleInfo{})
	if err != nil {
		t.Fatalf("Abgangsbuch drucken: %v", err)
	}
	zugang, err := GenerateZugangsbuchPDF(Zugangsbuch{Von: buchVon, Bis: buchBis, Gesamt: 2,
		Abschnitte: []ZugangsAbschnitt{{Titel: "Topf Eins", Zeilen: []ZugangsZeile{
			{Datum: tag, Barcode: "B-91", Titel: gross, Lieferant: haendler},
			{Datum: tag, Barcode: "B-92", Titel: "Atlas", Lieferant: "Naacher"}}}}}, SchuleInfo{})
	if err != nil {
		t.Fatalf("Zugangsbuch drucken: %v", err)
	}

	// Hinter der Nummer stehen Titel und Signatur oder Lieferant der Zeile.
	nach := func(roh []byte, nummer string) (string, string) {
		texte := pdftest.TexteInReihenfolge(t, roh)
		for stelle, text := range texte {
			if text == nummer {
				return texte[stelle+1], texte[stelle+2]
			}
		}
		t.Fatalf("%s steht nicht auf dem Blatt", nummer)
		return "", ""
	}
	titel, sig := nach(abgang, "B-81")
	pdftest.InSpalte(t, titel, gross, 9, abgangSpalteTitel-2)
	pdftest.InSpalte(t, sig, signatur, 9, abgangSpalteSignatur-2)
	if titel == gross || sig == signatur {
		t.Errorf("Abgangsbuch: Titel %q und Signatur %q stehen ungekürzt da", titel, sig)
	}
	if titel, sig = nach(abgang, "B-82"); titel != "Atlas" || sig != "Erd 1" {
		t.Errorf("Abgangsbuch: kurze Texte stehen als %q und %q da", titel, sig)
	}

	titel, lieferant := nach(zugang, "B-91")
	pdftest.InSpalte(t, titel, gross, 9, zugangSpalteTitel-2)
	pdftest.InSpalte(t, lieferant, haendler, 9, zugangSpalteLieferant-2)
	if titel == gross || lieferant == haendler {
		t.Errorf("Zugangsbuch: Titel %q und Lieferant %q stehen ungekürzt da", titel, lieferant)
	}
	if titel, lieferant = nach(zugang, "B-92"); titel != "Atlas" || lieferant != "Naacher" {
		t.Errorf("Zugangsbuch: kurze Texte stehen als %q und %q da", titel, lieferant)
	}
}
