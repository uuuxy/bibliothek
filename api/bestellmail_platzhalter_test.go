package api

import "testing"

// Kundennummer, Zahl der Titel und Zahl der Exemplare stehen an ihrem Platzhalter. Die beiden
// Zahlen haben denselben Typ; vertauscht nennt die Mail dem Händler eine falsche Menge.
func TestBestellmail_KundennummerUndZahlenStehenAnIhremPlatzhalter(t *testing.T) {
	_, body := resolveBestellMail("Betreff",
		"Kunde {{.Kundennummer}}, Titel {{.AnzahlTitel}}, Exemplare {{.AnzahlExemplare}}",
		bestellMailWerte{kundennummer: "K-77", anzahlTitel: 2, anzahlExemplare: 5, link: "", gueltigBis: nil, mittel: ""})
	if will := "Kunde K-77, Titel 2, Exemplare 5"; body != will {
		t.Errorf("Text %q, erwartet %q", body, will)
	}
}
