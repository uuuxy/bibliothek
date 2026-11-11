package pdf

import (
	"bytes"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"bibliothek/internal/pdftest"
)

// Die Blätter von Zugangs- und Abgangsbuch, geprüft am gedruckten Text. Die Erzeuger kennen
// weder Töpfe noch Abfragen: Sie drucken die Abschnitte, Überschriften und Zahlen, die ihre
// Eingabe trägt. Dass die Tür sie richtig füllt, prüft api/katalog/bestandsbuch_test.go.

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
	roh, err := ErzeugeAbgangsbuch(Abgangsbuch{
		Von: buchVon, Bis: buchBis,
		Abschnitte: []AbgangsAbschnitt{
			{Titel: "Topf Eins", Exemplare: 1, Zeilen: []AbgangsZeile{
				{Datum: tag, Anzahl: 1, Barcodes: "B-00042", Titel: "Mathebuch 7", Signatur: "Mat 7", GrundText: "Verlust"}}},
			{Titel: "Topf Leer"},
			{Titel: "Topf Zwei", Exemplare: 2, Zeilen: []AbgangsZeile{
				{Datum: tag.AddDate(0, 0, 1), Anzahl: 1, Barcodes: "B-00815", Titel: "Gregs Tagebuch", Signatur: "Jug Gre", GrundText: "Aussortiert"},
				{Datum: tag.AddDate(0, 0, 2), Anzahl: 1, Barcodes: "B-00816", Titel: "Atlas", Signatur: "Erd 1", GrundText: "Sonstiges"}}},
		},
		Gesamt: 9, OhneZeitpunkt: 7, AusKatalogGeloescht: 3,
	}, SchuleInfo{Name: "Philipp-Reis-Schule", Strasse: "Schulstr. 1", PLZ: "61440", Ort: "Oberursel"})

	pruefeBlatt(t, buchText(t, roh, err), []string{
		"Abgangsbuch", "Philipp-Reis-Schule", "Zeitraum: 16.03.2026 bis 15.09.2026",
		"Topf Eins", "12.04.2026", "B-00042", "Mathebuch 7", "Mat 7", "Verlust",
		"Summe Topf Eins: 1 Exemplar",
		"Topf Zwei", "13.04.2026", "B-00815", "Aussortiert", "14.04.2026", "B-00816", "Sonstiges",
		"Summe Topf Zwei: 2 Exemplare",
		"Abgänge im Zeitraum: 9 Exemplare",
		"7 weitere Exemplare sind ausgesondert",
		"3 Exemplare wurden in diesem Zeitraum aus dem Katalog gelöscht",
	}, []string{
		"Topf Leer", // eine Überschrift ohne Zeilen hilft auf dem abgehefteten Blatt niemandem
		"kein Exemplar aus dem Bestand gegangen",
		"Eins: 1 Exemplare", // die Mehrzahl hinter der Eins stand hier fest im Satz
	})
}

