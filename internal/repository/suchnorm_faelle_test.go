package repository

import (
	"encoding/json"
	"os"
	"testing"
)

// Die Normalform rechnet an drei Stellen: in der Datenbank, in Go für den Namensschlüssel des
// LUSD-Imports und im Browser (frontend/src/lib/utils/suchnorm.js). Die Fälle sind mit der
// SQL-Funktion erzeugt, der Browser liest dieselben; rechnete Go anders, meinten Import und Suche
// verschiedene Menschen.
func TestSuchnorm_GemeinsameFaelle(t *testing.T) {
	roh, err := os.ReadFile("../../frontend/src/lib/utils/suchnorm.faelle.json")
	if err != nil {
		t.Fatal(err)
	}
	var pruefung struct {
		Zeichen [][2]string `json:"zeichen"`
		Namen   [][2]string `json:"namen"`
	}
	if err := json.Unmarshal(roh, &pruefung); err != nil {
		t.Fatal(err)
	}
	if len(pruefung.Zeichen) < 600 || len(pruefung.Namen) < 10 {
		t.Fatalf("%d Zeichen und %d Namen, erwartet mindestens 600 und 10: liest der Test noch suchnorm.faelle.json?",
			len(pruefung.Zeichen), len(pruefung.Namen))
	}
	for _, fall := range append(pruefung.Zeichen, pruefung.Namen...) {
		if ist := suchnorm(fall[0]); ist != fall[1] {
			t.Errorf("suchnorm(%q) = %q, erwartet %q", fall[0], ist, fall[1])
		}
	}
}
