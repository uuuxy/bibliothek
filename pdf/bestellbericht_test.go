package pdf

// Der Bestellbericht, geprüft am gedruckten Blatt. Der Erzeuger druckt die Blöcke seiner
// Eingabe; dass die Tür sie nach den Töpfen der Schule füllt, prüfen
// api/bestellung/bestellbericht_fuellen_test.go und bestellbericht_mittel_test.go.

import (
	"bytes"
	"slices"
	"strings"
	"testing"
	"time"

	"bibliothek/internal/pdftest"
	"bibliothek/pkg/betrag"
)

func berichtTestdaten() []BestellberichtTopf {
	return []BestellberichtTopf{{
		Beschriftung: "ohne Zuordnung",
		Bestellungen: []BestellberichtBestellung{
			{
				LieferantName:   "Cornelsen",
				Kundennummer:    "C-88123",
				Bestelldatum:    time.Date(2026, 3, 14, 0, 0, 0, 0, time.UTC),
				Gesamtbetrag:    5400,
				AnzahlExemplare: 2,
				Positionen: []BestellberichtPosition{
					{TitelName: "Alexander Gesamtausgabe", ISBN: "9783124912008", Menge: 2, Einzelpreis: 2700},
				},
			},
			{
				LieferantName:   "Klett",
				Kundennummer:    "K-5000",
				Bestelldatum:    time.Date(2026, 7, 2, 0, 0, 0, 0, time.UTC),
				Gesamtbetrag:    0,
				AnzahlExemplare: 4,
				Positionen: []BestellberichtPosition{
					{TitelName: "Ein Titel ganz ohne erfassten Preis", ISBN: "", Menge: 4, Einzelpreis: 0},
				},
			},
		},
	}}
}

// Der Bericht kennt zwei Betriebsarten, und die Spaltenbreiten sind in beiden von Hand gesetzt:
// gofpdf rechnet nichts nach. Ein Zahlendreher dort fällt sonst erst auf einem Blatt auf, das
// gerade jemandem vorgelegt wird.
func TestBestellbericht_ErzeugtGueltigesPDFInBeidenBetriebsarten(t *testing.T) {
	von := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	bis := time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)
	schule := SchuleInfo{Name: "Testbibliothek"}

	faelle := []struct {
		name          string
		jahresansicht bool
		mitPreisen    bool
	}{
		{"Detailliste mit Preisen", false, true},
		{"Detailliste ohne Preise", false, false},
		{"Jahresansicht mit Preisen", true, true},
		{"Jahresansicht ohne Preise", true, false},
	}

	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			daten, err := ErzeugeBestellbericht(Bestellbericht{
				Titel:         "Testbericht",
				Von:           von,
				Bis:           bis,
				Jahresansicht: f.jahresansicht,
				MitPreisen:    f.mitPreisen,
				Toepfe:        berichtTestdaten(),
			}, schule)
			if err != nil {
				t.Fatalf("PDF-Erzeugung fehlgeschlagen: %v", err)
			}
			if !bytes.HasPrefix(daten, []byte("%PDF-")) {
				t.Error("Ergebnis ist kein PDF")
			}
			if len(daten) < 1000 {
				t.Errorf("PDF ist mit %d Bytes verdächtig klein", len(daten))
			}
		})
	}
}

