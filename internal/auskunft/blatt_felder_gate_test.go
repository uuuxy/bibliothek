package auskunft

import (
	"os"
	"reflect"
	"regexp"
	"strings"
	"testing"
)

// Das dritte Glied der Kette von der Spalte in leser über das Feld in DsgvoStammdaten zur Zeile
// auf dem Blatt. Das erste hält api.TestDsgvoAuskunft_KenntJedeLeserSpalte, das zweite der
// Compiler. Der Abschnitt der Stammdaten zählt seine Zeilen von Hand auf: Ohne dieses Gate
// fehlt ein neues Feld auf dem Blatt, und die gedruckte Auskunft ist kürzer als die abgerufene.
//
// Das Gate liest den Quelltext von blatt.go, weil es prüft, was dort von Hand steht. Es
// verlangt, dass ein Feld gedruckt wird; wie, ist Sache des Abschnitts.
func TestDsgvoPDF_DrucktJedesStammdatenfeld(t *testing.T) {
	// Ausnahmen: Feld → Begründung. Leer, weil jedes Feld eine gespeicherte Angabe über die
	// Person ist und auf ihr Blatt gehört.
	ausnahmen := map[string]string{}

	quelle, err := os.ReadFile("blatt.go")
	if err != nil {
		t.Fatalf("blatt.go lesen: %v", err)
	}
	// Nicht-leer-Garantie: Liest die Datei sich künftig anders (umbenannt, aufgeteilt),
	// soll das Gate auffallen und nicht stumm alles durchlassen.
	if !strings.Contains(string(quelle), "dsgvoStammdatenAbschnitt") {
		t.Fatal("Liveness: blatt.go enthält keinen Stammdaten-Abschnitt mehr — Gate zeigt ins Leere")
	}

	felder := reflect.TypeOf(DsgvoStammdaten{})
	if felder.NumField() < 20 {
		t.Fatalf("Liveness: nur %d Felder in DsgvoStammdaten", felder.NumField())
	}
	for i := 0; i < felder.NumField(); i++ {
		name := felder.Field(i).Name
		if _, ok := ausnahmen[name]; ok {
			continue
		}
		if !regexp.MustCompile(`\bst\.` + regexp.QuoteMeta(name) + `\b`).Match(quelle) {
			t.Errorf("DsgvoStammdaten.%s steht in der abgerufenen Auskunft, aber auf keiner Zeile des PDF "+
				"(internal/auskunft/blatt.go) — drucken oder begründet ausnehmen", name)
		}
	}
	for name := range ausnahmen {
		if _, gefunden := felder.FieldByName(name); !gefunden {
			t.Errorf("Ausnahme %q nennt kein Feld von DsgvoStammdaten mehr — Eintrag entfernen", name)
		}
	}
}
