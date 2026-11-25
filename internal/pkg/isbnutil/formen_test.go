package isbnutil

import (
	"encoding/json"
	"os"
	"slices"
	"testing"
)

// Formen und isbnFormen im Browser liefern dieselben Schreibweisen: Die Titelsuche am Server
// und die im Medienkatalog finden einen Bestand über beide Längen derselben ISBN.
func TestFormen_GoUndJavaScriptTeilenDiePrueffaelle(t *testing.T) {
	roh, err := os.ReadFile("../../../frontend/src/lib/utils/isbnFormen.faelle.json")
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
	paare := 0
	for _, f := range pruefung.Faelle {
		formen := Formen(f.Roh)
		if f.Andere == nil {
			if len(formen) > 1 {
				t.Errorf("%s: Formen(%q) = %v, erwartet höchstens eine Form", f.Fall, f.Roh, formen)
			}
			continue
		}
		paare++
		if len(formen) != 2 || !slices.Contains(formen, *f.Andere) {
			t.Errorf("%s: Formen(%q) = %v, erwartet zwei Formen mit %q", f.Fall, f.Roh, formen, *f.Andere)
		}
	}
	if paare < 6 {
		t.Errorf("nur %d Fälle mit zweiter Form, erwartet mindestens 6", paare)
	}
	for roh, erwartet := range map[string][]string{
		"9791234567896": {"9791234567896"},
		"mathematik":    nil,
		"7":             nil,
		"B12345678901":  nil,
		"3-16-148410-x": {"316148410X", "9783161484100"},
	} {
		if got := Formen(roh); !slices.Equal(got, erwartet) {
			t.Errorf("Formen(%q) = %v, erwartet %v", roh, got, erwartet)
		}
	}
}