// Ohne Bestellungen scheitert der Bericht nicht und sagt, dass im Zeitraum keine liegen: Wer
// einen Zeitraum ohne Bestellung wählt, bekommt ein Blatt und keine Fehlermeldung. Ein Block
// ohne Bestellung zählt dabei wie kein Block.
func TestBestellbericht_OhneBestellungen(t *testing.T) {
	von := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	bis := time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC)

	for name, toepfe := range map[string][]BestellberichtTopf{
		"kein Block":       nil,
		"ein leerer Block": {{Beschriftung: "Topf-Probe", Bestellungen: nil}},
	} {
		for _, mitPreisen := range []bool{true, false} {
			daten, err := ErzeugeBestellbericht(Bestellbericht{
				Titel:         "Leer",
				Von:           von,
				Bis:           bis,
				Jahresansicht: true,
				MitPreisen:    mitPreisen,
				Toepfe:        toepfe,
			}, SchuleInfo{Name: "Testbibliothek"})
			if err != nil {
				t.Fatalf("%s, mit Preisen %v: %v", name, mitPreisen, err)
			}
			text := pdftest.Suchtext(t, daten)
			if !strings.Contains(text, "Keine Bestellungen im gew") {
				t.Errorf("%s, mit Preisen %v: der Satz „Keine Bestellungen im gewählten Zeitraum.“ fehlt", name, mitPreisen)
			}
			if strings.Contains(text, "Topf-Probe") || strings.Contains(text, "Bestellungen im Detail") {
				t.Errorf("%s, mit Preisen %v: das Blatt ohne Bestellung trägt einen Block", name, mitPreisen)
			}
		}
	}
}

// Jeder Block steht mit seiner Beschriftung, seinen Bestellungen und seiner Summe da, in der
// Folge der Eingabe, und darunter die Summe über alle. Ein Block ohne Bestellung bekommt weder
// Überschrift noch Summe.
func TestBestellbericht_JederBlockStehtMitBeschriftungUndSumme(t *testing.T) {
	bestellung := func(lieferant string, summe betrag.Cent, exemplare int) BestellberichtBestellung {
		return BestellberichtBestellung{
			LieferantName: lieferant, Kundennummer: "K-1", Bestelldatum: time.Date(2026, 9, 3, 10, 0, 0, 0, time.UTC),
			Gesamtbetrag: summe, AnzahlExemplare: exemplare,
			Positionen: []BestellberichtPosition{{TitelName: "Titel " + lieferant, ISBN: "978", Menge: exemplare, Einzelpreis: summe / betrag.Cent(exemplare)}},
		}
	}
	toepfe := []BestellberichtTopf{
		{Beschriftung: "Block-Zwei", Bestellungen: []BestellberichtBestellung{bestellung("Haendler-C", 4550, 5)}},
		{Beschriftung: "Block-Leer", Bestellungen: nil},
		{Beschriftung: "Block-Eins", Bestellungen: []BestellberichtBestellung{bestellung("Haendler-A", 12000, 30), bestellung("Haendler-B", 8000, 20)}},
	}
	for _, f := range []struct {
		mitPreisen bool
		want       []string
	}{
		{true, []string{
			"Bestellungen: 3", "Exemplare: 55", "Gesamtbetrag: 245,50 €", "Bestellungen im Detail",
			"Block-Zwei", "03.09.2026  ·  Haendler-C  ·  Kd.-Nr. K-1", "Summe (5 Exemplare)", "45,50 €",
			"Summe Block-Zwei (1 Bestellungen, 5 Exemplare)", "45,50 €",
			"Block-Eins", "03.09.2026  ·  Haendler-A  ·  Kd.-Nr. K-1", "Summe (30 Exemplare)", "120,00 €",
			"03.09.2026  ·  Haendler-B  ·  Kd.-Nr. K-1", "Summe (20 Exemplare)", "80,00 €",
			"Summe Block-Eins (2 Bestellungen, 50 Exemplare)", "200,00 €",
			"Gesamtbetrag 01.09.2026 – 30.09.2026", "245,50 €",
		}},
		{false, []string{
			"Bestellungen: 3", "Exemplare: 55", "Bestellungen im Detail",
			"Block-Zwei", "03.09.2026  ·  Haendler-C  ·  Kd.-Nr. K-1", "Summe: 5 Exemplare",
			"Summe Block-Zwei (1 Bestellungen, 5 Exemplare)",
			"Block-Eins", "03.09.2026  ·  Haendler-A  ·  Kd.-Nr. K-1", "Summe: 30 Exemplare",
			"03.09.2026  ·  Haendler-B  ·  Kd.-Nr. K-1", "Summe: 20 Exemplare",
			"Summe Block-Eins (2 Bestellungen, 50 Exemplare)",
			"Gesamt 01.09.2026 – 30.09.2026", "55",
		}},
	} {
		roh, err := ErzeugeBestellbericht(Bestellbericht{
			Titel:         "Bestellbericht",
			Von:           time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC),
			Bis:           time.Date(2026, 9, 30, 0, 0, 0, 0, time.UTC),
			Jahresansicht: false,
			MitPreisen:    f.mitPreisen,
			Toepfe:        toepfe,
		}, SchuleInfo{Name: "Testschule"})
		if err != nil {
			t.Fatalf("mit Preisen %v: Bericht erzeugen: %v", f.mitPreisen, err)
		}
		texte := pdftest.TexteInReihenfolge(t, roh)
		// Jeder erwartete Text steht als eigene Zelle hinter dem vorigen: So fällt auch ein Block
		// an falscher Stelle oder eine Summe unter dem falschen Block auf.
		stelle := 0
		for _, want := range f.want {
			gefunden := slices.Index(texte[stelle:], want)
			if gefunden >= 0 {
				gefunden += stelle
			}
			if gefunden < 0 {
				t.Errorf("mit Preisen %v: %q steht nicht hinter den Texten davor; gedruckt ab dort: %q", f.mitPreisen, want, texte[stelle:])
				break
			}
			stelle = gefunden + 1
		}
		text := strings.Join(texte, "\n")
		if strings.Contains(text, "Block-Leer") {
			t.Errorf("mit Preisen %v: der Block ohne Bestellung steht auf dem Blatt", f.mitPreisen)
		}
		// Ohne Jahresansicht stehen die zwei Übersichten nicht da.
		if strings.Contains(text, "nach Monat") || strings.Contains(text, "nach Lieferant") {
			t.Errorf("mit Preisen %v: der Bericht ohne Jahresansicht trägt eine Übersicht", f.mitPreisen)
		}
		// Die Zeile einer Position: Titel, ISBN, Menge, mit Preisen dazu Einzelpreis und Betrag der Zeile.
		zeile := []string{"Titel Haendler-C", "978", "5"}
		if f.mitPreisen {
			zeile = append(zeile, "9,10 €", "45,50 €")
		}
		zeile = append(zeile, f.want[slices.Index(f.want, "03.09.2026  ·  Haendler-C  ·  Kd.-Nr. K-1")+1])
		i := slices.Index(texte, zeile[0])
		if i < 0 || i+len(zeile) > len(texte) || !slices.Equal(texte[i:i+len(zeile)], zeile) {
			t.Errorf("mit Preisen %v: die Zeile der Position steht nicht als %q da: %q", f.mitPreisen, zeile, texte)
		}
	}
}

