package bestellung

import (
	"strings"
	"testing"
	"time"

	"bibliothek/internal/pdftest"
	"bibliothek/pdf"
	"bibliothek/pkg/mitteltopf"
	"bibliothek/repository"
)

// Der Bestellbericht trennt die beiden Töpfe.
//
// Die Schule beschafft aus zwei getrennten Haushalten: Lernmittel aus Landesmitteln, den
// Bestand der Schülerbücherei aus Mitteln des Schulträgers. Für das Schulamt zählt der Anteil
// des Landes, für den Schulträger seiner; aus einer gemischten Liste mit einer Summe müsste
// jemand die zwei Zahlen von Hand ziehen.
//
// Geprüft am Blatt, das aus den Zeilen der Abfrage entsteht: Die Tür ordnet die Bestellungen
// den Töpfen zu (bestellberichtToepfe), der Erzeuger druckt die Blöcke.

// berichtBlatt druckt den Bericht, wie die Tür ihn aus den Zeilen der Abfrage füllt.
func berichtBlatt(t *testing.T, titel string, orders []repository.BerichtBestellung) string {
	t.Helper()
	toepfe, err := bestellberichtToepfe(orders)
	if err != nil {
		t.Fatalf("Blöcke füllen: %v", err)
	}
	roh, err := pdf.GenerateBestellberichtPDF(pdf.Bestellbericht{
		Titel:         titel,
		Von:           time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC),
		Bis:           time.Date(2026, 9, 30, 0, 0, 0, 0, time.UTC),
		Jahresansicht: false,
		MitPreisen:    true,
		Toepfe:        toepfe,
	}, pdf.SchuleInfo{Name: "Testschule"})
	if err != nil {
		t.Fatalf("Bericht erzeugen: %v", err)
	}
	return pdftest.Suchtext(t, roh)
}

func berichtBestellung(mittel string, betrag float64, exemplare int, titel string) repository.BerichtBestellung {
	return repository.BerichtBestellung{
		ID:              "id-" + titel,
		LieferantName:   "Testhändler",
		Kundennummer:    "K-1",
		Bestelldatum:    time.Date(2026, 9, 3, 10, 0, 0, 0, time.UTC),
		Gesamtbetrag:    betrag,
		AnzahlExemplare: exemplare,
		Mittel:          mittel,
		Positionen:      []repository.BerichtPosition{{TitelName: titel, ISBN: "978", Menge: exemplare, Einzelpreis: betrag / float64(exemplare)}},
	}
}

func TestBestellberichtTrenntDieToepfeUndIhreSummen(t *testing.T) {
	orders := []repository.BerichtBestellung{
		berichtBestellung(mitteltopf.Land, 120.00, 30, "Mathematik 7"),
		berichtBestellung(mitteltopf.Land, 80.00, 20, "Englisch 8"),
		berichtBestellung(mitteltopf.Schultraeger, 45.50, 5, "Die unendliche Geschichte"),
		berichtBestellung("", 10.00, 2, "Alt ohne Zuordnung"),
	}
	text := berichtBlatt(t, "Bestellbericht", orders)

	// Jeder Topf steht mit seiner eigenen Summe da, und die Bestellungen ohne Zuordnung als
	// eigener Block: Sie sind keinem der zwei Töpfe zugeschlagen.
	faelle := []struct{ topf, summe string }{
		{"Lernmittelfreiheit", "200,00"},
		{"lerb", "45,50"}, // „Schülerbücherei" — Umlaut, deshalb das Fragment
		{"ohne Zuordnung", "10,00"},
	}
	for _, f := range faelle {
		if !strings.Contains(text, f.topf) {
			t.Errorf("der Bericht nennt den Topf %q nicht", f.topf)
		}
		if !strings.Contains(text, f.summe) {
			t.Errorf("die Summe %s des Topfs %q fehlt im Bericht", f.summe, f.topf)
		}
	}

	// Die Gesamtsumme steht darunter: 200 + 45,50 + 10 = 255,50. Geht sie nicht auf, ist einer
	// der Blöcke doppelt gezählt oder einer fehlt.
	if !strings.Contains(text, "255,50") {
		t.Error("die Gesamtsumme 255,50 fehlt — die Summen der Töpfe müssen sie ergeben")
	}
}

// Ein Bericht über einen einzigen Topf (Lieferantenabrechnung mit Filter) nennt den anderen
// nicht: Das Blatt wird gegen eine Rechnung dieses Topfs gehalten.
func TestBestellberichtUeberEinenTopfNenntDenAnderenNicht(t *testing.T) {
	orders := []repository.BerichtBestellung{berichtBestellung(mitteltopf.Land, 120.00, 30, "Mathematik 7")}
	text := berichtBlatt(t, "Lieferantenabrechnung", orders)

	if !strings.Contains(text, "Lernmittelfreiheit") {
		t.Error("der Topf des Berichts fehlt")
	}
	if strings.Contains(text, "lerb") || strings.Contains(text, "ohne Zuordnung") {
		t.Error("der Bericht nennt einen Topf, zu dem er keine Bestellung enthält")
	}
}
