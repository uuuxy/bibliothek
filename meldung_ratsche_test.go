package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"net/http"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
	"unicode"
	"unicode/utf8"

	"github.com/google/go-cmp/cmp"
)

// Prüfung: Eine Meldung der Türen ist ein Satz für die Oberfläche, deutsch und nach M3 gesetzt.
//
// Warum: Die Oberfläche zeigt das Feld error einer Antwort unverändert an, an der Anmeldung wie in
// jedem Dialog. M3 (Content design, Grammar and punctuation): „Use sentence case for all product
// text", „Omit punctuation on single-line sentences", „Use periods on: Multiple sentences", „Em
// dashes are best avoided in UX writing". Ein Fehlertext beginnt dagegen klein (docs/STIL.md,
// Abschnitt 11); er geht deshalb nie als Meldung hinaus.
//
// Regel: Ein Aufruf von apierrors mit einem Status unter 500 und festem Text ist ein Verstoß, wenn
// er den Text als Fehlertext übergibt (AntworteFehler mit errors.New oder fmt.Errorf) oder wenn die
// Meldung (AntworteFehlerMitMeldung, BadRequest, Conflict, NotFound, Unauthorized, New) klein
// beginnt, ein englisches Wort trägt, ein Wort trägt, an dem der Filter für Datenbanktexte
// anschlägt, einen Gedankenstrich hat, als ein Satz mit Punkt endet oder als mehrere Sätze ohne.
// meldungBestand nennt je Datei, wie viele Verstöße noch stehen; die Zahl kann nur sinken.
//
// Von fmt.Sprintf liest sie den Formattext; er ist der feste Satz, die Werte setzt erst die Laufzeit
// ein. Eine Textkonstante, deren Name mit meldung oder Meldung beginnt, prüft sie an ihrer
// Deklaration, gleich, auf welchem Weg sie in die Antwort kommt; ebenso den festen Text, den
// eine Funktion mit meldung oder Meldung im Namen zurückgibt, und den Wert des Schlüssels
// "message" in einer Antwort (Sätze einer erfolgreichen Antwort).
//
// Sieht nicht: Texte, die erst zur Laufzeit entstehen (err.Error(), Variablen), einen Status aus
// einer Variablen, eigene Helfer außer inventur.schreibeFehler und Antworten, die {"error": …}
// selbst schreiben.

// meldungBestand: Dateien mit Verstößen und ihre Zahl. Eine Datei ohne Verstoß fällt heraus.
var meldungBestand = map[string]int{}

// meldungEnglisch sind Wörter, die es im Deutschen nicht gibt; eines davon in einer Meldung heißt,
// sie ist englisch.
var meldungEnglisch = map[string]bool{
	"the": true, "is": true, "are": true, "not": true, "no": true, "missing": true, "invalid": true,
	"required": true, "requires": true, "found": true, "cannot": true, "must": true, "empty": true,
	"provided": true, "parameter": true, "exceeded": true, "allowed": true, "already": true,
	"please": true, "failed": true, "unknown": true, "error": true, "method": true, "search": true,
	"query": true, "payload": true, "encoded": true, "record": true, "records": true, "student": true,
	"reader": true, "order": true, "supplier": true, "user": true, "session": true, "copy": true,
	"title": true, "scope": true, "least": true, "email": true, "password": true, "rate": true,
	"graduates": true, "damage": true, "for": true, "or": true, "and": true, "of": true, "be": true,
	"with": true, "type": true,
}

// meldungSatzende erkennt das Ende eines Satzes mitten im Text: ein Wort aus zwei Zeichen, eine
// Klammer, ein schließendes Anführungszeichen oder ein Platzhalter von fmt (%s), dann . ! ? oder
// :, dann ein großer Anfang. „z. B. G" ist so kein Satzende.
var meldungSatzende = regexp.MustCompile(`(?:[\p{L}\p{N}]{2}|[)“"]|%[a-zA-Z])[.!?:]\s+[\p{Lu}„"]`)

// meldungStatus löst die Namen der Status aus net/http auf (StatusBadRequest → 400).
func meldungStatus() map[string]int {
	status := map[string]int{"StatusTeapot": http.StatusTeapot}
	for code := 100; code < 600; code++ {
		if text := http.StatusText(code); text != "" {
			name := "Status" + strings.NewReplacer(" ", "", "-", "", "'", "").Replace(text)
			status[name] = code
		}
	}
	return status
}