// Die Übersicht nach Lieferant der Jahresansicht nennt die Lieferanten nach dem Namen
// geordnet, ohne Rücksicht auf Groß- und Kleinschreibung. Gesammelt werden sie in einer Map;
// ohne feste Ordnung stünden sie bei jedem Druck desselben Berichts anders untereinander.
func TestBestellbericht_LieferantenStehenNachNamenGeordnet(t *testing.T) {
	// Acht Lieferanten in der Reihenfolge ihrer Bestellungen: Dass eine Map sie zufällig
	// geordnet ausgibt, kommt einmal in 40.320 Läufen vor.
	namen := []string{"Klett", "cornelsen", "Westermann", "Buchner", "Auer", "Schroedel", "Diesterweg", "Oldenbourg"}
	want := []string{"Auer", "Buchner", "cornelsen", "Diesterweg", "Klett", "Oldenbourg", "Schroedel", "Westermann"}
	bestellungen := make([]BestellberichtBestellung, 0, len(namen))
	for i, name := range namen {
		bestellungen = append(bestellungen, BestellberichtBestellung{
			LieferantName: name, Bestelldatum: time.Date(2026, time.Month(i+1), 10, 0, 0, 0, 0, time.UTC),
			Gesamtbetrag: betrag.Cent(1000 * (i + 1)), AnzahlExemplare: i + 1,
		})
	}

	for _, f := range []struct {
		ueberschrift string
		mitPreisen   bool
	}{{"Ausgaben nach Lieferant", true}, {"Bestellungen nach Lieferant", false}} {
		roh, err := ErzeugeBestellbericht(Bestellbericht{
			Titel: "Jahresbericht",
			Von:   time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), Bis: time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC),
			Jahresansicht: true, MitPreisen: f.mitPreisen,
			Toepfe: []BestellberichtTopf{{Beschriftung: "ohne Zuordnung", Bestellungen: bestellungen}},
		}, SchuleInfo{Name: "Testbibliothek"})
		if err != nil {
			t.Fatalf("%s: Bericht drucken: %v", f.ueberschrift, err)
		}
		texte := pdftest.TexteInReihenfolge(t, roh)
		anfang := slices.Index(texte, f.ueberschrift)
		if anfang < 0 {
			t.Fatalf("die Überschrift %q steht nicht auf dem Blatt", f.ueberschrift)
		}
		// Die ersten acht Zellen nach der Überschrift, die genau einen Lieferantennamen tragen,
		// sind die Zeilen der Übersicht; die Detailliste folgt dahinter.
		var gedruckt []string
		for _, text := range texte[anfang:] {
			if slices.Contains(namen, text) && len(gedruckt) < len(namen) {
				gedruckt = append(gedruckt, text)
			}
		}
		if !slices.Equal(gedruckt, want) {
			t.Errorf("%s: Lieferanten in der Reihenfolge %q, erwartet %q", f.ueberschrift, gedruckt, want)
		}
	}
}

