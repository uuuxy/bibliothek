package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// Jede Advisory-Sperre braucht ihren eigenen Schlüssel: Zwei Sperren mit demselben Wert reihen
// Vorgänge hintereinander, die nichts miteinander zu tun haben. Gelesen werden die Aufrufe von
// pg_advisory_lock und pg_advisory_xact_lock mit $1 im Go-Code ohne Tests: Ihr Schlüssel muss
// eine Konstante des Pakets mit einer Zahl sein, und jeder Wert gehört genau einer Konstante.
//
// Blindheit: Sperren in SQL-Dateien (Migrationen, schema.sql, scripts/) und Schlüssel aus
// hashtext(); die Zahl der Auflagen in den Skripten hält auflagen_schreibpfad_ratsche_test.go.
// Ein Aufruf, dessen SQL erst aus Teilen entsteht, zählt nicht.
func TestAdvisorySperren_JederSchluesselEinmal(t *testing.T) {
	var quellen []sperrQuelle
	err := filepath.WalkDir(".", func(pfad string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch d.Name() {
			case ".git", "node_modules", "frontend", "vendor":
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(pfad, ".go") || strings.HasSuffix(pfad, "_test.go") {
			return nil
		}
		inhalt, err := os.ReadFile(pfad) //nolint:gosec // Testlauf über den eigenen Quellbaum
		if err != nil {
			return err
		}
		quellen = append(quellen, sperrQuelle{pfad: pfad, inhalt: string(inhalt)})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	schluessel, funde := pruefeSperrSchluessel(t, quellen)
	for _, fund := range funde {
		t.Error(fund)
	}
	// Nicht-leer-Garantie: Fände der Sammler die Sperren nicht mehr, prüfte er nichts.
	if len(schluessel) < 5 {
		t.Fatalf("nur %d Sperren gefunden: Der Sammler greift nicht mehr", len(schluessel))
	}
}

// Die Selbstprobe: derselbe Wert unter zwei Namen (auch mit Trennstrich geschrieben) und eine
// Zahl direkt im Aufruf werden gemeldet; dieselbe Konstante in zwei Aufrufen nicht.
func TestAdvisorySperren_Selbstprobe(t *testing.T) {
	quellen := []sperrQuelle{
		{pfad: "a/a.go", inhalt: `package a
const eins int64 = 7_1
func f(tx T) { tx.Exec(ctx, "SELECT pg_advisory_xact_lock($1)", eins) }
func g(tx T) { tx.Exec(ctx, "SELECT pg_advisory_xact_lock($1)", eins) }`},
		{pfad: "b/b.go", inhalt: "package b\nconst zwei int64 = 71\n" +
			"func h(tx T) { tx.Exec(ctx, `SELECT pg_advisory_lock($1)`, zwei) }\n" +
			"func i(tx T) { tx.Exec(ctx, \"SELECT pg_advisory_xact_lock($1)\", 72) }\n"},
	}
	_, funde := pruefeSperrSchluessel(t, quellen)
	if len(funde) != 2 {
		t.Fatalf("Funde %q, erwartet zwei: den doppelten Wert 71 und die Zahl im Aufruf", funde)
	}
	if !strings.Contains(funde[0], "71") || !strings.Contains(funde[1], "keine benannte Konstante") {
		t.Errorf("Funde %q", funde)
	}
}

// sperrQuelle ist eine Go-Datei mit ihrem Pfad.
type sperrQuelle struct {
	pfad, inhalt string
}

// pruefeSperrSchluessel liefert die Werte der gefundenen Sperren mit ihrer Konstante und die Funde.
func pruefeSperrSchluessel(t *testing.T, quellen []sperrQuelle) (map[int64]string, []string) {
	t.Helper()
	fset := token.NewFileSet()
	dateien := make([]*ast.File, len(quellen))
	konstanten := map[string]map[string]int64{} // Ordner → Name → Wert
	for i, q := range quellen {
		datei, err := parser.ParseFile(fset, q.pfad, q.inhalt, 0)
		if err != nil {
			t.Fatalf("%s: %v", q.pfad, err)
		}
		dateien[i] = datei
		ordner := filepath.Dir(q.pfad)
		if konstanten[ordner] == nil {
			konstanten[ordner] = map[string]int64{}
		}
		sammleZahlKonstanten(datei, konstanten[ordner])
	}

	schluessel := map[int64]string{}
	var funde []string
	for i, datei := range dateien {
		ordner := filepath.Dir(quellen[i].pfad)
		ast.Inspect(datei, func(n ast.Node) bool {
			aufruf, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			arg := sperrArgument(aufruf)
			if arg < 0 {
				return true
			}
			ort := fset.Position(aufruf.Pos())
			name, ok := aufruf.Args[arg].(*ast.Ident)
			wert, bekannt := konstanten[ordner][identName(name, ok)]
			if !ok || !bekannt {
				funde = append(funde, fmt.Sprintf("%s:%d: Der Schlüssel der Sperre ist keine benannte Konstante mit einer Zahl", ort.Filename, ort.Line))
				return true
			}
			eigen := ordner + "." + name.Name
			if frueher, belegt := schluessel[wert]; belegt && frueher != eigen {
				funde = append(funde, fmt.Sprintf("%s:%d: Schlüssel %d gehört schon %s, hier %s", ort.Filename, ort.Line, wert, frueher, eigen))
				return true
			}
			schluessel[wert] = eigen
			return true
		})
	}
	return schluessel, funde
}

// sammleZahlKonstanten trägt jede Konstante der Datei ein, deren Wert eine ganze Zahl ist.
func sammleZahlKonstanten(datei *ast.File, ziel map[string]int64) {
	for _, decl := range datei.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.CONST {
			continue
		}
		for _, spec := range gen.Specs {
			wert, ok := spec.(*ast.ValueSpec)
			if !ok || len(wert.Names) != len(wert.Values) {
				continue
			}
			for j, name := range wert.Names {
				lit, ok := wert.Values[j].(*ast.BasicLit)
				if !ok || lit.Kind != token.INT {
					continue
				}
				if zahl, err := strconv.ParseInt(lit.Value, 0, 64); err == nil {
					ziel[name.Name] = zahl
				}
			}
		}
	}
}

// sperrArgument liefert die Stelle des Schlüssels in einem Aufruf, dessen SQL eine Advisory-Sperre
// mit $1 nimmt; -1, wenn der Aufruf keine nimmt.
func sperrArgument(aufruf *ast.CallExpr) int {
	for i, arg := range aufruf.Args {
		lit, ok := arg.(*ast.BasicLit)
		if ok && lit.Kind == token.STRING && strings.Contains(lit.Value, "pg_advisory") && strings.Contains(lit.Value, "$1") {
			if i+1 < len(aufruf.Args) {
				return i + 1
			}
		}
	}
	return -1
}

// identName ist der Name eines Bezeichners, leer, wenn das Argument keiner ist.
func identName(name *ast.Ident, ok bool) string {
	if !ok {
		return ""
	}
	return name.Name
}
