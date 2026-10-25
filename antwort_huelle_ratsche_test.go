package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

// Eine Tür antwortet mit dem Wert selbst, ohne Hülle {"data": …}. Solange es beide Formen gab, las
// die Oberfläche an Türen, die den Wert selbst schickten, .data und bekam eine leere Liste, ohne
// dass ein Fehler erschien: Die Leserakte zeigte so keinen Bescheid (docs/sweeps.md, „Lesestelle
// erwartet die Antwortform des Nachbarpakets").
//
// Die Regel: Unter api/ trägt kein Map-Literal den Schlüssel "data", und im ganzen Baum trägt kein
// Feld den JSON-Namen data. Felder zählen überall, weil auch ein Typ aus repository/ oder
// internal/ als Antwort hinausgeht.
//
// Blindheit: Ein Schlüssel, der erst zur Laufzeit entsteht, und eine Hülle unter anderem Namen
// („result", „items"). Ob die Oberfläche eine Antwort in der Form liest, die die Tür schickt,
// prüft sie nicht; das zeigen die Tests im Browser.
func TestAntwortOhneHuelle(t *testing.T) {
	var verstoesse []string
	gelesen, unterAPI := 0, 0

	fset := token.NewFileSet()
	err := filepath.WalkDir(".", func(pfad string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch d.Name() {
			case "node_modules", ".git", "frontend", "testdata":
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
		pfad = filepath.ToSlash(pfad)
		gelesen++
		istTuer := strings.HasPrefix(pfad, "api/")
		if istTuer {
			unterAPI++
		}
		for _, fund := range huellenIn(fset, datei, istTuer) {
			verstoesse = append(verstoesse, pfad+":"+fund)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Baum nicht lesbar: %v", err)
	}
	if gelesen < 300 || unterAPI < 100 {
		t.Fatalf("nur %d Dateien gelesen, %d davon unter api/ — die Prüfung misst nichts", gelesen, unterAPI)
	}
	for _, v := range verstoesse {
		t.Errorf("%s — eine Tür antwortet mit dem Wert selbst, ohne Hülle {\"data\": …} "+
			"(docs/sweeps.md, „Lesestelle erwartet die Antwortform des Nachbarpakets“)", v)
	}
}

// huellenIn nennt die Stellen einer Datei, die eine Antwort in {"data": …} legen: ein Feld mit dem
// JSON-Namen data, mitMapSchluessel auch ein Map-Literal mit dem Schlüssel "data".
func huellenIn(fset *token.FileSet, datei *ast.File, mitMapSchluessel bool) []string {
	var funde []string
	ast.Inspect(datei, func(n ast.Node) bool {
		switch k := n.(type) {
		case *ast.KeyValueExpr:
			if lit, ok := k.Key.(*ast.BasicLit); ok && mitMapSchluessel && lit.Kind == token.STRING && lit.Value == `"data"` {
				funde = append(funde, strconv.Itoa(fset.Position(lit.Pos()).Line)+`: Schlüssel "data"`)
			}
		case *ast.Field:
			if k.Tag == nil {
				return true
			}
			tag, err := strconv.Unquote(k.Tag.Value)
			if err != nil {
				return true
			}
			if name, _, _ := strings.Cut(reflect.StructTag(tag).Get("json"), ","); name == "data" {
				funde = append(funde, strconv.Itoa(fset.Position(k.Pos()).Line)+`: Feld mit dem JSON-Namen data`)
			}
		}
		return true
	})
	return funde
}

func TestHuellenIn_ErkenntDieFormen(t *testing.T) {
	faelle := []struct {
		was              string
		quelle           string
		mitMapSchluessel bool
		funde            int
	}{
		{"Schlüssel in einer Tür", `package p; var a = map[string]any{"data": 1}`, true, 1},
		{"Schlüssel außerhalb von api/", `package p; var a = map[string]any{"data": 1}`, false, 0},
		{"anderer Schlüssel", `package p; var a = map[string]any{"daten": 1, "meldung": "x"}`, true, 0},
		{"Feld mit Tag", "package p; type T struct { Inhalt []int `json:\"data\"` }", false, 1},
		{"Feld mit Tag und Option", "package p; type T struct { Inhalt []int `json:\"data,omitempty\"` }", false, 1},
		{"Feld ohne Tag", `package p; type T struct { Data []int }`, false, 0},
		{"Feld mit anderem Namen", "package p; type T struct { Data []int `json:\"daten\"` }", false, 0},
	}
	for _, f := range faelle {
		fset := token.NewFileSet()
		datei, err := parser.ParseFile(fset, "probe.go", f.quelle, 0)
		if err != nil {
			t.Fatalf("%s: %v", f.was, err)
		}
		if funde := huellenIn(fset, datei, f.mitMapSchluessel); len(funde) != f.funde {
			t.Errorf("%s: %d Funde %v, erwartet %d", f.was, len(funde), funde, f.funde)
		}
	}
}