// meldungAufruf ist ein Aufruf von apierrors mit festem Text und bekanntem Status.
type meldungAufruf struct {
	pos         token.Position
	status      int
	text        string
	alsFehler   bool // der Text kam als errors.New oder fmt.Errorf
	gelesenFest bool
}

func TestMeldungen_SaetzeFuerDieOberflaeche(t *testing.T) {
	aufrufe := sammleMeldungen(t, ".")
	fest := 0
	gezaehlt := map[string]int{}
	var neu []string
	for _, a := range aufrufe {
		fest++
		if grund := meldungVerstoss(a); grund != "" {
			gezaehlt[a.pos.Filename]++
			if gezaehlt[a.pos.Filename] > meldungBestand[a.pos.Filename] {
				neu = append(neu, a.pos.String()+": "+grund+": "+strconv.Quote(a.text))
			}
		}
	}
	// Untergrenze: Liest der Durchlauf kaum feste Meldungen, prüft er nichts.
	if fest < 250 {
		t.Fatalf("nur %d feste Meldungen gelesen, erwartet mindestens 250 — der Detektor läuft leer", fest)
	}
	sort.Strings(neu)
	if len(neu) > 0 {
		t.Errorf("Meldung nicht als Satz für die Oberfläche:\n  %s\nFix: apierrors.AntworteFehlerMitMeldung(w, "+
			"status, \"Satz\", ursache) oder apierrors.BadRequest(\"Satz\", nil); groß beginnen, deutsch, ein Satz "+
			"ohne Punkt, mehrere Sätze mit Punkt, kein Gedankenstrich (docs/STIL.md, Abschnitt 11).",
			strings.Join(neu, "\n  "))
	}
	var veraltet []string
	for datei, zahl := range meldungBestand {
		if gezaehlt[datei] < zahl {
			veraltet = append(veraltet, datei+": "+strconv.Itoa(zahl)+" → "+strconv.Itoa(gezaehlt[datei]))
		}
	}
	sort.Strings(veraltet)
	if len(veraltet) > 0 {
		t.Errorf("meldungBestand führt mehr, als noch steht; austragen oder senken:\n  %s", strings.Join(veraltet, "\n  "))
	}
}

// Gegenprobe am Detektor: verbotene und erlaubte Formen.
func TestMeldungVerstoss_Formen(t *testing.T) {
	verboten := []meldungAufruf{
		{text: "rate limit exceeded"},
		{text: "leser nicht gefunden"},
		{text: "Leser nicht gefunden."},
		{text: "Leser nicht gefunden. Bitte neu laden"},
		{text: "Die Frist läuft noch — bitte warten"},
		{text: "Zu viele Versuche – bitte warten"},
		{text: "Eingabe enthält select * from"},
		{text: "Leser nicht gefunden", alsFehler: true},
		{text: "Zeile %d. Bitte prüfen"},
	}
	for _, a := range verboten {
		if meldungVerstoss(a) == "" {
			t.Errorf("meldungVerstoss(%q, alsFehler=%v) = \"\", erwartet ein Verstoß", a.text, a.alsFehler)
		}
	}
	erlaubt := []meldungAufruf{
		{text: "Leser nicht gefunden"},
		{text: "Der Leser liegt im Papierkorb. Erst wiederherstellen."},
		{text: "Übergabe nicht möglich: Der Bescheid ist nicht mehr offen."},
		{text: "Der Barcode muss mit G- beginnen, z. B. G-IPAD-01"},
		{text: "ID fehlt"},
		{text: "Unbekannte Art %q. Möglich sind: %s."},
		{text: "Die Sortierung „%s“. Möglich sind a und b."},
	}
	for _, a := range erlaubt {
		if grund := meldungVerstoss(a); grund != "" {
			t.Errorf("meldungVerstoss(%q) = %q, erwartet kein Verstoß", a.text, grund)
		}
	}
	if diff := cmp.Diff(400, meldungStatus()["StatusBadRequest"]); diff != "" {
		t.Errorf("meldungStatus()[StatusBadRequest] (-erwartet +ist):\n%s", diff)
	}
}

