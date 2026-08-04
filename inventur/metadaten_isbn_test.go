package inventur

import "testing"

// MARC21 020 $a ist ein Textfeld, kein ISBN-Feld: Neben der Nummer stehen dort Einband,
// Preis und Bemerkungen. Diese Vorlagen sind die Formen, die in DNB-Antworten zu Titeln
// dieser Schule tatsächlich vorkommen.
func TestBereinigeISBN(t *testing.T) {
	faelle := []struct {
		name    string
		eingabe string
		erwarte string
	}{
		{"blanke ISBN-13", "9783124912008", "9783124912008"},
		{"ISBN-10", "3124912004", "3124912004"},
		{"mit Bindestrichen", "978-3-12-491200-8", "9783124912008"},
		{"mit Einband und Preis dahinter", "978-3-12-491200-8 kart. : EUR 24.90", "9783124912008"},
		{"Pruefziffer X bleibt erhalten", "3-486-2072X-1", "34862072X1"},
		{"kleines x wird uebernommen", "3-486-2072x-1", "34862072x1"},

		// Alles, was nicht auf 10 oder 13 Stellen kommt, ist keine ISBN. Eine halb
		// erkannte Nummer wäre schlimmer als gar keine: Sie zeigt im Katalog auf ein
		// fremdes Buch, statt das Feld sichtbar leer zu lassen.
		{"zu kurz", "3-12-4912", ""},
		{"zu lang", "97831249120081", ""},
		{"leer", "", ""},
		{"nur Leerraum", "   ", ""},
		{"reiner Text ohne Nummer", "kart.", ""},
	}

	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			if got := bereinigeISBN(f.eingabe); got != f.erwarte {
				t.Errorf("bereinigeISBN(%q) = %q, erwartet %q", f.eingabe, got, f.erwarte)
			}
		})
	}
}

// Der letzte GÜLTIGE Wert gewinnt — ein nachfolgendes unbrauchbares $a darf eine bereits
// erkannte Nummer nicht wieder löschen.
func TestVerarbeiteISBNBehaeltGueltigeNummerBeiNachfolgendemSchrott(t *testing.T) {
	var b marcBibDaten
	b.verarbeiteISBN([]marcSubfield{
		{Code: "a", Value: "9783124912008"},
		{Code: "a", Value: "kart."},
	})

	if b.isbn != "9783124912008" {
		t.Errorf("isbn = %q, erwartet 9783124912008", b.isbn)
	}
}
