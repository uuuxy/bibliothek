package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

// Ein Test setzt keine Variable der Paketebene um, weder des eigenen Pakets noch eines fremden
// (paket.Name = …). Googles Regelwerk nennt solchen Zustand unsicher, wenn „independent test
// cases interact with each other through global state" (Best Practices, „Global state"): Der
// Test, der den Wert umsetzt, verändert jeden Test nach ihm, und ein vergessenes Zurücksetzen
// macht eine Prüfung grün oder rot je nach Reihenfolge der Dateien. Was ein Test austauschen
// will, gehört einem Typ, den der Rufer anlegt (tuer.Basis mit Mail und Rechte,
// katalog.Tueren mit cover, auth.Anmeldung mit imapTLS).
//
// Blindheit: Gesehen werden Zuweisungen, auch zusammengesetzte (+=) und ++/--, an einen Namen,
// den eine Programmdatei desselben Ordners auf Paketebene erklärt, und an paket.Name eines
// eingebundenen Pakets. Nicht gesehen werden Funktionen und Methoden, die einen Wert des Pakets
// setzen, Schreiben über einen Zeiger oder eine Map der Paketebene, die ein Test befüllt, und
// Variablen, die eine Testdatei selbst erklärt. Dass es im Programmcode keinen Wert gibt, den sie
// ändern könnten, hält TestPaketzustand_ProgrammcodeHaeltKeinenZustand.
func TestPaketzustand_TestsSetzenKeineVariableDesPakets(t *testing.T) {
	funde, gelesen := paketzustandFunde(t, ".")
	if gelesen < 500 {
		t.Fatalf("nur %d Testdateien gelesen: Der Sammler greift ins Leere", gelesen)
	}
	for _, f := range funde {
		t.Errorf("%s setzt eine Variable der Paketebene um. Der Wert gehört einem Typ, den der "+
			"Rufer anlegt und der Test selbst baut (docs/STIL.md, Abschnitt 10)", f)
	}
}

func TestPaketzustand_Selbstprobe(t *testing.T) {
	quellen := map[string]string{
		"a/a.go": "package a\nvar zaehler int\nvar Lader func()\n",
		"a/a_test.go": `package a
import (
	"os"
	q "bibliothek/quelle"
)
var eigene int
func TestX(t *testing.T) {
	zaehler = 1
	zaehler += 2
	zaehler++
	os.Stdout = nil
	q.Wert, eigene = 3, 4
	eigene = 5
	zaehler := 7
	zaehler = 8
	var lokal struct{ Lader func() }
	lokal.Lader = nil
}
func hilfe(Lader func()) { Lader = nil }
`,
	}
	funde := paketzustandIn(t, quellen)
	erwartet := []string{
		"a/a_test.go:8: zaehler", "a/a_test.go:9: zaehler", "a/a_test.go:10: zaehler",
		"a/a_test.go:11: os.Stdout", "a/a_test.go:12: q.Wert",
	}
	if strings.Join(funde, "\n") != strings.Join(erwartet, "\n") {
		t.Errorf("Funde:\n%s\nerwartet:\n%s", strings.Join(funde, "\n"), strings.Join(erwartet, "\n"))
	}
}

// paketzustandFunde liest alle Go-Dateien unter wurzel und liefert die Funde mit der Zahl der
// gelesenen Testdateien.
func paketzustandFunde(t *testing.T, wurzel string) ([]string, int) {
	t.Helper()
	quellen := leseGoQuellen(t, wurzel)
	tests := 0
	for pfad := range quellen {
		if strings.HasSuffix(pfad, "_test.go") {
			tests++
		}
	}
	return paketzustandIn(t, quellen), tests
}