// Von fmt.Sprintf zählt der Formattext als Meldung, von fmt.Errorf und errors.New als Fehlertext.
func TestFesterText_FormattextIstMeldung(t *testing.T) {
	faelle := []struct {
		ausdruck  string
		text      string
		alsFehler bool
	}{
		{`fmt.Sprintf("Die Datei hat %d Zeilen", n)`, "Die Datei hat %d Zeilen", false},
		{`fmt.Errorf("zeile %d: %w", n, err)`, "zeile %d: %w", true},
		{`errors.New("leser fehlt")`, "leser fehlt", true},
	}
	for _, f := range faelle {
		ausdruck, err := parser.ParseExpr(f.ausdruck)
		if err != nil {
			t.Fatalf("parser.ParseExpr(%q): %v", f.ausdruck, err)
		}
		text, alsFehler, ok := festerText(ausdruck, meldungKonstanten{})
		if !ok || text != f.text || alsFehler != f.alsFehler {
			t.Errorf("festerText(%s) = %q, %v, %v, erwartet %q, %v, true", f.ausdruck, text, alsFehler, ok, f.text, f.alsFehler)
		}
	}
}

// meldungVerstoss nennt den Grund, aus dem ein Text keine Meldung nach der Regel ist, sonst "".
func meldungVerstoss(a meldungAufruf) string {
	text := a.text
	if a.alsFehler {
		return "als Fehlertext übergeben"
	}
	erstes, _ := utf8.DecodeRuneInString(text)
	if text == "" || (!unicode.IsUpper(erstes) && !unicode.IsDigit(erstes) && erstes != '„') {
		return "beginnt nicht groß"
	}
	for _, wort := range strings.FieldsFunc(strings.ToLower(text), func(r rune) bool { return !unicode.IsLetter(r) }) {
		if meldungEnglisch[wort] {
			return "englisches Wort „" + wort + "“"
		}
	}
	klein := strings.ToLower(text)
	for _, muster := range []string{"pgx", "pgconn", "sql:", "unique constraint", "foreign key", "duplicate key",
		"violates", "insert into", "select ", "update ", "delete "} {
		if strings.Contains(klein, muster) {
			return "der Filter für Datenbanktexte schlägt an („" + muster + "“)"
		}
	}
	if strings.Contains(text, "—") || strings.Contains(text, " – ") {
		return "Gedankenstrich"
	}
	mehrere := meldungSatzende.MatchString(text)
	endetMitPunkt := strings.HasSuffix(text, ".") || strings.HasSuffix(text, "!") || strings.HasSuffix(text, "?")
	switch {
	case mehrere && !endetMitPunkt:
		return "mehrere Sätze ohne Punkt am Ende"
	case !mehrere && strings.HasSuffix(text, "."):
		return "ein Satz mit Punkt am Ende"
	}
	return ""
}

// sammleMeldungen liest jeden Aufruf von apierrors mit festem Text und einem Status unter 500 im
// Programmcode unter wurzel, ohne Tests. Ein fester Text ist auch eine Konstante des Pakets oder
// eines anderen (auth.MeldungNichtAngemeldet).
func sammleMeldungen(t *testing.T, wurzel string) []meldungAufruf {
	t.Helper()
	status := meldungStatus()
	fset := token.NewFileSet()
	dateien := map[string][]*ast.File{}
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
		if !strings.HasSuffix(pfad, ".go") || strings.HasSuffix(pfad, "_test.go") {
			return nil
		}
		datei, err := parser.ParseFile(fset, pfad, nil, 0)
		if err != nil {
			return err
		}
		dateien[filepath.Dir(pfad)] = append(dateien[filepath.Dir(pfad)], datei)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	fremde := map[string]string{} // paket.Name → Text
	eigene := map[string]map[string]string{}
	for ordner, paket := range dateien {
		eigene[ordner] = textKonstanten(paket)
		for name, wert := range eigene[ordner] {
			fremde[paket[0].Name.Name+"."+name] = wert
		}
	}
	var aufrufe []meldungAufruf
	for ordner, paket := range dateien {
		konstanten := meldungKonstanten{eigene: eigene[ordner], fremde: fremde}
		for _, datei := range paket {
			aufrufe = append(aufrufe, meldungsKonstanten(fset, datei)...)
			aufrufe = append(aufrufe, meldungsRueckgaben(fset, datei, konstanten)...)
			aufrufe = append(aufrufe, meldungsSchluessel(fset, datei, konstanten)...)
			ast.Inspect(datei, func(n ast.Node) bool {
				aufruf, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				if a, ok := meldungAusAufruf(aufruf, status, konstanten); ok {
					a.pos = fset.Position(aufruf.Pos())
					aufrufe = append(aufrufe, a)
				}
				return true
			})
		}
	}
	return aufrufe
}

