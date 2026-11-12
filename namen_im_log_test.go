package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"io/fs"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"
	"testing"
	"unicode"
)

// Prüfung: Eine Logzeile nennt keine Person mit ihrem Namen (docs/STIL.md, Abschnitt 11: „Eine
// Person steht mit ihrer Kennung im Log, nicht mit ihrem Namen").
//
// Warum: Die Zeilen der Stufen ERROR und WARN gehen mit allen Feldern in die tägliche Alarm-Mail
// und verlassen damit den Server. Die Meldung einer Zeile ist fest (sloglint, static-msg); ein
// Name käme nur über ein Feld hinein.
//
// Regel: In jedem Aufruf von slog, der eine Zeile schreibt (Debug, Info, Warn, Error, je auch mit
// Context, Log, LogAttrs) oder Felder sammelt (With, Group, String, Any und die übrigen Felder),
// auch an einem Logger, dessen Name mit „log" oder „slog" beginnt, endet kein Schlüssel auf
// „name" (name, vorname, schueler_name), und kein Wert liest Vorname, Nachname, Anzeigename,
// VollerName oder Name. Name ist erlaubt unter den Schlüsseln datei und objekt: Dort ist es der
// Name einer Datei.
//
// Sieht nicht: einen ganzen Datensatz als Wert (slog schreibt dann jedes Feld, auch den Namen),
// einen Namen im Text eines Fehlers (die Schreiber der Littera-Übernahme tun das; sie laufen nur
// im Kommandozeilen-Werkzeug), Felder aus einer Variablen (args...) und einen Logger unter einem
// anderen Namen.

// namensfelder tragen den Namen einer Person, kleingeschrieben verglichen.
var namensfelder = map[string]bool{"vorname": true, "nachname": true, "anzeigename": true, "vollername": true}

// dateiSchluessel nennen eine Datei; Name ist dort der Name der Datei.
var dateiSchluessel = map[string]bool{"datei": true, "objekt": true}

// slogFeldBauer bauen ein einzelnes Feld: erst der Schlüssel, dann der Wert.
var slogFeldBauer = []string{"String", "Int", "Int64", "Uint64", "Float64", "Bool", "Time", "Duration", "Any"}

// argumenteVorFeldern zählt je Methode die Argumente vor den Feldern (Kontext, Stufe, Meldung).
var argumenteVorFeldern = map[string]int{
	"Debug": 1, "Info": 1, "Warn": 1, "Error": 1,
	"DebugContext": 2, "InfoContext": 2, "WarnContext": 2, "ErrorContext": 2,
	"Log": 3, "LogAttrs": 3, "With": 0,
}

// logFeld ist ein Feld einer Logzeile; schluessel bleibt leer, wenn er nicht als Text dasteht.
type logFeld struct {
	schluessel string
	wert       ast.Expr
}

// namenImLog nennt die Stellen der Datei, an denen ein Feld einer Logzeile einen Namen trägt,
// und zählt die Aufrufe, die eine Zeile schreiben.
func namenImLog(datei *ast.File, fset *token.FileSet) (funde []string, zeilen int) {
	pruefe := func(knoten ast.Node, felder []logFeld) {
		for _, feld := range felder {
			if nenntNamen(feld.schluessel) {
				funde = append(funde, fmt.Sprintf("%s: Schlüssel %q", fset.Position(knoten.Pos()), feld.schluessel))
			}
			if gelesen := liestNamen(feld.wert, !dateiSchluessel[feld.schluessel]); gelesen != "" {
				funde = append(funde, fmt.Sprintf("%s: Wert %s", fset.Position(knoten.Pos()), gelesen))
			}
		}
	}
	ast.Inspect(datei, func(n ast.Node) bool {
		switch knoten := n.(type) {
		case *ast.CallExpr:
			if felder, schreibtZeile, ok := logFelderDes(knoten); ok {
				if schreibtZeile {
					zeilen++
				}
				pruefe(knoten, felder)
			}
		case *ast.CompositeLit:
			if knoten.Type != nil && types.ExprString(knoten.Type) == "slog.Attr" {
				pruefe(knoten, []logFeld{attrLiteral(knoten)})
			}
		}
		return true
	})
	return funde, zeilen
}

