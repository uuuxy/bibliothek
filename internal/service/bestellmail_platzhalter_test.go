package service

import "testing"

// Kundennummer, Zahl der Titel und Zahl der Exemplare stehen an ihrem Platzhalter. Die beiden
// Zahlen haben denselben Typ; vertauscht nennt die Mail dem Händler eine falsche Menge.
func TestBestellmail_KundennummerUndZahlenStehenAnIhremPlatzhalter(t *testing.T) {
	_, body := LoeseBestellMailAuf("Betreff",
		"Kunde {{.Kundennummer}}, Titel {{.AnzahlTitel}}, Exemplare {{.AnzahlExemplare}}",
		BestellMailWerte{Kundennummer: "K-77", AnzahlTitel: 2, AnzahlExemplare: 5, Link: "", GueltigBis: nil, Mittel: ""})
	if will := "Kunde K-77, Titel 2, Exemplare 5"; body != will {
		t.Errorf("Text %q, erwartet %q", body, will)
	}
}