// meldungsKonstanten liest die Textkonstanten einer Datei, deren Name mit meldung oder Meldung
// beginnt.
func meldungsKonstanten(fset *token.FileSet, datei *ast.File) []meldungAufruf {
	var gefunden []meldungAufruf
	ast.Inspect(datei, func(n ast.Node) bool {
		decl, ok := n.(*ast.GenDecl)
		if !ok || decl.Tok != token.CONST {
			return true
		}
		for _, spec := range decl.Specs {
			werte, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			for i, name := range werte.Names {
				if !istMeldungsName(name.Name) || i >= len(werte.Values) {
					continue
				}
				if text, _, ok := festerText(werte.Values[i], meldungKonstanten{}); ok {
					gefunden = append(gefunden, meldungAufruf{pos: fset.Position(name.Pos()), text: text, gelesenFest: true})
				}
			}
		}
		return true
	})
	return gefunden
}

// meldungsRueckgaben liest den festen Text, den eine Funktion mit meldung oder Meldung im Namen
// zurückgibt; eine leere Rückgabe (kein Satz) zählt nicht.
func meldungsRueckgaben(fset *token.FileSet, datei *ast.File, konstanten meldungKonstanten) []meldungAufruf {
	var gefunden []meldungAufruf
	for _, decl := range datei.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		name := ""
		if ok {
			name = fn.Name.Name
		}
		if !ok || fn.Body == nil || (!istMeldungsName(name) && !strings.HasSuffix(name, "Meldung")) {
			continue
		}
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			rueckgabe, ok := n.(*ast.ReturnStmt)
			if !ok {
				return true
			}
			for _, wert := range rueckgabe.Results {
				if text, alsFehler, ok := festerText(wert, konstanten); ok && !alsFehler && text != "" && !istMeldungsKonstante(wert) {
					gefunden = append(gefunden, meldungAufruf{pos: fset.Position(wert.Pos()), text: text, gelesenFest: true})
				}
			}
			return true
		})
	}
	return gefunden
}

// meldungsSchluessel liest den festen Wert des Schlüssels "message" in einem Literal einer Map.
func meldungsSchluessel(fset *token.FileSet, datei *ast.File, konstanten meldungKonstanten) []meldungAufruf {
	var gefunden []meldungAufruf
	ast.Inspect(datei, func(n ast.Node) bool {
		paar, ok := n.(*ast.KeyValueExpr)
		if !ok {
			return true
		}
		if schluessel, ok := paar.Key.(*ast.BasicLit); !ok || schluessel.Value != `"message"` {
			return true
		}
		if text, alsFehler, ok := festerText(paar.Value, konstanten); ok && !alsFehler && !istMeldungsKonstante(paar.Value) {
			gefunden = append(gefunden, meldungAufruf{pos: fset.Position(paar.Value.Pos()), text: text, gelesenFest: true})
		}
		return true
	})
	return gefunden
}

// istMeldungsName sagt, ob eine Konstante nach ihrem Namen eine Meldung trägt.
func istMeldungsName(name string) bool {
	return strings.HasPrefix(name, "meldung") || strings.HasPrefix(name, "Meldung")
}

// istMeldungsKonstante erkennt den Namen einer Meldungskonstante im Aufruf; sie ist an ihrer
// Deklaration schon gezählt.
func istMeldungsKonstante(e ast.Expr) bool {
	switch v := e.(type) {
	case *ast.Ident:
		return istMeldungsName(v.Name)
	case *ast.SelectorExpr:
		return istMeldungsName(v.Sel.Name)
	}
	return false
}

// meldungKonstanten löst Namen von Textkonstanten auf: eigene des Pakets, fremde als paket.Name.
type meldungKonstanten struct {
	eigene map[string]string
	fremde map[string]string
}