// logFelderDes nennt die Felder eines Aufrufs, der eine Logzeile schreibt oder Felder sammelt; ok
// ist falsch für jeden anderen Aufruf.
func logFelderDes(aufruf *ast.CallExpr) (felder []logFeld, schreibtZeile, ok bool) {
	argumente := aufruf.Args
	if _, istFeld := istAufruf(aufruf, "slog", slogFeldBauer...); istFeld && len(argumente) == 2 {
		schluessel, _ := schluesselText(argumente[0])
		return []logFeld{{schluessel, argumente[1]}}, false, true
	}
	if _, istGruppe := istAufruf(aufruf, "slog", "Group"); istGruppe && len(argumente) > 0 {
		schluessel, _ := schluesselText(argumente[0])
		return append([]logFeld{{schluessel: schluessel}}, felderAus(argumente[1:])...), false, true
	}
	sel, istMethode := aufruf.Fun.(*ast.SelectorExpr)
	if !istMethode || !istLogger(sel.X) {
		return nil, false, false
	}
	vorFeldern, istLogAufruf := argumenteVorFeldern[sel.Sel.Name]
	if !istLogAufruf || len(argumente) < vorFeldern {
		return nil, false, false
	}
	return felderAus(argumente[vorFeldern:]), sel.Sel.Name != "With", true
}

// felderAus liest die Felder wie slog: Ein gebautes Feld steht für sich und wird an seinem eigenen
// Aufruf geprüft, sonst folgt auf einen Schlüssel sein Wert.
func felderAus(argumente []ast.Expr) []logFeld {
	var felder []logFeld
	for i := 0; i < len(argumente); i++ {
		if _, gebaut := istAufruf(argumente[i], "slog", append(slices.Clone(slogFeldBauer), "Group")...); gebaut {
			continue
		}
		schluessel, istText := schluesselText(argumente[i])
		if !istText || i+1 == len(argumente) {
			felder = append(felder, logFeld{wert: argumente[i]})
			continue
		}
		felder = append(felder, logFeld{schluessel, argumente[i+1]})
		i++
	}
	return felder
}

// attrLiteral liest Schlüssel und Wert aus slog.Attr{Key: …, Value: …}, auch ohne Feldnamen.
func attrLiteral(literal *ast.CompositeLit) logFeld {
	var feld logFeld
	for i, element := range literal.Elts {
		name, wert := map[int]string{0: "Key", 1: "Value"}[i], element
		if paar, ok := element.(*ast.KeyValueExpr); ok {
			name, wert = types.ExprString(paar.Key), paar.Value
		}
		switch name {
		case "Key":
			feld.schluessel, _ = schluesselText(wert)
		case "Value":
			feld.wert = wert
		}
	}
	return feld
}

// istLogger erkennt einen Empfänger, der nach einem Logger heißt: slog.Default(), logger,
// s.logAusgabe.
func istLogger(empfaenger ast.Expr) bool {
	teile := strings.FieldsFunc(types.ExprString(empfaenger), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_'
	})
	return slices.ContainsFunc(teile, func(teil string) bool {
		klein := strings.ToLower(teil)
		return strings.HasPrefix(klein, "log") || strings.HasPrefix(klein, "slog")
	})
}

// nenntNamen erkennt einen Schlüssel, der nach einem Namen heißt: name, vorname, schueler_name.
func nenntNamen(schluessel string) bool {
	return slices.ContainsFunc(strings.Split(strings.ToLower(schluessel), "_"), func(teil string) bool {
		return strings.HasSuffix(teil, "name") || strings.HasSuffix(teil, "namen")
	})
}

// liestNamen nennt das erste Namensfeld, das der Wert liest; mitName zählt auch das Feld Name.
func liestNamen(wert ast.Expr, mitName bool) string {
	if wert == nil {
		return ""
	}
	gelesen := ""
	ast.Inspect(wert, func(n ast.Node) bool {
		if gelesen != "" {
			return false
		}
		switch knoten := n.(type) {
		case *ast.SelectorExpr:
			if namensfelder[strings.ToLower(knoten.Sel.Name)] || (mitName && knoten.Sel.Name == "Name") {
				gelesen = types.ExprString(knoten)
			}
		case *ast.Ident:
			if namensfelder[strings.ToLower(knoten.Name)] {
				gelesen = knoten.Name
			}
		}
		return gelesen == ""
	})
	return gelesen
}

