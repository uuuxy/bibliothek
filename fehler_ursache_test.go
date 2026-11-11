package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// Prüfung gegen die verworfene Ursache eines 500.
//
// Warum: Eine Tür, die im Zweig `if err != nil` mit 500 antwortet und dabei einen festen Text
// statt des Fehlers übergibt (`errors.New("database error")`), schreibt nur diesen Text ins Log.
// Die Ursache aus der Datenbank fehlt, und ein Abbruch durch den Browser in der Kette wird nicht
// mehr als Abbruch erkannt (apierrors.istAbbruch). Die Antwort an die Oberfläche bleibt ohnehin
// neutral; die Ursache geht nur ins Log.
//
// Regel: Ein `if`, dessen Bedingung `<err> != nil` enthält und dessen Rumpf mit 500 antwortet
// (http.StatusInternalServerError oder apierrors.Internal), nennt den Fehler im Rumpf, etwa
// `fmt.Errorf("…: %w", err)`. Reparatur bei Rot: den Fehler mit %w einpacken.

// fehlerNamenImVergleich: die Namen, die in der Bedingung mit `!= nil` verglichen werden.
func fehlerNamenImVergleich(cond ast.Expr) []string {
	var namen []string
	ast.Inspect(cond, func(n ast.Node) bool {
		be, ok := n.(*ast.BinaryExpr)
		if !ok || be.Op != token.NEQ {
			return true
		}
		id, ok := be.X.(*ast.Ident)
		if nilWert, istNil := be.Y.(*ast.Ident); ok && istNil && nilWert.Name == "nil" && istFehlerName(id.Name) {
			namen = append(namen, id.Name)
		}
		return true
	})
	return namen
}

// antwortet500 erkennt die Antwort 500 als Status-Konstante und als Helfer der Hausform.
func antwortet500(body *ast.BlockStmt) bool {
	gefunden := false
	ast.Inspect(body, func(n ast.Node) bool {
		if s, ok := n.(*ast.SelectorExpr); ok {
			paket, istName := s.X.(*ast.Ident)
			if s.Sel.Name == "StatusInternalServerError" || s.Sel.Name == "Internal" && istName && paket.Name == "apierrors" {
				gefunden = true
			}
		}
		return !gefunden
	})
	return gefunden
}

func nenntEinen(body *ast.BlockStmt, namen []string) bool {
	genannt := false
	ast.Inspect(body, func(n ast.Node) bool {
		if id, ok := n.(*ast.Ident); ok {
			for _, name := range namen {
				if id.Name == name {
					genannt = true
				}
			}
		}
		return !genannt
	})
	return genannt
}

// verwirftUrsache meldet ein if, das einen Fehler prüft, mit 500 antwortet und den Fehler nicht
// weiterreicht.
func verwirftUrsache(ifs *ast.IfStmt) bool {
	namen := fehlerNamenImVergleich(ifs.Cond)
	return len(namen) > 0 && antwortet500(ifs.Body) && !nenntEinen(ifs.Body, namen)
}

func TestKein500VerwirftDieUrsache(t *testing.T) {
	var treffer []string
	geprueft := 0 // Zweige mit Fehler und 500: der Boden gegen einen blinden Lauf
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
			ifs, ok := n.(*ast.IfStmt)
			if !ok {
				return true
			}
			if len(fehlerNamenImVergleich(ifs.Cond)) > 0 && antwortet500(ifs.Body) {
				geprueft++
			}
			if verwirftUrsache(ifs) {
				treffer = append(treffer, fset.Position(ifs.Pos()).String())
			}
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if geprueft < 250 {
		t.Fatalf("nur %d Zweige mit Fehler und 500 gefunden, erwartet über 250 (334 beim Bau): Läuft die Prüfung "+
			"noch im Wurzelordner, erkennt sie die Antwort noch?", geprueft)
	}
	sort.Strings(treffer)
	if len(treffer) > 0 {
		t.Errorf("500 ohne Ursache: Der Zweig prüft einen Fehler und gibt ihn nicht weiter:\n  %s\n"+
			"Fix: fmt.Errorf(\"…: %%w\", err) statt eines festen Texts.", strings.Join(treffer, "\n  "))
	}
}

// Gegenprobe am Detektor: ein Muster, das nichts findet, meldet ewig „alles gut“.
func TestVerworfeneUrsacheDetektorErkenntDieForm(t *testing.T) {
	quelle := `package p
func verwirft(w http.ResponseWriter) { _, err := f(); if err != nil { apierrors.AntworteFehler(w, http.StatusInternalServerError, errors.New("database error")); return } }
func verwirft2() error { _, err := f(); if err != nil { return apierrors.Internal("x", nil) }; return nil }
func verwirft3(w http.ResponseWriter) { if _, ladeErr := f(); ladeErr != nil { http.Error(w, "kaputt", http.StatusInternalServerError) } }
func sauber(w http.ResponseWriter) { _, err := f(); if err != nil { apierrors.AntworteFehler(w, http.StatusInternalServerError, fmt.Errorf("x: %w", err)); return } }
func sauber2() error { _, err := f(); if err != nil { return apierrors.Internal("x", err) }; return nil }
func sauber3(w http.ResponseWriter) { _, err := f(); if err != nil { apierrors.AntworteFehlerMitMeldung(w, http.StatusBadRequest, "Datum falsch", nil) } }
func sauber4(w http.ResponseWriter) { if ok := f(); ok != nil { w.WriteHeader(http.StatusInternalServerError) } }
`
	fset := token.NewFileSet()
	datei, err := parser.ParseFile(fset, "probe.go", quelle, 0)
	if err != nil {
		t.Fatal(err)
	}
	// sauber3: Ein 400 mit fester Meldung zu einer Eingabe verliert keine Diagnose. sauber4: ok
	// ist kein Fehlername.
	erwartet := map[string]bool{"verwirft": true, "verwirft2": true, "verwirft3": true,
		"sauber": false, "sauber2": false, "sauber3": false, "sauber4": false}
	gesehen := 0
	for _, decl := range datei.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		gesehen++
		gefunden := false
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			if ifs, ok := n.(*ast.IfStmt); ok && verwirftUrsache(ifs) {
				gefunden = true
			}
			return true
		})
		if gefunden != erwartet[fn.Name.Name] {
			t.Errorf("%s: erkannt=%v, erwartet %v", fn.Name.Name, gefunden, erwartet[fn.Name.Name])
		}
	}
	if gesehen != len(erwartet) {
		t.Fatalf("%d Funktionen in der Probe gelesen, erwartet %d", gesehen, len(erwartet))
	}
}
