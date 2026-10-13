package repository

import "testing"

// Die eine Frage beider Importe an den Titelbestand: zuerst die ISBN, in welcher Schreibweise
// auch immer, sonst der Titeltext. Der Titeltext zählt auch für eine Zeile mit ISBN, weil der
// Bestand denselben Titel ohne ISBN führen kann.
func TestTitelBestand_TitelZu(t *testing.T) {
	b := NeuerTitelBestand()
	b.Merke("9783060619000", "Deutschbuch 9", "id-deutsch")
	b.Merke("", "Faust", "id-faust")
	b.Merke("3-12-733771-X", "Lambacher Schweizer 7", "id-mathe") // zehnstellig mit Bindestrichen

	for _, f := range []struct {
		name, isbn, titel string
		id                string
		bekannt           bool
	}{
		{"über die ISBN, anderer Titeltext", "9783060619000", "Ein anderer Titel", "id-deutsch", true},
		{"über die ISBN mit Bindestrichen", "978-3-06-061900-0", "Ein anderer Titel", "id-deutsch", true},
		{"zehnstellig gemerkt, dreizehnstellig gefragt", "9783127337716", "Ein anderer Titel", "id-mathe", true},
		{"zehnstellig gemerkt, ohne Bindestriche und klein gefragt", "312733771x", "Ein anderer Titel", "id-mathe", true},
		{"ohne ISBN über den Titeltext", "", "Faust", "id-faust", true},
		{"unbekannte ISBN, bekannter Titeltext", "9780000000002", "Faust", "id-faust", true},
		{"Titeltext mit Anführungszeichen und doppeltem Leerraum", "", `"Faust"`, "id-faust", true},
		{"die ISBN geht dem Titeltext vor", "9783060619000", "Faust", "id-deutsch", true},
		{"weder ISBN noch Titeltext", "9780000000002", "Unbekannt", "", false},
		{"alles leer", "", "", "", false},
	} {
		id, bekannt := b.TitelZu(f.isbn, f.titel)
		if id != f.id || bekannt != f.bekannt {
			t.Errorf("%s: TitelZu(%q, %q) = %q, %v; erwartet %q, %v", f.name, f.isbn, f.titel, id, bekannt, f.id, f.bekannt)
		}
	}
}

// Ein nachgetragener Titel ersetzt, was unter derselben ISBN oder demselben Titeltext stand:
// Der Lauf hat ihn eben angelegt, und die folgenden Zeilen der Datei meinen ihn.
func TestTitelBestand_MerkeErsetzt(t *testing.T) {
	b := NeuerTitelBestand()
	b.Merke("9783060619000", "Deutschbuch 9", "id-alt")
	b.Merke("978-3-06-061900-0", "Deutschbuch 9", "id-neu")
	if id, _ := b.TitelZu("9783060619000", ""); id != "id-neu" {
		t.Errorf("über die ISBN: %q, erwartet den nachgetragenen Titel", id)
	}
	if id, _ := b.TitelZu("", "Deutschbuch 9"); id != "id-neu" {
		t.Errorf("über den Titeltext: %q, erwartet den nachgetragenen Titel", id)
	}
	if len(b.nachISBN) != 1 {
		t.Errorf("zwei Schreibweisen derselben ISBN stehen als %d Schlüssel im Bestand", len(b.nachISBN))
	}
}