// meldungAusAufruf liest Status und festen Text eines Aufrufs von apierrors; false heißt: kein
// solcher Aufruf, Status 500 und mehr, oder der Text entsteht erst zur Laufzeit.
func meldungAusAufruf(aufruf *ast.CallExpr, status map[string]int, konstanten meldungKonstanten) (meldungAufruf, bool) {
	// Der Helfer der Inventur gibt seine Meldung an AntworteFehlerMitMeldung weiter.
	if name, ok := aufruf.Fun.(*ast.Ident); ok && name.Name == "schreibeFehler" && len(aufruf.Args) == 3 {
		code := statusAus(aufruf.Args[1], status)
		text, alsFehler, ok := festerText(aufruf.Args[2], konstanten)
		if code <= 0 || code >= 500 || !ok || alsFehler || istMeldungsKonstante(aufruf.Args[2]) {
			return meldungAufruf{}, false
		}
		return meldungAufruf{status: code, text: text, gelesenFest: true}, true
	}
	sel, ok := aufruf.Fun.(*ast.SelectorExpr)
	if !ok {
		return meldungAufruf{}, false
	}
	if paket, ok := sel.X.(*ast.Ident); !ok || paket.Name != "apierrors" {
		return meldungAufruf{}, false
	}
	code, textArg := -1, ast.Expr(nil)
	switch sel.Sel.Name {
	case "AntworteFehler":
		if len(aufruf.Args) == 3 {
			code, textArg = statusAus(aufruf.Args[1], status), aufruf.Args[2]
		}
	case "AntworteFehlerMitMeldung":
		if len(aufruf.Args) == 4 {
			code, textArg = statusAus(aufruf.Args[1], status), aufruf.Args[2]
		}
	case "BadRequest":
		code, textArg = http.StatusBadRequest, aufruf.Args[0]
	case "NotFound":
		code, textArg = http.StatusNotFound, aufruf.Args[0]
	case "Unauthorized":
		code, textArg = http.StatusUnauthorized, aufruf.Args[0]
	case "Conflict":
		code, textArg = http.StatusConflict, aufruf.Args[0]
	case "New":
		if len(aufruf.Args) == 3 {
			code, textArg = statusAus(aufruf.Args[0], status), aufruf.Args[1]
		}
	}
	if code <= 0 || code >= 500 || textArg == nil || istMeldungsKonstante(textArg) {
		return meldungAufruf{}, false
	}
	text, alsFehler, ok := festerText(textArg, konstanten)
	if !ok || (sel.Sel.Name != "AntworteFehler" && alsFehler) {
		return meldungAufruf{}, false
	}
	if sel.Sel.Name == "AntworteFehler" && !alsFehler {
		return meldungAufruf{}, false
	}
	if text == "" && sel.Sel.Name == "NotFound" {
		return meldungAufruf{}, false // die Vorgabe von apierrors.NotFound
	}
	return meldungAufruf{status: code, text: text, alsFehler: alsFehler, gelesenFest: true}, true
}

// statusAus löst einen Status als Name aus net/http oder als Zahl auf, sonst -1.
func statusAus(e ast.Expr, status map[string]int) int {
	switch v := e.(type) {
	case *ast.SelectorExpr:
		if code, ok := status[v.Sel.Name]; ok {
			return code
		}
	case *ast.BasicLit:
		if code, err := strconv.Atoi(v.Value); err == nil {
			return code
		}
	}
	return -1
}

// festerText liefert den Text eines Literals, einer Textkonstante, einer Verkettung daraus oder
// des ersten Arguments von errors.New, fmt.Errorf und fmt.Sprintf; alsFehler sagt, dass er als
// Fehler kam.
func festerText(e ast.Expr, konstanten meldungKonstanten) (text string, alsFehler, ok bool) {
	switch v := e.(type) {
	case *ast.BasicLit:
		if v.Kind == token.STRING {
			s, err := strconv.Unquote(v.Value)
			return s, false, err == nil
		}
	case *ast.Ident:
		s, ok := konstanten.eigene[v.Name]
		return s, false, ok
	case *ast.SelectorExpr:
		if paket, ok := v.X.(*ast.Ident); ok {
			s, ok := konstanten.fremde[paket.Name+"."+v.Sel.Name]
			return s, false, ok
		}
	case *ast.BinaryExpr:
		links, _, ok1 := festerText(v.X, konstanten)
		rechts, _, ok2 := festerText(v.Y, konstanten)
		return links + rechts, false, ok1 && ok2 && v.Op == token.ADD
	case *ast.CallExpr:
		sel, ok := v.Fun.(*ast.SelectorExpr)
		if !ok || len(v.Args) == 0 {
			return "", false, false
		}
		paket, ok := sel.X.(*ast.Ident)
		if !ok {
			return "", false, false
		}
		istNew := paket.Name == "errors" && sel.Sel.Name == "New"
		istErrorf := paket.Name == "fmt" && sel.Sel.Name == "Errorf"
		istSprintf := paket.Name == "fmt" && sel.Sel.Name == "Sprintf"
		if !istNew && !istErrorf && !istSprintf {
			return "", false, false
		}
		s, _, ok := festerText(v.Args[0], konstanten)
		return s, !istSprintf, ok
	}
	return "", false, false
}
