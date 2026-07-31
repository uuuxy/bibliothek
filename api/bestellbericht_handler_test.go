package api

import (
	"bytes"
	"testing"
	"time"

	"bibliothek/pdf"
)

func berichtTestdaten() []berichtOrder {
	return []berichtOrder{
		{
			LieferantName:   "Cornelsen",
			Kundennummer:    "C-88123",
			Bestelldatum:    time.Date(2026, 3, 14, 0, 0, 0, 0, time.UTC),
			Gesamtbetrag:    54.00,
			AnzahlExemplare: 2,
			Positionen: []berichtPosition{
				{TitelName: "Alexander Gesamtausgabe", ISBN: "9783124912008", Menge: 2, Einzelpreis: 27.00},
			},
		},
		{
			LieferantName:   "Klett",
			Kundennummer:    "K-5000",
			Bestelldatum:    time.Date(2026, 7, 2, 0, 0, 0, 0, time.UTC),
			Gesamtbetrag:    0,
			AnzahlExemplare: 4,
			Positionen: []berichtPosition{
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
			daten, err := generateBestellBerichtPDF(berichtTestdaten(), schule, "Testbericht", von, bis, f.jahresansicht, f.mitPreisen)
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
		daten, err := generateBestellBerichtPDF(nil, pdf.SchuleInfo{Name: "Testbibliothek"}, "Leer", von, bis, true, mitPreisen)
		if err != nil {
			t.Fatalf("mitPreisen=%v: %v", mitPreisen, err)
		}
		if !bytes.HasPrefix(daten, []byte("%PDF-")) {
			t.Errorf("mitPreisen=%v: Ergebnis ist kein PDF", mitPreisen)
		}
	}
}
