package isbnutil

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

// Die Fälle sind dieselben wie in repository/isbn_normalform_pg_test.go — dort läuft die
// SQL-Seite und die Parität, hier die reine Go-Regel.
func TestNormalform(t *testing.T) {
	faelle := []struct{ in, want string }{
		{"978-3-16-148410-0", "9783161484100"},
		{" 978 3 16 148410 0 ", "9783161484100"},
		{"9783161484100", "9783161484100"},
		// Zehnstellig mit richtigem Prüfzeichen: die Beispielpaare der ISBN-Norm.
		{"3-16-148410-x", "9783161484100"},
		{"0-306-40615-2", "9780306406157"},
		{"080442957X", "9780804429573"},
		// Zehnstellig mit falschem Prüfzeichen bleibt zehnstellig. 3499500252 steht am
		// Testserver; gerechnet führte sie auf 9783499500251, die ISBN eines anderen Buchs.
		{"3499500252", "3499500252"},
		{"3-16-148410-0", "3161484100"},
		{"979-12-345-6789-6", "9791234567896"},
		{"ISBN-0000000001", "ISBN-0000000001"},         // keine ISBN: bleibt
		{"3-12-345678-9 kart.", "3-12-345678-9 kart."}, // Beiwerk: bleibt
		{" - ", ""},
		{"", ""},
	}
	for _, f := range faelle {
		if got := Normalform(f.in); got != f.want {
			t.Errorf("Normalform(%q) = %q, erwartet %q", f.in, got, f.want)
		}
	}
}

// Die Normalform ist ein Fixpunkt: Was die Datenbank gespeichert hat, ändert ein zweiter
// Durchgang nicht mehr. Sonst schriebe jedes Speichern eines unveränderten Titels die ISBN um.
func TestNormalform_ZweimalIstEinmal(t *testing.T) {
	for _, roh := range []string{
		"3-16-148410-x", "0306406152", "3499500252", "978-3-16-148410-0", "9791234567896",
		"ISBN-0000000001", "3-12-345678-9 kart.", " - ",
	} {
		einmal := Normalform(roh)
		if zweimal := Normalform(einmal); zweimal != einmal {
			t.Errorf("Normalform(%q) = %q, noch einmal %q", roh, einmal, zweimal)
		}
	}
}

// Zwei Rechnungen, eine Wahrheit: Die Datenbank führt eine ISBN in der Normalform, die
// Katalogsuche im Browser sucht mit isbnFormen (frontend/src/lib/utils/isbnFormen.js) nach
// beiden Längen. Rechnen beide die dreizehnstellige Form verschieden, findet die Suche einen
// Titel nicht, den die Datenbank unter dieser ISBN führt. Dieselbe Bauart wie bei Code 39
// (pkg/code39/zwilling_test.go): Beide Seiten lesen dieselben Fälle — Beispielpaare der
// ISBN-Norm und das Paar vom Testserver, keine ausgedachten Nummern: Eine erfundene ISBN
// hielte den Test grün, ohne dass die Rechnung stimmt.
func TestNormalform_GoUndJavaScriptTeilenDiePrueffaelle(t *testing.T) {
	const faelleDatei = "../../frontend/src/lib/utils/isbnFormen.faelle.json"
	roh, err := os.ReadFile(faelleDatei)
	if err != nil {
		t.Fatalf("Prüffälle lesen: %v", err)
	}
	var pruefung struct {
		Faelle []struct {
			Fall       string  `json:"fall"`
			Roh        string  `json:"roh"`
			Normalform *string `json:"normalform"`
		} `json:"faelle"`
	}
	if err := json.Unmarshal(roh, &pruefung); err != nil {
		t.Fatalf("Prüffälle: %v", err)
	}

	umgerechnet, zehnGeblieben := 0, 0
	for _, f := range pruefung.Faelle {
		if f.Normalform == nil {
			t.Errorf("%s: der Fall nennt keine normalform", f.Fall)
			continue
		}
		got := Normalform(f.Roh)
		if got != *f.Normalform {
			t.Errorf("%s: Normalform(%q) = %q, erwartet %q", f.Fall, f.Roh, got, *f.Normalform)
		}
		switch zehnstellig := len(CleanISBN(f.Roh)) == 10; {
		case zehnstellig && len(got) == 13:
			umgerechnet++
		case zehnstellig && len(got) == 10:
			zehnGeblieben++
		}
	}
	if umgerechnet < 3 || zehnGeblieben < 1 {
		t.Errorf("%d zehnstellige Fälle umgerechnet, %d zehnstellig geblieben — erwartet mindestens 3 und 1: "+
			"liest der Test noch auf isbnFormen.faelle.json?", umgerechnet, zehnGeblieben)
	}

	// Liest die JavaScript-Seite dieselbe Datei? Sonst prüfte jede Seite nur sich selbst.
	vitest, err := os.ReadFile("../../frontend/src/lib/utils/isbnFormen.test.js")
	if err != nil {
		t.Fatalf("Vitest lesen: %v", err)
	}
	if !strings.Contains(string(vitest), "./isbnFormen.faelle.json") || !strings.Contains(string(vitest), "f.normalform") {
		t.Error("isbnFormen.test.js liest die normalform aus isbnFormen.faelle.json nicht mehr")
	}
}
