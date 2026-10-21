package docs

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

// Der Teil der Ratsche „Tag in der falschen Zeitzone", der am Syntaxbaum misst: ein Kalenderwert
// aus der Uhr des Containers. Zwei Formen:
//
//  1. time.Now() als Tag, Jahr oder Text, auch über AddDate oder Add: time.Now().Year(),
//     time.Now().AddDate(0, 0, 28).Format(…). Eine Kette mit In(…) oder UTC() nennt ihre Zone
//     und zählt nicht.
//  2. eine Funktion, die einen Zeitpunkt jetzt oder heute bekommt und an ihm Year, Month, Day,
//     Weekday, YearDay, ISOWeek, Date, Clock, Hour oder Format aufruft, ohne ihn vorher mit
//     In(…) umzurechnen. Die Funktion rechnet um, nicht ihr Aufrufer: So gilt die Regel für
//     jeden, der time.Now() hineinreicht.
//
// Blind für einen Zeitpunkt aus der Datenbank (pgx liest ihn in time.Local, im Betrieb UTC), für
// einen Parameter mit anderem Namen und für eine Umrechnung, die nur in einem Zweig steht.

// kalenderMethoden machen aus einem Zeitpunkt einen Tag, ein Jahr oder einen Text; an ihnen
// hängt die Zone.
var kalenderMethoden = map[string]bool{
	"Year": true, "Month": true, "Day": true, "Weekday": true, "YearDay": true, "ISOWeek": true,
	"Date": true, "Clock": true, "Hour": true, "Format": true, "AppendFormat": true,
}

// verschiebeMethoden lassen die Zone eines Zeitpunkts, wie sie ist.
var verschiebeMethoden = map[string]bool{"AddDate": true, "Add": true, "Truncate": true, "Round": true}

// uhrFunde zählt die Kalender-Aufrufe einer Datei, die mit der Uhr des Containers rechnen.
func uhrFunde(datei *ast.File) int {
	funde := 0
	for _, decl := range datei.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}
		jetztParameter := zeitParameter(fn)
		umgerechnet := map[string]token.Pos{}
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			if zuweisung, ok := n.(*ast.AssignStmt); ok {
				for i, links := range zuweisung.Lhs {
					id, ok := links.(*ast.Ident)
					if ok && jetztParameter[id.Name] && i < len(zuweisung.Rhs) && nenntZone(zuweisung.Rhs[i]) {
						umgerechnet[id.Name] = zuweisung.Pos()
					}
				}
			}
			aufruf, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := aufruf.Fun.(*ast.SelectorExpr)
			if !ok || !kalenderMethoden[sel.Sel.Name] {
				return true
			}
			wurzel, mitZone := kettenWurzel(sel.X)
			if mitZone {
				return true
			}
			if istTimeNow(wurzel) {
				funde++
			}
			if id, ok := wurzel.(*ast.Ident); ok && jetztParameter[id.Name] {
				if pos, ok := umgerechnet[id.Name]; !ok || pos > aufruf.Pos() {
					funde++
				}
			}
			return true
		})
	}
	return funde
}

// zeitParameter sind die Parameter vom Typ time.Time, die den Zeitpunkt des Aufrufs tragen.
func zeitParameter(fn *ast.FuncDecl) map[string]bool {
	namen := map[string]bool{}
	for _, feld := range fn.Type.Params.List {
		typ, ok := feld.Type.(*ast.SelectorExpr)
		if !ok || typ.Sel.Name != "Time" {
			continue
		}
		if paket, ok := typ.X.(*ast.Ident); !ok || paket.Name != "time" {
			continue
		}
		for _, name := range feld.Names {
			if name.Name == "jetzt" || name.Name == "heute" {
				namen[name.Name] = true
			}
		}
	}
	return namen
}

// kettenWurzel geht von x über AddDate, Add, Truncate und Round zum Anfang der Kette; mitZone
// sagt, ob unterwegs In(…) oder UTC() die Zone nennt.
func kettenWurzel(x ast.Expr) (ast.Expr, bool) {
	for {
		aufruf, ok := x.(*ast.CallExpr)
		if !ok {
			return x, false
		}
		sel, ok := aufruf.Fun.(*ast.SelectorExpr)
		if !ok {
			return x, false
		}
		switch {
		case sel.Sel.Name == "In" || sel.Sel.Name == "UTC":
			return x, true
		case verschiebeMethoden[sel.Sel.Name]:
			x = sel.X
		default:
			return x, false
		}
	}
}

// nenntZone: Der Ausdruck rechnet einen Zeitpunkt in eine Zone um (… .In(…)).
func nenntZone(x ast.Expr) bool {
	_, mitZone := kettenWurzel(x)
	return mitZone
}

func istTimeNow(x ast.Expr) bool {
	aufruf, ok := x.(*ast.CallExpr)
	if !ok {
		return false
	}
	sel, ok := aufruf.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "Now" {
		return false
	}
	paket, ok := sel.X.(*ast.Ident)
	return ok && paket.Name == "time"
}

// Der Detektor selbst: Ohne diese Fälle könnte ein Umbau still nichts mehr finden.
func TestKalendertag_SelbstprobeDerUhr(t *testing.T) {
	faelle := []struct {
		name, quelle string
		funde        int
	}{
		{"Jahr aus time.Now", `func f() int { return time.Now().Year() }`, 1},
		{"Datum über AddDate", `func f() string { return time.Now().AddDate(0, 0, 28).Format("2006-01-02") }`, 1},
		{"UTC ausdrücklich", `func f() string { return time.Now().UTC().Format(time.RFC3339) }`, 0},
		{"Zone der Schule", `func f() int { return time.Now().In(zone).Year() }`, 0},
		{"jetzt ohne Umrechnung", `func f(jetzt time.Time) int { return jetzt.Year() }`, 1},
		{"jetzt über AddDate", `func f(jetzt time.Time) time.Month { return jetzt.AddDate(0, 1, 0).Month() }`, 1},
		{"jetzt umgerechnet", `func f(jetzt time.Time) int { jetzt = jetzt.In(zone); return jetzt.Year() }`, 0},
		{"Umrechnung zu spät", `func f(jetzt time.Time) int { j := jetzt.Year(); jetzt = jetzt.In(zone); return j }`, 1},
		{"jetzt nur verglichen", `func f(jetzt, frist time.Time) bool { return jetzt.After(frist) }`, 0},
		{"anderer Name", `func f(tag time.Time) int { return tag.Year() }`, 0},
	}
	for _, f := range faelle {
		datei, err := parser.ParseFile(token.NewFileSet(), "probe.go", "package probe\n\nimport \"time\"\n\n"+f.quelle, 0)
		if err != nil {
			t.Fatalf("%s: %v", f.name, err)
		}
		if got := uhrFunde(datei); got != f.funde {
			t.Errorf("%s: %d Funde, erwartet %d", f.name, got, f.funde)
		}
	}
}