// Die Übersichten der Jahresansicht zählen die Bestellungen aller Blöcke: Der Monat und der
// Lieferant einer Bestellung hängen nicht an ihrem Topf.
func TestBestellbericht_UebersichtenZaehlenUeberAlleBloecke(t *testing.T) {
	bestellung := func(monat time.Month, summe betrag.Cent, exemplare int) BestellberichtBestellung {
		return BestellberichtBestellung{
			LieferantName: "Haendler-A", Kundennummer: "K-1", Bestelldatum: time.Date(2026, monat, 10, 0, 0, 0, 0, time.UTC),
			Gesamtbetrag: summe, AnzahlExemplare: exemplare,
		}
	}
	roh, err := ErzeugeBestellbericht(Bestellbericht{
		Titel: "Jahresbericht",
		Von:   time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), Bis: time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC),
		Jahresansicht: true, MitPreisen: true,
		Toepfe: []BestellberichtTopf{
			{Beschriftung: "Block-Eins", Bestellungen: []BestellberichtBestellung{bestellung(time.March, 10000, 10), bestellung(time.May, 725, 3)}},
			{Beschriftung: "Block-Zwei", Bestellungen: []BestellberichtBestellung{bestellung(time.March, 2050, 2)}},
		},
	}, SchuleInfo{Name: "Testbibliothek"})
	if err != nil {
		t.Fatalf("Bericht drucken: %v", err)
	}
	texte := pdftest.TexteInReihenfolge(t, roh)
	zeile := func(kopf string) []string {
		i := slices.Index(texte, kopf)
		if i < 0 || i+3 >= len(texte) {
			t.Fatalf("die Zeile %q steht nicht auf dem Blatt: %q", kopf, texte)
		}
		return texte[i+1 : i+4]
	}
	// Je Zeile: Bestellungen, Exemplare, Betrag.
	if ist, will := zeile("März"), []string{"2", "12", "120,50 €"}; !slices.Equal(ist, will) {
		t.Errorf("März: %q, erwartet %q", ist, will)
	}
	if ist, will := zeile("Mai"), []string{"1", "3", "7,25 €"}; !slices.Equal(ist, will) {
		t.Errorf("Mai: %q, erwartet %q", ist, will)
	}
	// Die Zeile des Lieferanten trägt Bestellungen und Betrag über beide Blöcke.
	anfang := slices.Index(texte, "Ausgaben nach Lieferant")
	if anfang < 0 {
		t.Fatal("die Übersicht nach Lieferant fehlt")
	}
	i := slices.Index(texte[anfang:], "Haendler-A")
	if i < 0 || !slices.Equal(texte[anfang+i+1:anfang+i+3], []string{"3", "127,75 €"}) {
		t.Errorf("Lieferant über beide Blöcke: %q, erwartet 3 Bestellungen und 127,75 €", texte[anfang:])
	}
}

