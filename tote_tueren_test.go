package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
)

// Gate gegen tote Interface-Türen.
//
// Anlass (16.08.2026): UpsertBookTitle stand fünf Wochen ohne produktiven Aufrufer
// im BookRepository-Interface — nur Implementierung und ein eigener Test hielten es
// am Leben. In der Zeit driftete sein Blanking-Schutz vom echten Bulk-Pfad weg.
// `deadcode` (x/tools) KANN diese Klasse nicht finden: Sobald ein Typ in ein
// Interface gewandelt wird, gilt seine komplette Methodentabelle als erreichbar
// (an genau diesem Fall per Gegenprobe belegt). Deshalb dieser eigene Detektor.
//
// Regel: Jede Methode, die eines UNSERER Interfaces deklariert, muss außerhalb
// von _test.go-Dateien mindestens einmal als Selektor benutzt werden. Der Abgleich
// läuft über den Namen (konservativ: ein Aufruf gleichen Namens irgendwo genügt) —
// das erzeugt keine Fehlalarme durch dynamische Aufrufe, übersieht dafür Türen,
// deren Name anderswo lebt. Als Ratsche reicht das: Neue Türen ohne jeden Aufrufer
// werden rot.
//
// Reparatur bei Rot: Entweder die Tür samt Implementierung und Test zurückbauen
// (Regelfall — Parallel-Türen driften), oder sie hier mit Begründung eintragen.
var bewussteTueren = map[string]string{
	// KEIN toter Code, sondern eine offene Feature-Lücke (16.08.2026): Diese Methode
	// ist der EINZIGE Schreiber von schadensfaelle.storniert_am im ganzen System —
	// sie setzt den Storno UND schreibt das Audit-Protokoll. Gelesen wird die Spalte
	// überall (offene-Gebühren-Filter, DSGVO-PDF, LUSD-Abgleich), aber es gibt noch
	// keinen Endpunkt und keine Oberfläche, die stornieren kann. Ohne diese Tür
	// blockiert eine zu erlassende Gebühr Schülerlöschung und LUSD-Abgleich dauerhaft
	// (der einzige Ausweg wäre, sie wahrheitswidrig auf "bezahlt" zu setzen).
	// Rückbau wäre falsch — die Tür muss GEBAUT werden (Endpunkt + Rechte + UI).
	"StornierungGebuehr": "Feature-Lücke: einziger storniert_am-Schreiber, Endpunkt/UI fehlen noch",
}

func TestInterfaceOhneProduktivenAufrufer(t *testing.T) {
	deklariert := map[string]token.Position{} // Methodenname → Deklarationsort
	benutzt := map[string]bool{}              // Selektorname → irgendwo produktiv verwendet

	fset := token.NewFileSet()
	err := filepath.WalkDir(".", func(pfad string, eintrag fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if eintrag.IsDir() {
			name := eintrag.Name()
			if name == "frontend" || name == "node_modules" || name == ".git" || name == "e2e" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(pfad, ".go") || strings.HasSuffix(pfad, "_test.go") {
			return nil
		}
		datei, err := parser.ParseFile(fset, pfad, nil, 0)
		if err != nil {
			return err
		}
		ast.Inspect(datei, func(n ast.Node) bool {
			switch k := n.(type) {
			case *ast.InterfaceType:
				for _, m := range k.Methods.List {
					// Eingebettete Interfaces haben keine Namen — die deklarieren
					// wir nicht selbst, also zählen sie hier nicht.
					for _, name := range m.Names {
						deklariert[name.Name] = fset.Position(name.Pos())
					}
				}
			case *ast.SelectorExpr:
				benutzt[k.Sel.Name] = true
			}
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatalf("Quelltext durchlaufen: %v", err)
	}

	// Selbstprüfung des Scanners: Findet er die bekannten, sicher lebendigen
	// Türen nicht mehr, greift er nicht mehr — dann ist Grün keine Aussage.
	if len(deklariert) < 20 {
		t.Fatalf("nur %d Interface-Methoden gefunden — der Scanner greift vermutlich nicht mehr", len(deklariert))
	}
	if _, ok := deklariert["BulkUpsertBookTitles"]; !ok {
		t.Fatal("BulkUpsertBookTitles nicht als Interface-Methode erkannt — der Scanner greift nicht mehr")
	}

	for name, ort := range deklariert {
		if benutzt[name] {
			continue
		}
		if grund, ok := bewussteTueren[name]; ok {
			t.Logf("bewusste Tür ohne Aufrufer: %s (%s)", name, grund)
			continue
		}
		t.Errorf("Interface-Methode %s (%s) hat keinen produktiven Aufrufer — tote Tür.\n"+
			"→ Zurückbauen (Interface, Implementierung, Test) oder in bewussteTueren begründen.",
			name, ort)
	}
}
