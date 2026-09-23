package api

import (
	"os"
	"reflect"
	"regexp"
	"strings"
	"testing"
)

// Das dritte Glied der Auskunfts-Kette (23.09.2026).
//
// Die Kette ist: Spalte in `schueler` → Feld in DsgvoStammdaten → Zeile auf dem Blatt,
// das die betroffene Person bekommt. Das erste Glied hält
// TestDsgvoAuskunft_KenntJedeSchuelerSpalte, das zweite der Compiler (die Abfrage wird in
// den Struct gescannt). Das dritte hielt bis heute niemand: Der PDF-Abschnitt zählt seine
// Zeilen von Hand auf, und wer ein Feld ergänzt, ergänzt die Zeile leicht nicht.
//
// Gemessen am 23.09.2026, bevor dieses Gate stand: `art` und `hat_zugangskonto` fehlten
// seit Migration 123 auf dem Blatt. Die abgerufene Auskunft (JSON) war damit länger als
// die gedruckte — zwei Auskünfte auf dieselbe Frage.
//
// Das Gate liest den Quelltext von dsgvo_pdf.go, weil es genau das prüfen soll, was dort
// von Hand steht. Es nennt das Feld, nicht die Zeile: WIE ein Wert gedruckt wird, ist
// Sache des Abschnitts; DASS er gedruckt wird, ist die Zusicherung.
func TestDsgvoPDF_DrucktJedesStammdatenfeld(t *testing.T) {
	// Ausnahmen: Feld → Begründung. Heute leer — jedes Feld der Auskunft ist eine
	// gespeicherte Angabe über die Person und gehört auf ihr Blatt.
	ausnahmen := map[string]string{}

	quelle, err := os.ReadFile("dsgvo_pdf.go")
	if err != nil {
		t.Fatalf("dsgvo_pdf.go lesen: %v", err)
	}
	// Nicht-leer-Garantie: Liest die Datei sich künftig anders (umbenannt, aufgeteilt),
	// soll das Gate auffallen und nicht stumm alles durchlassen.
	if !strings.Contains(string(quelle), "dsgvoStammdatenAbschnitt") {
		t.Fatal("Liveness: dsgvo_pdf.go enthält keinen Stammdaten-Abschnitt mehr — Gate zeigt ins Leere")
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
				"(api/dsgvo_pdf.go) — drucken oder begründet ausnehmen", name)
		}
	}
	for name := range ausnahmen {
		if _, gefunden := felder.FieldByName(name); !gefunden {
			t.Errorf("Ausnahme %q nennt kein Feld von DsgvoStammdaten mehr — Eintrag entfernen", name)
		}
	}
}