// Ohne Abgänge sagt das Blatt es. Die Hinweise auf das, was in keiner Liste steht, bleiben.
func TestAbgangsbuch_LeererZeitraumSagtEsUndBehaeltDieHinweise(t *testing.T) {
	roh, err := ErzeugeAbgangsbuch(Abgangsbuch{
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
	roh, err := ErzeugeZugangsbuch(Zugangsbuch{
		Von: buchVon, Bis: buchBis,
		Abschnitte: []ZugangsAbschnitt{
			{Titel: "Topf Eins", Exemplare: 1, Zeilen: []ZugangsZeile{
				{Datum: tag, Anzahl: 1, Barcodes: "B-00100", Titel: "Mathebuch 7", Lieferant: "Buchhandlung Land"}}},
			{Titel: "Topf Leer"},
		},
		Gesamt: 5,
	}, SchuleInfo{Name: "Philipp-Reis-Schule"})

	pruefeBlatt(t, buchText(t, roh, err), []string{
		"Zugangsbuch", "Zeitraum: 16.03.2026 bis 15.09.2026",
		"Topf Eins", "03.05.2026", "B-00100", "Mathebuch 7", "Buchhandlung Land",
		"Summe Topf Eins: 1 Exemplar",
		"Zugänge im Zeitraum: 5 Exemplare",
	}, []string{"Topf Leer", "keine Bestellung hinterlegt", "kein Exemplar in den Bestand gekommen", "Eins: 1 Exemplare"})
}

// Den Abschnitt der Zugänge ohne hinterlegte Bestellung erklärt das Blatt genau dann, wenn die
// Eingabe es verlangt.
func TestZugangsbuch_ErklaertOhneZuordnungNurAufVerlangen(t *testing.T) {
	buch := Zugangsbuch{
		Von: buchVon, Bis: buchBis,
		Abschnitte: []ZugangsAbschnitt{{Titel: "ohne Zuordnung", Exemplare: 1, Zeilen: []ZugangsZeile{
			{Datum: buchVon, Anzahl: 1, Barcodes: "B-00300", Titel: "Fundstück aus dem Schrank"}}}},
		Gesamt: 1,
	}
	roh, err := ErzeugeZugangsbuch(buch, SchuleInfo{})
	pruefeBlatt(t, buchText(t, roh, err), []string{"Summe ohne Zuordnung: 1 Exemplar", "Zugänge im Zeitraum: 1 Exemplar"},
		[]string{"keine Bestellung hinterlegt", ": 1 Exemplare"})

	buch.OhneZuordnung = true
	roh, err = ErzeugeZugangsbuch(buch, SchuleInfo{})
	pruefeBlatt(t, buchText(t, roh, err), []string{"keine Bestellung hinterlegt", "Summe ohne Zuordnung: 1 Exemplar"}, nil)

	leer, err := ErzeugeZugangsbuch(Zugangsbuch{Von: buchVon, Bis: buchBis}, SchuleInfo{})
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
		abgaenge = append(abgaenge, AbgangsZeile{Datum: buchVon, Anzahl: 1, Barcodes: nummer, Titel: "Mathebuch", Signatur: "Mat 7", GrundText: "Verlust"})
		zugaenge = append(zugaenge, ZugangsZeile{Datum: buchVon, Anzahl: 1, Barcodes: nummer, Titel: "Mathebuch", Lieferant: "Buchhandlung"})
	}
	abgang, err := ErzeugeAbgangsbuch(Abgangsbuch{Von: buchVon, Bis: buchBis, Gesamt: anzahl,
		Abschnitte: []AbgangsAbschnitt{{Titel: "Topf Eins", Zeilen: abgaenge}}}, SchuleInfo{})
	if err != nil {
		t.Fatalf("Abgangsbuch drucken: %v", err)
	}
	zugang, err := ErzeugeZugangsbuch(Zugangsbuch{Von: buchVon, Bis: buchBis, Gesamt: anzahl,
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
	abgang, err := ErzeugeAbgangsbuch(Abgangsbuch{Von: buchVon, Bis: buchBis, Gesamt: 2,
		Abschnitte: []AbgangsAbschnitt{{Titel: "Topf Eins", Zeilen: []AbgangsZeile{
			{Datum: tag, Anzahl: 1, Barcodes: "B-81", Titel: gross, Signatur: signatur, GrundText: "Verlust"},
			{Datum: tag, Anzahl: 1, Barcodes: "B-82", Titel: "Atlas", Signatur: "Erd 1", GrundText: "Verlust"}}}}}, SchuleInfo{})
	if err != nil {
		t.Fatalf("Abgangsbuch drucken: %v", err)
	}
	zugang, err := ErzeugeZugangsbuch(Zugangsbuch{Von: buchVon, Bis: buchBis, Gesamt: 2,
		Abschnitte: []ZugangsAbschnitt{{Titel: "Topf Eins", Zeilen: []ZugangsZeile{
			{Datum: tag, Anzahl: 1, Barcodes: "B-91", Titel: gross, Lieferant: haendler},
			{Datum: tag, Anzahl: 1, Barcodes: "B-92", Titel: "Atlas", Lieferant: "Naacher"}}}}}, SchuleInfo{})
	if err != nil {
		t.Fatalf("Zugangsbuch drucken: %v", err)
	}

	// Die Zellen einer Zeile stehen in ihrer Reihenfolge um den Barcode herum: im Abgangsbuch
	// Titel, Signatur, Grund, Anzahl, Barcodes; im Zugangsbuch Titel, Anzahl, Barcodes, Lieferant.
	um := func(roh []byte, nummer string, erste, zweite int) (string, string) {
		texte := pdftest.TexteInReihenfolge(t, roh)
		for stelle, text := range texte {
			if text == nummer {
				return texte[stelle+erste], texte[stelle+zweite]
			}
		}
		t.Fatalf("%s steht nicht auf dem Blatt", nummer)
		return "", ""
	}
	nach := func(roh []byte, nummer string) (string, string) {
		if bytes.Equal(roh, abgang) {
			return um(roh, nummer, -4, -3)
		}
		return um(roh, nummer, -2, 1)
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

// Jede Seite des Zugangsbuchs trägt „Seite x von y", gezählt je Ausdruck: Am abgehefteten Blatt
// sieht man, ob eine Seite fehlt.
func TestZugangsbuch_ZaehltSeitenJeAusdruck(t *testing.T) {
	zeilen := make([]ZugangsZeile, 0, 120)
	for i := 0; i < cap(zeilen); i++ {
		zeilen = append(zeilen, ZugangsZeile{Datum: buchVon, Anzahl: 1, Barcodes: fmt.Sprintf("B-%05d", 80000+i), Titel: "Lesebuch", Lieferant: "Buchhandlung"})
	}
	for name, abschnitte := range map[string][]ZugangsAbschnitt{
		"leer": nil,
		"lang": {{Titel: "Topf Eins", Zeilen: zeilen}},
	} {
		roh, err := ErzeugeZugangsbuch(Zugangsbuch{Von: buchVon, Bis: buchBis, Abschnitte: abschnitte}, SchuleInfo{})
		if err != nil {
			t.Fatalf("%s: Zugangsbuch drucken: %v", name, err)
		}
		seiten := pdftest.TexteJeSeite(t, roh)
		for nr, seite := range seiten {
			soll := fmt.Sprintf("Seite %d von %d", nr+1, len(seiten))
			if !slices.Contains(seite, soll) {
				t.Errorf("%s, Seite %d: %q fehlt", name, nr+1, soll)
			}
		}
		if name == "lang" && len(seiten) < 2 {
			t.Errorf("lang: %d Seite, erwartet mehrere", len(seiten))
		}
	}
}

// Eine Zeile mit vielen Barcodes läuft in ihrer Spalte um; reicht eine Seite nicht, geht sie auf
// der nächsten unter den Spaltenköpfen weiter, mit Titel und Datum. Kein Barcode fehlt, keine
// gedruckte Zeile ist breiter als die Spalte, und ein Bereich bricht nicht um.
func TestBestandsbuch_BarcodesLaufenUmUndKeinerFehlt(t *testing.T) {
	const anzahl = 300
	nummern := make([]string, anzahl)
	for i := range nummern {
		nummern[i] = fmt.Sprintf("B-%05d", 60000+2*i) // jede zweite Nummer: kein Bereich
	}
	bereich := "B-50000\u00a0–\u00a0B-50044"
	roh, err := ErzeugeZugangsbuch(Zugangsbuch{Von: buchVon, Bis: buchBis, Gesamt: anzahl + 46,
		Abschnitte: []ZugangsAbschnitt{{Titel: "Topf Eins", Exemplare: anzahl + 46, Zeilen: []ZugangsZeile{
			{Datum: buchVon, Titel: "Lesebuch", Anzahl: 46, Barcodes: bereich + ", B-50050", Lieferant: "Naacher"},
			{Datum: buchVon, Titel: "Mathebuch", Anzahl: anzahl, Barcodes: strings.Join(nummern, ", "), Lieferant: "Naacher"},
		}}}}, SchuleInfo{})
	if err != nil {
		t.Fatalf("Zugangsbuch drucken: %v", err)
	}

	seiten := pdftest.TexteJeSeite(t, roh)
	if len(seiten) < 2 {
		t.Fatalf("%d Seite(n) für %d Barcodes in einer Zeile, erwartet mindestens 2", len(seiten), anzahl)
	}
	gesehen := map[string]bool{}
	bereichGesehen := false
	for nr, seite := range seiten {
		stuecke := 0
		for _, text := range seite {
			bereichGesehen = bereichGesehen || strings.HasPrefix(text, bereich)
			if !strings.HasPrefix(text, "B-6") {
				continue
			}
			stuecke++
			pdftest.InSpalte(t, text, text, 9, zugangSpalteBarcodes-2)
			for _, n := range strings.Split(text, ",") {
				if n = strings.TrimSpace(n); n != "" {
					gesehen[n] = true
				}
			}
		}
		if stuecke > 0 && (!slices.Contains(seite, "Barcodes") || !slices.Contains(seite, "Mathebuch")) {
			t.Errorf("Seite %d: %d Zeilen mit Barcodes ohne Spaltenköpfe oder ohne Titel der Zeile", nr+1, stuecke)
		}
	}
	for _, n := range nummern {
		if !gesehen[n] {
			t.Errorf("%s fehlt auf dem Blatt", n)
		}
	}
	if !bereichGesehen {
		t.Errorf("der Bereich %q steht nicht zusammen in einer Zeile", bereich)
	}
}