// leseGoQuellen liest alle Go-Dateien unter wurzel, je mit dem Pfad relativ zu wurzel.
func leseGoQuellen(t *testing.T, wurzel string) map[string]string {
	t.Helper()
	quellen := map[string]string{}
	err := filepath.WalkDir(wurzel, func(pfad string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if name := d.Name(); name == "node_modules" || name == "frontend" || name == "testdata" ||
				(strings.HasPrefix(name, ".") && pfad != wurzel) {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(pfad, ".go") {
			return nil
		}
		inhalt, err := os.ReadFile(pfad)
		if err != nil {
			return err
		}
		relativ, err := filepath.Rel(wurzel, pfad)
		if err != nil {
			return err
		}
		quellen[filepath.ToSlash(relativ)] = string(inhalt)
		return nil
	})
	if err != nil {
		t.Fatalf("Baum lesen: %v", err)
	}
	return quellen
}

// paketzustandIn prüft die Testdateien gegen die Variablen, die die Programmdateien desselben
// Ordners auf Paketebene erklären.
func paketzustandIn(t *testing.T, quellen map[string]string) []string {
	t.Helper()
	pfade := make([]string, 0, len(quellen))
	for p := range quellen {
		pfade = append(pfade, p)
	}
	sort.Strings(pfade)
	fset := token.NewFileSet()
	paketVariablen := map[string]map[string]bool{}
	for _, p := range pfade {
		if strings.HasSuffix(p, "_test.go") {
			continue
		}
		datei, err := parser.ParseFile(fset, p, quellen[p], parser.SkipObjectResolution)
		if err != nil {
			t.Fatalf("%s: %v", p, err)
		}
		ordner := filepath.Dir(p)
		if paketVariablen[ordner] == nil {
			paketVariablen[ordner] = map[string]bool{}
		}
		for _, dekl := range datei.Decls {
			if g, ok := dekl.(*ast.GenDecl); ok && g.Tok == token.VAR {
				for _, spez := range g.Specs {
					werte, ok := spez.(*ast.ValueSpec)
					if !ok {
						continue
					}
					for _, n := range werte.Names {
						paketVariablen[ordner][n.Name] = true
					}
				}
			}
		}
	}
	var funde []string
	for _, p := range pfade {
		if !strings.HasSuffix(p, "_test.go") {
			continue
		}
		// Mit Auflösung der Namen: Ein Name, den die Datei selbst erklärt (lokal oder auf ihrer
		// Paketebene), trägt sein Objekt; ein Name aus einer Programmdatei bleibt offen.
		datei, err := parser.ParseFile(fset, p, quellen[p], 0)
		if err != nil {
			t.Fatalf("%s: %v", p, err)
		}
		importe := map[string]bool{}
		for _, imp := range datei.Imports {
			pfad, err := strconv.Unquote(imp.Path.Value)
			if err != nil {
				t.Fatalf("%s: Pfad des Imports %s: %v", p, imp.Path.Value, err)
			}
			name := filepath.Base(pfad)
			if imp.Name != nil {
				name = imp.Name.Name
			}
			importe[name] = true
		}
		variablen := paketVariablen[filepath.Dir(p)]
		pruefe := func(ziel ast.Expr) {
			switch x := ziel.(type) {
			case *ast.SelectorExpr:
				if id, ok := x.X.(*ast.Ident); ok && id.Obj == nil && importe[id.Name] {
					funde = append(funde, p+":"+strconv.Itoa(fset.Position(x.Pos()).Line)+": "+id.Name+"."+x.Sel.Name)
				}
			case *ast.Ident:
				if x.Obj == nil && variablen[x.Name] {
					funde = append(funde, p+":"+strconv.Itoa(fset.Position(x.Pos()).Line)+": "+x.Name)
				}
			}
		}
		ast.Inspect(datei, func(n ast.Node) bool {
			switch s := n.(type) {
			case *ast.AssignStmt:
				if s.Tok != token.DEFINE {
					for _, ziel := range s.Lhs {
						pruefe(ziel)
					}
				}
			case *ast.IncDecStmt:
				pruefe(s.X)
			}
			return true
		})
	}
	return funde
}

// Programmcode hält auf Paketebene keinen Wert, der sich ändert; ein Programm (package main)
// darf es. Googles Regelwerk: „Libraries should not force their clients to use APIs that rely
// on global state", und sicher ist solcher Zustand nur, wenn er „logically constant" ist (Best
// Practices, „Global state"). Was sich ändert, gehört einem Typ, den der Rufer anlegt und
// weiterreicht (auth.Anmeldung mit der Bremse der Fehlversuche, service.CoverDienst mit seiner
// Sperre, clientip.Ermittler am Server).
//
// Erlaubt ist ein Anfangswert aus Literalen von Map, Slice und Array, ein Text (Verkettung mit
// einem Literal, fmt.Sprintf; Strings lassen sich in Go nicht ändern), ein Funktionsliteral, ein
// Fehlerwert (errors.New, fmt.Errorf, apierrors) und ein Wert aus regexp.MustCompile oder
// strings.NewReplacer, die laut ihrer Doku gleichzeitig benutzt werden dürfen. Jeder andere
// Wert steht mit seinem Grund in paketwerteOhneZustand. In einer Funktion außer init weist
// niemand einer Variable der Paketebene etwas zu, auch keinem Element und keinem Feld, und
// delete, clear und copy schreiben nicht in sie; ausgenommen sind die Namen aus
// paketwerteOhneZustand, die beim ersten Gebrauch gebaut werden (sync.Once).
//
// Blindheit: Eine Methode oder Funktion, die einen erlaubten Wert ändert (eine Map der
// Paketebene als Argument), sieht die Prüfung nicht. Erzeugter Code (DO NOT EDIT) zählt nicht.
func TestPaketzustand_ProgrammcodeHaeltKeinenZustand(t *testing.T) {
	funde, gelesen := programmzustandIn(t, leseGoQuellen(t, "."), paketwerteOhneZustand)
	if gelesen < 150 {
		t.Fatalf("nur %d Variablen der Paketebene gelesen: Der Sammler greift ins Leere", gelesen)
	}
	for _, f := range funde {
		t.Errorf("%s. Was sich ändert, gehört einem Typ, den der Rufer anlegt; bleibt der Wert nach "+
			"dem Bau fest, steht er mit Grund in paketwerteOhneZustand (docs/STIL.md, Abschnitt 12)", f)
	}
}

// paketwerteOhneZustand sind Werte der Paketebene, die ein Aufruf baut und danach niemand mehr
// ändert, je mit dem Grund.
var paketwerteOhneZustand = map[string]string{
	"api/katalog/cover.go: coverHTTPClient": "ein http.Client, den alle Aufrufe lesen; er ist für " +
		"gleichzeitige Aufrufe gebaut",
	"api/tuer/eingabe.go: Pruefer": "der Prüfer der validate-Tags; seine Regeln stehen nach dem " +
		"Anlegen fest",
	"db/rechte_vorgabe.go: RechteOptional":        "Tabelle, beim Start gebaut",
	"internal/littera/barcode.go: barcodeZiffern": "Tabelle, beim Start gebaut",
	"internal/lusd/spalten.go: spalteZuKopf":      "Tabelle, beim Start gebaut",
	"internal/pgtest/pgtest.go: fehler": "eine Verbindung zur Testdatenbank je Testlauf; nur Tests " +
		"binden das Paket ein",
	"internal/pgtest/pgtest.go: once": "eine Verbindung zur Testdatenbank je Testlauf; nur Tests " +
		"binden das Paket ein",
	"internal/pgtest/pgtest.go: pool": "eine Verbindung zur Testdatenbank je Testlauf; nur Tests " +
		"binden das Paket ein",
	"internal/pgtest/pgtest.go: sperrVerbindung": "eine Verbindung zur Testdatenbank je Testlauf; " +
		"nur Tests binden das Paket ein",
	"inventur/import_idempotenz.go: importLaeuftMarker":                 "Antwort aus Literalen, einmal kodiert",
	"inventur/titel_aendern.go: sqlTitelAendern":                        "SQL aus der Liste der Felder, beim Start gebaut",
	"pdf/etikett_schueler.go: MusterSchuelerEtikett":                    "Beispiel für die Vorschau, nur gelesen",
	"pkg/lmfplan/sommerferien.go: sommerferienHessen":                   "Ferientabelle aus Literalen, beim Start gebaut",
	"pkg/lmfplan/uebrige_ferien.go: uebrigeFerienHessen":                "Ferientabelle aus Literalen, beim Start gebaut",
	"pkg/schulzeit/schulzeit.go: zone":                                  "die Zeitzone der Schule, beim ersten Gebrauch geladen",
	"pkg/schulzeit/schulzeit.go: zoneOnce":                              "die Zeitzone der Schule, beim ersten Gebrauch geladen",
	"repository/protokoll_auskunft.go: protokollAngabeJeSchluessel":     "Tabelle, beim Start gebaut",
	"repository/protokoll_auskunft.go: protokollOhneAngabeJeSchluessel": "Tabelle, beim Start gebaut",
	"repository/spur_tilgung.go: spurTilgungen": "Tabelle der Tilgungen; drei SQL-Texte bauen " +
		"Hilfsfunktionen beim Start",
}

func TestPaketzustand_ProgrammcodeSelbstprobe(t *testing.T) {
	quellen := map[string]string{
		"a/a.go": `package a

import (
	"errors"
	"regexp"
	"strings"
	"sync"

	"bibliothek/apierrors"
	"bibliothek/pkg/ratelimit"
)

var ErrX = errors.New("x")
var errY = apierrors.BadRequest("Satz.", nil)
var muster = regexp.MustCompile("a+")
var ersatz = strings.NewReplacer("a", "b")
var tabelle = map[string]int{"a": 1}
var liste = []string{"a"}
var roh = []byte("x")
var zahl = 1 << 10
var text = "a" + "b"
var bremse = ratelimit.New(1)
var mu sync.Mutex
var zeiger = &tabelle
var gebaut = func() int { return 1 }()
var erlaubt = baue()
var feld = T{A: 1}
var sql = "SELECT " + spalten()
var satz = fmt.Sprintf("%d", 1)
var schritte = []T{{F: func() {}}}
var spaeter *T

func init() { tabelle["b"] = 2 }

func lade() { spaeter = &T{} }

func f() {
	tabelle["c"] = 3
	liste = append(liste, "b")
	zahl++
	delete(tabelle, "a")
	var tabelle = map[string]int{}
	tabelle["d"] = 4
}
`,
		"a/b.go":      "package a\n\nfunc g() { text = \"c\" }\n",
		"a/a_test.go": "package a\n\nvar imTest sync.Mutex\n",
		"m/main.go":   "package main\n\nvar zaehler int\n\nfunc main() { zaehler++ }\n",
		"g/gen.go":    "// Code generated by x. DO NOT EDIT.\n\npackage g\n\nvar Info = &T{}\n",
	}
	ausnahmen := map[string]string{"a/a.go: erlaubt": "Probe", "a/a.go: spaeter": "Probe", "a/a.go: veraltet": "Probe"}
	funde, gelesen := programmzustandIn(t, quellen, ausnahmen)
	erwartet := []string{
		"a/a.go: bremse: Anfangswert ohne Zustand nicht erkennbar",
		"a/a.go: feld: Anfangswert ohne Zustand nicht erkennbar",
		"a/a.go: gebaut: Anfangswert ohne Zustand nicht erkennbar",
		"a/a.go: mu: Anfangswert ohne Zustand nicht erkennbar",
		"a/a.go: veraltet: steht in den Ausnahmen, ist aber keine mehr",
		"a/a.go: zeiger: Anfangswert ohne Zustand nicht erkennbar",
		"a/a.go:38: tabelle: Zuweisung an eine Variable der Paketebene",
		"a/a.go:39: liste: Zuweisung an eine Variable der Paketebene",
		"a/a.go:40: zahl: Zuweisung an eine Variable der Paketebene",
		"a/a.go:41: tabelle: Zuweisung an eine Variable der Paketebene",
		"a/b.go:3: text: Zuweisung an eine Variable der Paketebene",
	}
	if diff := cmp.Diff(erwartet, funde); diff != "" {
		t.Errorf("Funde (-erwartet +ist):\n%s", diff)
	}
	if gelesen != 19 {
		t.Errorf("gelesen = %d Variablen der Paketebene, erwartet 19", gelesen)
	}
}

// programmzustandIn prüft die Programmdateien außerhalb von package main und erzeugtem Code. Es
// liefert die Funde sortiert und die Zahl der gelesenen Variablen der Paketebene.
func programmzustandIn(t *testing.T, quellen map[string]string, ausnahmen map[string]string) ([]string, int) {
	t.Helper()
	fset := token.NewFileSet()
	dateien := map[string]*ast.File{}
	paketVariablen := map[string]map[string]bool{}
	for pfad, inhalt := range quellen {
		if strings.HasSuffix(pfad, "_test.go") {
			continue
		}
		datei, err := parser.ParseFile(fset, pfad, inhalt, parser.ParseComments)
		if err != nil {
			t.Fatalf("%s: %v", pfad, err)
		}
		if datei.Name.Name == "main" || istErzeugt(datei) {
			continue
		}
		dateien[pfad] = datei
		ordner := filepath.Dir(pfad)
		if paketVariablen[ordner] == nil {
			paketVariablen[ordner] = map[string]bool{}
		}
		for _, w := range paketwerte(datei) {
			for _, n := range w.Names {
				paketVariablen[ordner][n.Name] = true
			}
		}
	}

	// gebaut sind je Ordner die Namen aus den Ausnahmen; sie dürfen beim ersten Gebrauch gesetzt
	// werden.
	gebaut := map[string]bool{}
	for schluessel := range ausnahmen {
		pfad, name, _ := strings.Cut(schluessel, ": ")
		gebaut[filepath.Dir(pfad)+"/"+name] = true
	}
	var funde []string
	gelesen := 0
	ausnahmeGesehen := map[string]bool{}
	for pfad, datei := range dateien {
		for _, w := range paketwerte(datei) {
			for i, n := range w.Names {
				if n.Name == "_" {
					continue
				}
				gelesen++
				var wert ast.Expr
				if i < len(w.Values) {
					wert = w.Values[i]
				} else if len(w.Values) == 1 {
					wert = w.Values[0]
				}
				if wert != nil && istOhneZustand(wert) {
					continue
				}
				schluessel := pfad + ": " + n.Name
				if _, ok := ausnahmen[schluessel]; ok {
					ausnahmeGesehen[schluessel] = true
					continue
				}
				funde = append(funde, schluessel+": Anfangswert ohne Zustand nicht erkennbar")
			}
		}
		funde = append(funde, schreibstellen(fset, pfad, datei, paketVariablen[filepath.Dir(pfad)], gebaut)...)
	}
	for schluessel := range ausnahmen {
		if !ausnahmeGesehen[schluessel] {
			funde = append(funde, schluessel+": steht in den Ausnahmen, ist aber keine mehr")
		}
	}
	sort.Strings(funde)
	return funde, gelesen
}

// istErzeugt erkennt erzeugten Code an der Marke „DO NOT EDIT" vor der Paketzeile.
func istErzeugt(datei *ast.File) bool {
	for _, gruppe := range datei.Comments {
		if gruppe.Pos() > datei.Package {
			break
		}
		if strings.Contains(gruppe.Text(), "DO NOT EDIT") {
			return true
		}
	}
	return false
}

// paketwerte liefert die Erklärungen der Variablen auf Paketebene einer Datei.
func paketwerte(datei *ast.File) []*ast.ValueSpec {
	var werte []*ast.ValueSpec
	for _, dekl := range datei.Decls {
		if g, ok := dekl.(*ast.GenDecl); ok && g.Tok == token.VAR {
			for _, spez := range g.Specs {
				if w, ok := spez.(*ast.ValueSpec); ok {
					werte = append(werte, w)
				}
			}
		}
	}
	return werte
}

// istOhneZustand erkennt Anfangswerte, die sich nach dem Bau nicht ändern lassen oder die ihre
// Doku für gleichzeitige Benutzung freigibt.
func istOhneZustand(wert ast.Expr) bool {
	switch x := wert.(type) {
	case *ast.BasicLit, *ast.Ident, *ast.SelectorExpr, *ast.FuncLit:
		return true
	case *ast.ParenExpr:
		return istOhneZustand(x.X)
	case *ast.UnaryExpr:
		return x.Op != token.AND && istOhneZustand(x.X)
	case *ast.BinaryExpr:
		if x.Op == token.ADD && (istTextLiteral(x.X) || istTextLiteral(x.Y)) {
			return true
		}
		return istOhneZustand(x.X) && istOhneZustand(x.Y)
	case *ast.KeyValueExpr:
		return istOhneZustand(x.Key) && istOhneZustand(x.Value)
	case *ast.CompositeLit:
		switch x.Type.(type) {
		case nil, *ast.MapType, *ast.ArrayType:
		default:
			return false
		}
		for _, element := range x.Elts {
			if !istOhneZustand(element) {
				return false
			}
		}
		return true
	case *ast.CallExpr:
		if _, ok := x.Fun.(*ast.ArrayType); ok {
			return len(x.Args) == 1 && istOhneZustand(x.Args[0])
		}
		aufruf, ok := x.Fun.(*ast.SelectorExpr)
		if !ok {
			return false
		}
		paket, ok := aufruf.X.(*ast.Ident)
		if !ok {
			return false
		}
		switch paket.Name + "." + aufruf.Sel.Name {
		case "errors.New", "fmt.Errorf", "fmt.Sprintf", "regexp.MustCompile", "strings.NewReplacer":
			return true
		}
		return paket.Name == "apierrors"
	}
	return false
}

// istTextLiteral erkennt ein Text-Literal, auch als Teil einer längeren Verkettung.
func istTextLiteral(wert ast.Expr) bool {
	switch x := wert.(type) {
	case *ast.BasicLit:
		return x.Kind == token.STRING
	case *ast.ParenExpr:
		return istTextLiteral(x.X)
	case *ast.BinaryExpr:
		return x.Op == token.ADD && (istTextLiteral(x.X) || istTextLiteral(x.Y))
	}
	return false
}

// schreibstellen findet in den Funktionen einer Datei, außer in init, Zuweisungen an Variablen
// der Paketebene und delete, clear und copy auf ihnen. Namen in gebaut übergeht es.
func schreibstellen(fset *token.FileSet, pfad string, datei *ast.File, paketVariablen, gebaut map[string]bool) []string {
	eigene := map[*ast.ValueSpec]bool{}
	for _, w := range paketwerte(datei) {
		eigene[w] = true
	}
	// istPaketVariable: Ein Name dieser Datei trägt sein Objekt, ein Name aus einer anderen Datei
	// des Pakets bleibt offen.
	istPaketVariable := func(id *ast.Ident) bool {
		if id.Obj == nil {
			return paketVariablen[id.Name]
		}
		w, ok := id.Obj.Decl.(*ast.ValueSpec)
		return ok && id.Obj.Kind == ast.Var && eigene[w]
	}
	var funde []string
	pruefe := func(ziel ast.Expr) {
		for {
			switch x := ziel.(type) {
			case *ast.ParenExpr:
				ziel = x.X
				continue
			case *ast.StarExpr:
				ziel = x.X
				continue
			case *ast.IndexExpr:
				ziel = x.X
				continue
			case *ast.SelectorExpr:
				ziel = x.X
				continue
			case *ast.Ident:
				if istPaketVariable(x) && !gebaut[filepath.Dir(pfad)+"/"+x.Name] {
					funde = append(funde, pfad+":"+strconv.Itoa(fset.Position(x.Pos()).Line)+": "+x.Name+
						": Zuweisung an eine Variable der Paketebene")
				}
			}
			return
		}
	}
	for _, dekl := range datei.Decls {
		funktion, ok := dekl.(*ast.FuncDecl)
		if !ok || funktion.Body == nil || (funktion.Name.Name == "init" && funktion.Recv == nil) {
			continue
		}
		ast.Inspect(funktion.Body, func(n ast.Node) bool {
			switch s := n.(type) {
			case *ast.AssignStmt:
				if s.Tok != token.DEFINE {
					for _, ziel := range s.Lhs {
						pruefe(ziel)
					}
				}
			case *ast.IncDecStmt:
				pruefe(s.X)
			case *ast.CallExpr:
				if name, ok := s.Fun.(*ast.Ident); ok && name.Obj == nil && len(s.Args) > 0 &&
					(name.Name == "delete" || name.Name == "clear" || name.Name == "copy") {
					pruefe(s.Args[0])
				}
			}
			return true
		})
	}
	return funde
}
