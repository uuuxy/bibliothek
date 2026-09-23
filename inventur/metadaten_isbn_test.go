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

// Ein nachfolgendes unbrauchbares $a darf eine bereits erkannte Nummer nicht wieder löschen.
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

// Die ISBN-13 gewinnt vor der ISBN-10, gleich in welcher Reihenfolge; innerhalb einer Länge
// der erste gültige Wert. Die DNB führt beide Formen desselben Buchs, die ISBN-10 zuletzt —
// bis zum 23.09.2026 gewann sie deshalb, und ein über die Freitextsuche bestellter Titel
// entstand zehnstellig (docs/OFFEN.md 5.5).
func TestVerarbeiteISBNNimmtDieISBN13(t *testing.T) {
	faelle := []struct {
		name    string
		felder  [][]string // je 020 die Werte seiner $a
		erwarte string
	}{
		{"erst 13, dann 10 (wie die DNB)", [][]string{{"9783751200530"}, {"3751200533"}}, "9783751200530"},
		{"erst 10, dann 13", [][]string{{"3751200533"}, {"9783751200530"}}, "9783751200530"},
		{"nur 10", [][]string{{"3751200533"}}, "3751200533"},
		{"zwei 13: die erste", [][]string{{"9783751200530"}, {"9783551652713"}}, "9783751200530"},
	}
	for _, f := range faelle {
		var b marcBibDaten
		for _, werte := range f.felder {
			var felder []marcSubfield
			for _, w := range werte {
				felder = append(felder, marcSubfield{Code: "a", Value: w})
			}
			b.verarbeiteISBN(felder)
		}
		if b.isbn != f.erwarte {
			t.Errorf("%s: isbn = %q, erwartet %q", f.name, b.isbn, f.erwarte)
		}
	}
}
