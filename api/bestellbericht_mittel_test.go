package api

import (
	"strings"
	"testing"
	"time"

	"bibliothek/pdf"
	"bibliothek/repository"
)

// Der Bestellbericht trennt die beiden Töpfe (#596, Bauplan 7.3 Schritt 4).
//
// Die Schule beschafft aus zwei getrennten Haushalten: Lernmittel aus Landesmitteln, den
// Bestand der Schülerbücherei aus Mitteln des Schulträgers. Der Bericht war bis zum
// 12.09.2026 EINE Liste mit EINER Summe — genau die Zahl, die niemand verwenden kann:
// Für das Schulamt zählt der Landes-Anteil, für den Schulträger seiner, und wer die
// Summen von Hand aus einer gemischten Liste zieht, rechnet jeden Monat neu.
//
// Geprüft am entpackten PDF-Inhaltsstrom, nicht an einer Zwischenstruktur: Der Bericht
// ist ein Blatt Papier, gegen das eine Rechnung geprüft wird.

func berichtBestellung(mittel string, betrag float64, exemplare int, titel string) berichtOrder {
	return berichtOrder{
		ID:              "id-" + titel,
		LieferantName:   "Testhändler",
		Kundennummer:    "K-1",
		Bestelldatum:    time.Date(2026, 9, 3, 10, 0, 0, 0, time.UTC),
		Gesamtbetrag:    betrag,
		AnzahlExemplare: exemplare,
		Mittel:          mittel,
		Positionen:      []berichtPosition{{TitelName: titel, ISBN: "978", Menge: exemplare, Einzelpreis: betrag / float64(exemplare)}},
	}
}

func TestBestellberichtTrenntDieToepfeUndIhreSummen(t *testing.T) {
	orders := []berichtOrder{
		berichtBestellung(repository.MittelLand, 120.00, 30, "Mathematik 7"),
		berichtBestellung(repository.MittelLand, 80.00, 20, "Englisch 8"),
		berichtBestellung(repository.MittelSchultraeger, 45.50, 5, "Die unendliche Geschichte"),
		berichtBestellung("", 10.00, 2, "Alt ohne Zuordnung"),
	}

	roh, err := generateBestellBerichtPDF(orders, pdf.SchuleInfo{Name: "Testschule"}, bestellBerichtOpts{
		Titel:         "Bestellbericht",
		Von:           time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC),
		Bis:           time.Date(2026, 9, 30, 0, 0, 0, 0, time.UTC),
		Jahresansicht: false,
		MitPreisen:    true,
	})
	if err != nil {
		t.Fatalf("Bericht erzeugen: %v", err)
	}
	text := pdfText(t, roh)

	// Jeder Topf steht mit seiner eigenen Summe da — und die Alt-Bestellungen ohne
	// Zuordnung als das, was sie sind: nicht dem einen oder anderen zugeschlagen.
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

	// Und die Gesamtsumme steht darunter: 200 + 45,50 + 10 = 255,50. Geht sie nicht auf,
	// ist einer der Blöcke doppelt gezählt oder einer fehlt — das Gate aus #596
	// („Land + Kreis + ohne Zuordnung = Gesamt").
	if !strings.Contains(text, "255,50") {
		t.Error("die Gesamtsumme 255,50 fehlt — die Summen der Töpfe müssen sie ergeben")
	}
}

// Ein Bericht über einen einzigen Topf (Lieferantenabrechnung mit Filter) nennt den
// anderen NICHT: Das Blatt wird gegen eine Rechnung dieses Topfs gehalten.
func TestBestellberichtUeberEinenTopfNenntDenAnderenNicht(t *testing.T) {
	orders := []berichtOrder{berichtBestellung(repository.MittelLand, 120.00, 30, "Mathematik 7")}

	roh, err := generateBestellBerichtPDF(orders, pdf.SchuleInfo{Name: "Testschule"}, bestellBerichtOpts{
		Titel:         "Lieferantenabrechnung",
		Von:           time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC),
		Bis:           time.Date(2026, 9, 30, 0, 0, 0, 0, time.UTC),
		Jahresansicht: false,
		MitPreisen:    true,
	})
	if err != nil {
		t.Fatalf("Bericht erzeugen: %v", err)
	}
	text := pdfText(t, roh)

	if !strings.Contains(text, "Lernmittelfreiheit") {
		t.Error("der Topf des Berichts fehlt")
	}
	if strings.Contains(text, "lerb") || strings.Contains(text, "ohne Zuordnung") {
		t.Error("der Bericht nennt einen Topf, zu dem er keine Bestellung enthält")
	}
}