// Ohne Preise nennen die Übersichten Exemplare statt Beträge, und die Übersicht nach Lieferant
// heißt nicht „Ausgaben".
func TestBestellbericht_UebersichtenOhnePreiseNennenExemplare(t *testing.T) {
	bestellung := func(monat time.Month, exemplare int) BestellberichtBestellung {
		return BestellberichtBestellung{
			LieferantName: "Haendler-A", Kundennummer: "K-1", Bestelldatum: time.Date(2026, monat, 10, 0, 0, 0, 0, time.UTC),
			Gesamtbetrag: 99900, AnzahlExemplare: exemplare,
		}
	}
	roh, err := ErzeugeBestellbericht(Bestellbericht{
		Titel: "Jahresbericht",
		Von:   time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), Bis: time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC),
		Jahresansicht: true, MitPreisen: false,
		Toepfe: []BestellberichtTopf{
			{Beschriftung: "Block-Eins", Bestellungen: []BestellberichtBestellung{bestellung(time.March, 10), bestellung(time.May, 3)}},
			{Beschriftung: "Block-Zwei", Bestellungen: []BestellberichtBestellung{bestellung(time.March, 2)}},
		},
	}, SchuleInfo{Name: "Testbibliothek"})
	if err != nil {
		t.Fatalf("Bericht drucken: %v", err)
	}
	texte := pdftest.TexteInReihenfolge(t, roh)
	will := []string{
		"Übersicht nach Monat", "Monat", "Bestellungen", "Exemplare",
		"März", "2", "12", "Mai", "1", "3", "Gesamt", "3", "15",
		"Bestellungen nach Lieferant", "Lieferant", "Bestellungen", "Exemplare", "Haendler-A", "3", "15",
		"Bestellungen im Detail",
	}
	anfang := slices.Index(texte, will[0])
	if anfang < 0 || anfang+len(will) > len(texte) || !slices.Equal(texte[anfang:anfang+len(will)], will) {
		t.Errorf("die Übersichten ohne Preise stehen als %q da, erwartet %q", texte, will)
	}
	if text := strings.Join(texte, "\n"); strings.Contains(text, "€") || strings.Contains(text, "Ausgaben") {
		t.Errorf("der Mengenbericht nennt einen Betrag oder Ausgaben: %q", texte)
	}
}

// Eine Position ohne ISBN trägt in ihrer Spalte einen Strich: Eine leere Zelle läse sich wie
// eine vergessene Angabe.
func TestBestellbericht_PositionOhneISBNTraegtEinenStrich(t *testing.T) {
	roh, err := ErzeugeBestellbericht(Bestellbericht{
		Titel: "Testbericht", Von: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), Bis: time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC),
		Jahresansicht: false, MitPreisen: false,
		Toepfe: berichtTestdaten(),
	}, SchuleInfo{Name: "Testbibliothek"})
	if err != nil {
		t.Fatalf("Bericht drucken: %v", err)
	}
	texte := pdftest.TexteInReihenfolge(t, roh)
	i := slices.Index(texte, "Ein Titel ganz ohne erfassten Preis")
	if i < 0 || i+2 >= len(texte) || texte[i+1] != "—" || texte[i+2] != "4" {
		t.Errorf("hinter dem Titel ohne ISBN stehen nicht Strich und Menge: %q", texte)
	}
}

