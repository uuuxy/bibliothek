package isbnutil

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

// Zwei Rechnungen in die andere Länge, eine Wahrheit: Die Bestelltür (api/isbn_handler.go)
// rechnet mit AndereForm, die Katalogsuche im Browser mit isbnFormen
// (frontend/src/lib/utils/isbnFormen.js). Rechnen beide verschieden, fragt die Bestelltür
// nach einem anderen Titel, als die Suche findet. Dieselbe Bauart wie bei Code 39
// (pkg/code39/zwilling_test.go): Beide Seiten lesen dieselben Fälle — Beispielpaare der
// ISBN-Norm und das Paar vom Testserver, keine ausgedachten Nummern: Eine erfundene ISBN
// hielte den Test grün, ohne dass die Rechnung stimmt.
func TestAndereForm_GoUndJavaScriptTeilenDiePrueffaelle(t *testing.T) {
	const faelleDatei = "../../frontend/src/lib/utils/isbnFormen.faelle.json"
	roh, err := os.ReadFile(faelleDatei)
	if err != nil {
		t.Fatalf("Prüffälle lesen: %v", err)
	}
	var pruefung struct {
		Faelle []struct {
			Fall   string  `json:"fall"`
			Roh    string  `json:"roh"`
			Andere *string `json:"andere"`
		} `json:"faelle"`
	}
	if err := json.Unmarshal(roh, &pruefung); err != nil {
		t.Fatalf("Prüffälle: %v", err)
	}

	mit, ohne := 0, 0
	for _, f := range pruefung.Faelle {
		want := ""
		if f.Andere != nil {
			want = *f.Andere
			mit++
		} else {
			ohne++
		}
		if got := AndereForm(f.Roh); got != want {
			t.Errorf("%s: AndereForm(%q) = %q, erwartet %q", f.Fall, f.Roh, got, want)
		}
	}
	if mit < 8 || ohne < 4 {
		t.Errorf("%d Fälle mit anderer Form, %d ohne — erwartet mindestens 8 und 4: liest der Test "+
			"noch auf isbnFormen.faelle.json?", mit, ohne)
	}

	// Liest die JavaScript-Seite dieselbe Datei? Sonst prüfte jede Seite nur sich selbst.
	vitest, err := os.ReadFile("../../frontend/src/lib/utils/isbnFormen.test.js")
	if err != nil {
		t.Fatalf("Vitest lesen: %v", err)
	}
	if !strings.Contains(string(vitest), "./isbnFormen.faelle.json") {
		t.Error("isbnFormen.test.js liest isbnFormen.faelle.json nicht mehr ein")
	}
}

// Was keine ISBN ist, hat keine andere Form — auch mit Zeichen, die der Browser für die Suche
// streicht (siehe _grenze in der Falldatei): Der Server hält die Normalform der Datenbank.
func TestAndereForm_KeineISBN(t *testing.T) {
	for _, roh := range []string{"mathematik", "B97601826457", "ISBN 978-3-16-148410-0"} {
		if got := AndereForm(roh); got != "" {
			t.Errorf("AndereForm(%q) = %q, erwartet keine", roh, got)
		}
	}
}
