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

// Prüfung: Eine Leseschleife meldet eine Zeile, die sich nicht lesen lässt, statt sie auszulassen.
//
// Warum: Lässt die Schleife die Zeile aus, kommt die Liste kürzer an, und niemand erfährt es. Vor
// dem Löschen eines Titels hieße eine ausgelassene Ausleihe „nichts verliehen". Die Regel steht
// in repository/zeile_unlesbar.go.
//
// Regel: In einer Schleife über X.Next() führt der Fehler eines Scan weder zu continue noch
// übernimmt die Schleife die Zeile nur bei err == nil, und der Zweig des Fehlers endet mit return
// oder einem Abbruch (t.Fatal…, log.Fatal…, os.Exit, panic). Tests eingeschlossen.
//
// Sieht nicht: einen Fehler, der erst später geprüft wird, und ein Scan in einer eigenen
// Hilfsfunktion.
func TestLeseschleifen_LassenKeineZeileAus(t *testing.T) {
	schleifen, verstoesse := pruefeLeseschleifen(t, ".")
	// Untergrenze: Findet der Durchlauf kaum Schleifen, prüft er nichts.
	if schleifen < 150 {
		t.Fatalf("nur %d Leseschleifen mit Scan gefunden, erwartet mindestens 150 — der Detektor läuft leer", schleifen)
	}
	t.Logf("%d Leseschleifen mit Scan gelesen", schleifen)
	if len(verstoesse) > 0 {
		t.Errorf("Leseschleife lässt eine unlesbare Zeile aus:\n  %s\n"+
			"Fix: den Fehler zurückgeben, in repository/ als zeileUnlesbar(err) (repository/zeile_unlesbar.go).",
			strings.Join(verstoesse, "\n  "))
	}
}

// pruefeLeseschleifen zählt die Schleifen über X.Next(), die ein Scan tragen, und nennt jede
// Stelle, an der der Fehler eines Scan die Zeile auslässt.
func pruefeLeseschleifen(t *testing.T, wurzel string) (int, []string) {
	t.Helper()
	fset := token.NewFileSet()
	schleifen := 0
	var verstoesse []string
	err := filepath.WalkDir(wurzel, func(pfad string, eintrag fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if eintrag.IsDir() {
			switch eintrag.Name() {
			case "frontend", "node_modules", ".git", "vendor":
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(pfad, ".go") {
			return nil
		}
		datei, err := parser.ParseFile(fset, pfad, nil, 0)
		if err != nil {
			return err
		}
		ast.Inspect(datei, func(n ast.Node) bool {
			schleife, ok := n.(*ast.ForStmt)
			if !ok || !istNextAufruf(schleife.Cond) {
				return true
			}
			if traegtScan(schleife.Body) {
				schleifen++
			}
			for i, anweisung := range schleife.Body.List {
				wenn, ok := anweisung.(*ast.IfStmt)
				if !ok || !prueftScanFehler(wenn, schleife.Body.List, i) {
					continue
				}
				if grund := lassenAus(wenn); grund != "" {
					verstoesse = append(verstoesse, fset.Position(wenn.Pos()).String()+": "+grund)
				}
			}
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(verstoesse)
	return schleifen, verstoesse
}

// istNextAufruf meldet, ob die Bedingung einer Schleife ein Aufruf X.Next() ist.
func istNextAufruf(e ast.Expr) bool {
	aufruf, ok := e.(*ast.CallExpr)
	if !ok {
		return false
	}
	auswahl, ok := aufruf.Fun.(*ast.SelectorExpr)
	return ok && auswahl.Sel.Name == "Next" && len(aufruf.Args) == 0
}

// istScanAufruf meldet, ob der Ausdruck ein Aufruf X.Scan(…) ist.
func istScanAufruf(e ast.Expr) bool {
	aufruf, ok := e.(*ast.CallExpr)
	if !ok {
		return false
	}
	auswahl, ok := aufruf.Fun.(*ast.SelectorExpr)
	return ok && auswahl.Sel.Name == "Scan"
}

// traegtScan meldet, ob ein Block irgendwo ein Scan aufruft, außerhalb eigener Funktionen.
func traegtScan(block *ast.BlockStmt) bool {
	gefunden := false
	ast.Inspect(block, func(n ast.Node) bool {
		if _, ok := n.(*ast.FuncLit); ok {
			return false
		}
		if e, ok := n.(ast.Expr); ok && istScanAufruf(e) {
			gefunden = true
		}
		return !gefunden
	})
	return gefunden
}

// prueftScanFehler meldet, ob das if den Fehler eines Scan prüft: im eigenen Init, in der
// Anweisung davor oder direkt in der Bedingung.
func prueftScanFehler(wenn *ast.IfStmt, anweisungen []ast.Stmt, i int) bool {
	if zuweisung, ok := wenn.Init.(*ast.AssignStmt); ok && len(zuweisung.Rhs) == 1 && istScanAufruf(zuweisung.Rhs[0]) {
		return true
	}
	if i > 0 {
		if zuweisung, ok := anweisungen[i-1].(*ast.AssignStmt); ok && len(zuweisung.Rhs) == 1 && istScanAufruf(zuweisung.Rhs[0]) {
			return true
		}
	}
	vergleich, ok := wenn.Cond.(*ast.BinaryExpr)
	return ok && istScanAufruf(vergleich.X)
}

// lassenAus nennt, wie das if eine unlesbare Zeile auslässt, oder liefert "".
func lassenAus(wenn *ast.IfStmt) string {
	vergleich, ok := wenn.Cond.(*ast.BinaryExpr)
	if !ok {
		return ""
	}
	if nichts, ok := vergleich.Y.(*ast.Ident); !ok || nichts.Name != "nil" {
		return ""
	}
	switch {
	case vergleich.Op == token.EQL:
		return "übernimmt die Zeile nur, wenn Scan gelingt"
	case vergleich.Op != token.NEQ:
		return ""
	case enthaelt(wenn.Body, istContinue) && !enthaelt(wenn.Body, istEnde):
		return "springt bei einem Fehler von Scan mit continue zur nächsten Zeile"
	case !enthaelt(wenn.Body, istEnde):
		return "läuft nach einem Fehler von Scan weiter, ohne zurückzukehren"
	}
	return ""
}

// enthaelt meldet, ob ein Knoten im Block die Bedingung erfüllt, außerhalb eigener Funktionen.
func enthaelt(block *ast.BlockStmt, bedingung func(ast.Node) bool) bool {
	gefunden := false
	ast.Inspect(block, func(n ast.Node) bool {
		if _, ok := n.(*ast.FuncLit); ok {
			return false
		}
		if bedingung(n) {
			gefunden = true
		}
		return !gefunden
	})
	return gefunden
}

func istContinue(n ast.Node) bool {
	sprung, ok := n.(*ast.BranchStmt)
	return ok && sprung.Tok == token.CONTINUE
}

// istEnde meldet ein return oder einen Aufruf, der nicht zurückkehrt.
func istEnde(n ast.Node) bool {
	if _, ok := n.(*ast.ReturnStmt); ok {
		return true
	}
	aufruf, ok := n.(*ast.CallExpr)
	if !ok {
		return false
	}
	switch fun := aufruf.Fun.(type) {
	case *ast.Ident:
		return fun.Name == "panic"
	case *ast.SelectorExpr:
		switch fun.Sel.Name {
		case "Fatal", "Fatalf", "Fatalln", "FailNow", "Exit", "Skip", "Skipf", "SkipNow":
			return true
		}
	}
	return false
}