// Der Titel einer Position steht gekürzt mit Auslassungszeichen da, sobald er gedruckt breiter
// wäre als seine Spalte, mit und ohne Preisspalten. Nach Zeichen gekürzt lief ein gewöhnlicher
// langer Titel 4 mm in die Spalte der ISBN.
func TestBestellbericht_TitelBleibtInSeinerSpalte(t *testing.T) {
	const (
		lang  = "Seydlitz – Geographie Gymnasium Hessen, Schülerband für die Klassen 5 und 6, Ausgabe 2024 mit Arbeitsheft"
		gross = "DEUTSCHBUCH GYMNASIUM – ALLGEMEINE AUSGABE 2019, 5. SCHULJAHR, SCHÜLERBUCH MIT ARBEITSHEFT"
	)
	bestellungen := []BestellberichtBestellung{{
		LieferantName: "Cornelsen", Kundennummer: "C-1", Bestelldatum: time.Date(2026, 3, 14, 0, 0, 0, 0, time.UTC),
		Gesamtbetrag: 8100, AnzahlExemplare: 3,
		Positionen: []BestellberichtPosition{
			{TitelName: lang, ISBN: "ISBN-PROBE-1", Menge: 1, Einzelpreis: 2700},
			{TitelName: gross, ISBN: "ISBN-PROBE-2", Menge: 1, Einzelpreis: 2700},
			{TitelName: "Atlas", ISBN: "ISBN-PROBE-3", Menge: 1, Einzelpreis: 2700},
		},
	}}
	for _, mitPreisen := range []bool{true, false} {
		roh, err := ErzeugeBestellbericht(Bestellbericht{
			Titel: "Testbericht", Von: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), Bis: time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC),
			Jahresansicht: false, MitPreisen: mitPreisen,
			Toepfe: []BestellberichtTopf{{Beschriftung: "ohne Zuordnung", Bestellungen: bestellungen}},
		}, SchuleInfo{Name: "Testbibliothek"})
		if err != nil {
			t.Fatalf("mit Preisen %v: %v", mitPreisen, err)
		}
		platz := spaltenFuerBericht(mitPreisen).Titel - 2
		texte := pdftest.TexteInReihenfolge(t, roh)
		gefunden := 0
		for stelle, text := range texte {
			// Vor der ISBN steht der Titel der Position.
			switch text {
			case "ISBN-PROBE-1":
				gefunden++
				pdftest.InSpalte(t, texte[stelle-1], lang, 8, platz)
			case "ISBN-PROBE-2":
				gefunden++
				pdftest.InSpalte(t, texte[stelle-1], gross, 8, platz)
			case "ISBN-PROBE-3":
				gefunden++
				if texte[stelle-1] != "Atlas" {
					t.Errorf("mit Preisen %v: der kurze Titel steht als %q da", mitPreisen, texte[stelle-1])
				}
			}
		}
		if gefunden != 3 {
			t.Errorf("mit Preisen %v: %d von 3 Positionen gefunden", mitPreisen, gefunden)
		}
	}
}

// Das Bestelldatum steht am Kalendertag der Schule da, und die Übersicht zählt die Bestellung zu
// dessen Monat. Die Datenbank liefert den Zeitpunkt in UTC: 22:30 Uhr am 31.08. ist in der Schule
// der 01.09., eine halbe Stunde nach Mitternacht.
func TestBestellbericht_BestelldatumStehtAmKalendertagDerSchule(t *testing.T) {
	roh, err := ErzeugeBestellbericht(Bestellbericht{
		Titel: "Jahresbericht",
		Von:   time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), Bis: time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC),
		Jahresansicht: true, MitPreisen: false,
		Toepfe: []BestellberichtTopf{{Beschriftung: "Block-Eins", Bestellungen: []BestellberichtBestellung{{
			LieferantName: "Haendler-A", Kundennummer: "K-1", Bestelldatum: time.Date(2026, time.August, 31, 22, 30, 0, 0, time.UTC),
			Gesamtbetrag: 1000, AnzahlExemplare: 1,
		}}}},
	}, SchuleInfo{Name: "Testbibliothek"})
	if err != nil {
		t.Fatalf("Bericht drucken: %v", err)
	}
	texte := pdftest.TexteInReihenfolge(t, roh)
	if !slices.Contains(texte, "01.09.2026  ·  Haendler-A  ·  Kd.-Nr. K-1") {
		t.Errorf("die Kopfzeile der Bestellung nennt nicht den 01.09.2026: %q", texte)
	}
	if !slices.Contains(texte, "September") || slices.Contains(texte, "August") {
		t.Errorf("die Übersicht nach Monat zählt die Bestellung nicht im September: %q", texte)
	}
}