func schluesselText(ausdruck ast.Expr) (string, bool) {
	literal, ok := ausdruck.(*ast.BasicLit)
	if !ok || literal.Kind != token.STRING {
		return "", false
	}
	text, err := strconv.Unquote(literal.Value)
	return text, err == nil
}

func TestLogzeileNenntKeinenNamen(t *testing.T) {
	var funde []string
	zeilen := 0
	err := filepath.WalkDir(".", func(pfad string, eintrag fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if eintrag.IsDir() {
			switch eintrag.Name() {
			case "frontend", "node_modules", ".git", "e2e":
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(pfad, ".go") || strings.HasSuffix(pfad, "_test.go") {
			return nil
		}
		fset := token.NewFileSet()
		datei, err := parser.ParseFile(fset, pfad, nil, parser.SkipObjectResolution)
		if err != nil {
			return err
		}
		inDatei, zeilenInDatei := namenImLog(datei, fset)
		funde = append(funde, inDatei...)
		zeilen += zeilenInDatei
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%d Aufrufe geprüft, die eine Logzeile schreiben", zeilen)
	// Nicht-leer-Garantie: Erkennt der Detektor kaum Logzeilen, prüft er nichts.
	if zeilen < 250 {
		t.Fatalf("nur %d Aufrufe gefunden, die eine Logzeile schreiben — der Detektor greift nicht mehr", zeilen)
	}
	sort.Strings(funde)
	if len(funde) > 0 {
		t.Errorf("Logzeile mit dem Namen einer Person:\n  %s\n"+
			"Die Zeilen ERROR und WARN gehen mit allen Feldern in die Alarm-Mail. Die Person mit ihrer "+
			"Kennung nennen (leser, konto), nicht mit ihrem Namen (docs/STIL.md 11).",
			strings.Join(funde, "\n  "))
	}
}

func TestNamenImLogDetektorErkenntDieForm(t *testing.T) {
	faelle := []struct {
		aufruf string
		fund   bool
	}{
		{`slog.Error("x", "name", n)`, true},
		{`slog.Warn("x", "schueler_name", n)`, true},
		{`slog.Info("x", "leser", s.Vorname+" "+s.Nachname)`, true},
		{`slog.ErrorContext(ctx, "x", "wer", kurz(k.Anzeigename))`, true},
		{`slog.Warn("x", "schueler", s.Name)`, true},
		{`slog.Info("x", slog.String("vorname", v))`, true},
		{`slog.Info("x", slog.Group("schueler", "nachname", n))`, true},
		{`logger.With("nachname", n).Info("x")`, true},
		{`slog.Default().Warn("x", "wer", s.VollerName())`, true},
		{`slog.Log(ctx, slog.LevelWarn, "x", "wer", vorname)`, true},
		{`_ = slog.Attr{Key: "anzeigename", Value: slog.StringValue(n)}`, true},
		{`_ = slog.Attr{"klasse", slog.StringValue(s.Nachname)}`, true},
		{`slog.Info("x", "datei", d.Name)`, false},
		{`slog.Warn("x", "objekt", d.Name(), "fehler", err)`, false},
		{`slog.Error("x", "leser", id, "klasse", kl.Klasse, "fehler", err)`, false},
		{`logAusgabe.Info("x", "anzahl", n)`, false},
		{`fmt.Println("name", s.Vorname)`, false},
		{`katalog.Info("name", s.Vorname)`, false},
	}
	for _, fall := range faelle {
		fset := token.NewFileSet()
		datei, err := parser.ParseFile(fset, "probe.go", "package p\nfunc f() {\n"+fall.aufruf+"\n}\n", 0)
		if err != nil {
			t.Fatalf("%s: %v", fall.aufruf, err)
		}
		if funde, _ := namenImLog(datei, fset); (len(funde) > 0) != fall.fund {
			t.Errorf("%s: Funde %v, erwartet ein Fund: %v", fall.aufruf, funde, fall.fund)
		}
	}

	fset := token.NewFileSet()
	quelle := "package p\nfunc f() {\nslog.Info(\"x\")\nlogger.With(\"a\", 1).Warn(\"y\")\n_ = slog.String(\"b\", s)\n}\n"
	datei, err := parser.ParseFile(fset, "probe.go", quelle, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, zeilen := namenImLog(datei, fset); zeilen != 2 {
		t.Errorf("%d Aufrufe als Logzeile gezählt, erwartet 2 (Info und Warn, nicht With und String)", zeilen)
	}
}
