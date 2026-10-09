package api

import (
	"bytes"
	"slices"
	"testing"
	"time"

	"bibliothek/internal/pdftest"
	"bibliothek/pdf"
	"bibliothek/repository"
)

func berichtTestdaten() []repository.BerichtBestellung {
	return []repository.BerichtBestellung{
		{
			LieferantName:   "Cornelsen",
			Kundennummer:    "C-88123",
			Bestelldatum:    time.Date(2026, 3, 14, 0, 0, 0, 0, time.UTC),
			Gesamtbetrag:    54.00,
			AnzahlExemplare: 2,
			Positionen: []repository.BerichtPosition{
				{TitelName: "Alexander Gesamtausgabe", ISBN: "9783124912008", Menge: 2, Einzelpreis: 27.00},
			},
		},
		{
			LieferantName:   "Klett",
			Kundennummer:    "K-5000",
			Bestelldatum:    time.Date(2026, 7, 2, 0, 0, 0, 0, time.UTC),
			Gesamtbetrag:    0,
			AnzahlExemplare: 4,
			Positionen: []repository.BerichtPosition{
				{TitelName: "Ein Titel ganz ohne erfassten Preis", ISBN: "", Menge: 4, Einzelpreis: 0},
			},
		},
	}
}

// Der Bericht kennt zwei Betriebsarten, und die Spaltenbreiten sind in beiden von Hand
// gesetzt — gofpdf rechnet nichts nach. Ein Zahlendreher dort faellt sonst erst dem
// Betreiber auf, auf einem Blatt, das er gerade jemandem vorlegt.
func TestBerichtErzeugtGueltigesPDFInBeidenBetriebsarten(t *testing.T) {
	von := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	bis := time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)
	schule := pdf.SchuleInfo{Name: "Testbibliothek"}

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
			opts := bestellBerichtOpts{
				Titel:         "Testbericht",
				Von:           von,
				Bis:           bis,
				Jahresansicht: f.jahresansicht,
				MitPreisen:    f.mitPreisen,
			}
			daten, err := generateBestellBerichtPDF(berichtTestdaten(), schule, opts)
			if err != nil {
				t.Fatalf("PDF-Erzeugung fehlgeschlagen: %v", err)
			}
			if !bytes.HasPrefix(daten, []byte("%PDF-")) {
				t.Error("Ergebnis ist kein PDF")
			}
			if len(daten) < 1000 {
				t.Errorf("PDF ist mit %d Bytes verdaechtig klein", len(daten))
			}
		})
	}
}

// Ohne Bestellungen darf der Bericht nicht scheitern — der Betreiber waehlt auch mal einen
// Zeitraum, in dem nichts passiert ist.
func TestBerichtOhneBestellungen(t *testing.T) {
	von := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	bis := time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC)

	for _, mitPreisen := range []bool{true, false} {
		opts := bestellBerichtOpts{
			Titel:         "Leer",
			Von:           von,
			Bis:           bis,
			Jahresansicht: true,
			MitPreisen:    mitPreisen,
		}
		daten, err := generateBestellBerichtPDF(nil, pdf.SchuleInfo{Name: "Testbibliothek"}, opts)
		if err != nil {
			t.Fatalf("mitPreisen=%v: %v", mitPreisen, err)
		}
		if !bytes.HasPrefix(daten, []byte("%PDF-")) {
			t.Errorf("mitPreisen=%v: Ergebnis ist kein PDF", mitPreisen)
		}
	}
}

// Die Übersicht nach Lieferant der Jahresansicht nennt die Lieferanten nach dem Namen
// geordnet, ohne Rücksicht auf Groß- und Kleinschreibung. Gesammelt werden sie in einer Map;
// ohne feste Ordnung stünden sie bei jedem Druck desselben Berichts anders untereinander.
func TestBericht_LieferantenStehenNachNamenGeordnet(t *testing.T) {
	// Acht Lieferanten in der Reihenfolge ihrer Bestellungen: Dass eine Map sie zufällig
	// geordnet ausgibt, kommt einmal in 40.320 Läufen vor.
	namen := []string{"Klett", "cornelsen", "Westermann", "Buchner", "Auer", "Schroedel", "Diesterweg", "Oldenbourg"}
	want := []string{"Auer", "Buchner", "cornelsen", "Diesterweg", "Klett", "Oldenbourg", "Schroedel", "Westermann"}
	orders := make([]repository.BerichtBestellung, 0, len(namen))
	for i, name := range namen {
		orders = append(orders, repository.BerichtBestellung{
			LieferantName: name, Bestelldatum: time.Date(2026, time.Month(i+1), 10, 0, 0, 0, 0, time.UTC),
			Gesamtbetrag: float64(10 * (i + 1)), AnzahlExemplare: i + 1,
		})
	}

	for _, f := range []struct {
		ueberschrift string
		mitPreisen   bool
	}{{"Ausgaben nach Lieferant", true}, {"Bestellungen nach Lieferant", false}} {
		roh, err := generateBestellBerichtPDF(orders, pdf.SchuleInfo{Name: "Testbibliothek"}, bestellBerichtOpts{
			Titel: "Jahresbericht",
			Von:   time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), Bis: time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC),
			Jahresansicht: true, MitPreisen: f.mitPreisen,
		})
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

// Der Titel einer Position steht gekürzt mit Auslassungszeichen da, sobald er gedruckt breiter
// wäre als seine Spalte, mit und ohne Preisspalten. Nach Zeichen gekürzt lief ein gewöhnlicher
// langer Titel 4 mm in die Spalte der ISBN.
func TestBericht_TitelBleibtInSeinerSpalte(t *testing.T) {
	const (
		lang  = "Seydlitz – Geographie Gymnasium Hessen, Schülerband für die Klassen 5 und 6, Ausgabe 2024 mit Arbeitsheft"
		gross = "DEUTSCHBUCH GYMNASIUM – ALLGEMEINE AUSGABE 2019, 5. SCHULJAHR, SCHÜLERBUCH MIT ARBEITSHEFT"
	)
	bestellungen := []repository.BerichtBestellung{{
		LieferantName: "Cornelsen", Kundennummer: "C-1", Bestelldatum: time.Date(2026, 3, 14, 0, 0, 0, 0, time.UTC),
		Gesamtbetrag: 81, AnzahlExemplare: 3,
		Positionen: []repository.BerichtPosition{
			{TitelName: lang, ISBN: "ISBN-PROBE-1", Menge: 1, Einzelpreis: 27},
			{TitelName: gross, ISBN: "ISBN-PROBE-2", Menge: 1, Einzelpreis: 27},
			{TitelName: "Atlas", ISBN: "ISBN-PROBE-3", Menge: 1, Einzelpreis: 27},
		},
	}}
	for _, mitPreisen := range []bool{true, false} {
		roh, err := generateBestellBerichtPDF(bestellungen, pdf.SchuleInfo{Name: "Testbibliothek"}, bestellBerichtOpts{
			Titel: "Testbericht", Von: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), Bis: time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC),
			Jahresansicht: false, MitPreisen: mitPreisen,
		})
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
