package main

// Gate: Eine Kennung, die als UUID in die Datenbank geht, wird an der Tür geprüft.
//
// Anlass (ZAP-Lauf 13.09.2026, am Stack nachgestellt): `?session_id=x` am Fehlbestand,
// `?titel_id=x` an den Vormerkungen, `{"session_id":"x"}` an Inventur-Abschluss und
// -Abbruch — jedes Mal ging der Text ungeprüft an Postgres, und `invalid input syntax for
// type uuid` (SQLSTATE 22P02) kam als 500 zurück. `ValidateUUIDParamsMiddleware` prüft
// nur Pfad-Parameter.
//
// Warum nicht zentral „22P02 → 400" in apierrors: Das wäre die Bugklasse Fehler-Kollaps
// (fehler_kollaps_test.go) — ein kaputter Wert, den der SERVER erzeugt, sähe dann aus wie
// ein Bedienfehler. Die Prüfung gehört an die Stelle, die weiß, dass hier eine Eingabe
// ankommt.
//
// Regel: In jedem Struct, das ein Handler aus dem Request-Body liest (DecodeAndValidate
// oder json.NewDecoder(...).Decode), trägt jedes Feld mit JSON-Namen auf „id"/„ids" vom
// Typ string, *string oder []string einen validate-Tag mit „uuid" — oder steht mit Grund
// in uuidEingabenAusnahmen. Liest ein Handler direkt per json.NewDecoder, muss er außerdem
// selbst prüfen: uuid.Parse/uuid.Validate oder Validate.Struct. Dasselbe für
// `Query().Get("…id")` mit festem Namen.
//
// Reparatur bei Rot: `validate:"omitempty,uuid_oder_leer"` am Feld (api/http_utils.go),
// für Query-Parameter uuidAusQuery; ist die Kennung keine UUID (Barcode, LUSD-ID),
// Eintrag in uuidEingabenAusnahmen mit der Spalte, die das belegt.

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
)

// uuidEingabenAusnahmen: „paket.Typ.Feld" bzw. „paket:Funktion:query:name" /
// „paket:Funktion:ohne Validierung" → warum es so richtig ist. Belegt am 13.09.2026 an
// information_schema.columns: alle Barcode-Spalten und schueler.lusd_id sind VARCHAR.
var uuidEingabenAusnahmen = map[string]string{
	"api.CreateStudentRequest.BarcodeID":      "schueler.barcode_id ist VARCHAR (Ausweis-Barcode)",
	"api.patchStudentRequest.BarcodeID":       "schueler.barcode_id ist VARCHAR (Ausweis-Barcode)",
	"api.patchStudentRequest.LusdID":          "schueler.lusd_id ist VARCHAR (LUSD-Schlüssel)",
	"api.CreateUserRequest.BarcodeID":         "benutzer.barcode_id ist VARCHAR",
	"api.UpdateUserRequest.BarcodeID":         "benutzer.barcode_id ist VARCHAR",
	"api.GeraetRequest.BarcodeID":             "geraete.barcode_id ist VARCHAR",
	"api.InventurScanRequest.BarcodeID":       "gescannter Exemplar-Barcode, buecher_exemplare.barcode_id ist VARCHAR",
	"api.PrintLabelsRequest.FormatID":         "Formatschlüssel aus api/label_formats.go (zweckform_l4760), keine Spalte",
	"api.SchuelerEtikettenRequest.FormatID":   "Formatschlüssel aus api/label_formats.go (zweckform_l4760), keine Spalte",
	"api.EtikettenGedrucktRequest.BarcodeIDs": "Exemplar-Barcodes, gehen in buecher_exemplare.barcode_id (VARCHAR)",
	// Der Stapel ist ein Slice und läuft nicht durch DecodeAndValidate; geprüft wird je
	// Eintrag in processSingleBatchItem (Validate.Struct), das dieser Detektor nicht sieht.
	"api:ActionBatchHandler:ohne Validierung": "Validate.Struct je Eintrag in processSingleBatchItem",
}

type uuidPaket struct {
	typen   map[string]ast.Expr
	dateien []*ast.File
}

func ladeUUIDPakete(t *testing.T, wurzeln ...string) map[string]*uuidPaket {
	t.Helper()
	pakete := map[string]*uuidPaket{}
	fset := token.NewFileSet()
	for _, wurzel := range wurzeln {
		err := filepath.WalkDir(wurzel, func(pfad string, e fs.DirEntry, err error) error {
			if err != nil || e.IsDir() || !strings.HasSuffix(pfad, ".go") || strings.HasSuffix(pfad, "_test.go") {
				return err
			}
			datei, err := parser.ParseFile(fset, pfad, nil, 0)
			if err != nil {
				return err
			}
			verz := filepath.Dir(pfad)
			if pakete[verz] == nil {
				pakete[verz] = &uuidPaket{typen: map[string]ast.Expr{}}
			}
			nimmUUIDDatei(pakete[verz], datei)
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	return pakete
}

func nimmUUIDDatei(p *uuidPaket, datei *ast.File) {
	p.dateien = append(p.dateien, datei)
	ast.Inspect(datei, func(n ast.Node) bool {
		if ts, ok := n.(*ast.TypeSpec); ok {
			p.typen[ts.Name.Name] = ts.Type
		}
		return true
	})
}

// pruefeUUIDEingaben liefert jede Stelle, an der eine Kennung ungeprüft hereinkommt.
func pruefeUUIDEingaben(pakete map[string]*uuidPaket) (maengel []string, gesehen map[string]bool) {
	gesehen = map[string]bool{}
	melde := func(schluessel, ort string) {
		if gesehen[schluessel] {
			return
		}
		gesehen[schluessel] = true
		if _, ok := uuidEingabenAusnahmen[schluessel]; !ok {
			maengel = append(maengel, schluessel+" ("+ort+")")
		}
	}
	for verz, p := range pakete {
		paket := filepath.Base(verz)
		for _, datei := range p.dateien {
			for _, decl := range datei.Decls {
				if fn, ok := decl.(*ast.FuncDecl); ok && fn.Body != nil {
					pruefeUUIDFunktion(p, paket, fn, melde)
				}
			}
		}
	}
	sort.Strings(maengel)
	return maengel, gesehen
}

func pruefeUUIDFunktion(p *uuidPaket, paket string, fn *ast.FuncDecl, melde func(schluessel, ort string)) {
	typen := lokaleVariablen(fn.Body)
	uuidSichtbar := ruftAuf(fn.Body, "uuid", "Parse", "Validate") || ruftHelferAuf(fn.Body, "alleUUIDs")
	validateSichtbar := ruftAuf(fn.Body, "Validate", "Struct", "Var")
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		if name, ok := queryIDName(call); ok {
			melde(fmt.Sprintf("%s:%s:query:%s", paket, fn.Name.Name, name), fn.Name.Name)
		}
		ziel, direkt := dekodierZiel(call)
		if ziel == "" || typen[ziel] == nil || (direkt && uuidSichtbar) {
			return true
		}
		kennungen := 0
		pruefeStructFelder(p, paket, fn.Name.Name, typen[ziel], map[string]bool{}, melde, &kennungen)
		if direkt && !validateSichtbar && kennungen > 0 {
			melde(fmt.Sprintf("%s:%s:ohne Validierung", paket, fn.Name.Name), fn.Name.Name)
		}
		return true
	})
}

func lokaleVariablen(body *ast.BlockStmt) map[string]ast.Expr {
	typen := map[string]ast.Expr{}
	ast.Inspect(body, func(n ast.Node) bool {
		if vs, ok := n.(*ast.ValueSpec); ok && vs.Type != nil {
			for _, name := range vs.Names {
				typen[name.Name] = vs.Type
			}
		}
		return true
	})
	return typen
}

// ruftAuf: irgendwo im Rumpf steht paket.Name(…) mit einem der Namen.
func ruftAuf(body *ast.BlockStmt, paket string, namen ...string) bool {
	gefunden := false
	ast.Inspect(body, func(n ast.Node) bool {
		if sel, ok := n.(*ast.SelectorExpr); ok {
			if id, ok := sel.X.(*ast.Ident); ok && id.Name == paket {
				for _, name := range namen {
					gefunden = gefunden || sel.Sel.Name == name
				}
			}
		}
		return !gefunden
	})
	return gefunden
}

// ruftHelferAuf: irgendwo im Rumpf steht name(…) — ein Prüf-Helfer im selben Paket
// (inventur.alleUUIDs).
func ruftHelferAuf(body *ast.BlockStmt, name string) bool {
	gefunden := false
	ast.Inspect(body, func(n ast.Node) bool {
		if c, ok := n.(*ast.CallExpr); ok {
			if id, ok := c.Fun.(*ast.Ident); ok && id.Name == name {
				gefunden = true
			}
		}
		return !gefunden
	})
	return gefunden
}

// dekodierZiel: Name der Variablen hinter &x in DecodeAndValidate(w, r, &x) oder
// json.NewDecoder(...).Decode(&x); direkt=true für den zweiten Weg.
func dekodierZiel(call *ast.CallExpr) (string, bool) {
	var arg ast.Expr
	direkt := false
	switch f := call.Fun.(type) {
	case *ast.Ident:
		if (f.Name == "DecodeAndValidate" || f.Name == "DecodeStrictAndValidate") && len(call.Args) == 3 {
			arg = call.Args[2]
		}
	case *ast.SelectorExpr:
		if inner, ok := f.X.(*ast.CallExpr); ok && f.Sel.Name == "Decode" && len(call.Args) == 1 {
			if sel, ok := inner.Fun.(*ast.SelectorExpr); ok && sel.Sel.Name == "NewDecoder" {
				arg, direkt = call.Args[0], true
			}
		}
	}
	if u, ok := arg.(*ast.UnaryExpr); ok && u.Op == token.AND {
		if id, ok := u.X.(*ast.Ident); ok {
			return id.Name, direkt
		}
	}
	return "", false
}

// queryIDName: `….Query().Get("…id")` mit festem Namen.
func queryIDName(call *ast.CallExpr) (string, bool) {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "Get" || len(call.Args) != 1 {
		return "", false
	}
	inner, ok := sel.X.(*ast.CallExpr)
	if !ok {
		return "", false
	}
	if q, ok := inner.Fun.(*ast.SelectorExpr); !ok || q.Sel.Name != "Query" {
		return "", false
	}
	lit, ok := call.Args[0].(*ast.BasicLit)
	if !ok {
		return "", false
	}
	name := strings.Trim(lit.Value, `"`)
	return name, istKennungsName(name)
}

func istKennungsName(name string) bool {
	n := strings.ToLower(name)
	return strings.HasSuffix(n, "id") || strings.HasSuffix(n, "ids")
}

func pruefeStructFelder(p *uuidPaket, paket, funktion string, typ ast.Expr, besucht map[string]bool,
	melde func(schluessel, ort string), kennungen *int) {
	name := funktion
	var st *ast.StructType
	switch k := typ.(type) {
	case *ast.StructType:
		st = k
	case *ast.Ident:
		if besucht[k.Name] || p.typen[k.Name] == nil {
			return
		}
		besucht[k.Name] = true
		if s, ok := p.typen[k.Name].(*ast.StructType); ok {
			st, name = s, k.Name
		} else {
			pruefeStructFelder(p, paket, funktion, p.typen[k.Name], besucht, melde, kennungen)
			return
		}
	case *ast.ArrayType:
		pruefeStructFelder(p, paket, funktion, k.Elt, besucht, melde, kennungen)
		return
	case *ast.StarExpr:
		pruefeStructFelder(p, paket, funktion, k.X, besucht, melde, kennungen)
		return
	}
	if st == nil {
		return
	}
	for _, feld := range st.Fields.List {
		tag := reflect.StructTag("")
		if feld.Tag != nil {
			tag = reflect.StructTag(strings.Trim(feld.Tag.Value, "`"))
		}
		jsonName := strings.Split(tag.Get("json"), ",")[0]
		if istTextKennung(feld.Type) && istKennungsName(jsonName) {
			for _, n := range feld.Names {
				schluessel := fmt.Sprintf("%s.%s.%s", paket, name, n.Name)
				if _, ausnahme := uuidEingabenAusnahmen[schluessel]; !ausnahme {
					*kennungen++
				}
				if !strings.Contains(tag.Get("validate"), "uuid") {
					melde(schluessel, funktion)
				}
			}
		}
		pruefeStructFelder(p, paket, funktion, feld.Type, besucht, melde, kennungen)
	}
}

func istTextKennung(typ ast.Expr) bool {
	switch k := typ.(type) {
	case *ast.Ident:
		return k.Name == "string"
	case *ast.StarExpr:
		return istTextKennung(k.X)
	case *ast.ArrayType:
		return istTextKennung(k.Elt)
	}
	return false
}

func TestUUIDEingabenWerdenAnDerTuerGeprueft(t *testing.T) {
	maengel, gesehen := pruefeUUIDEingaben(ladeUUIDPakete(t, "api", "inventur"))
	if len(gesehen) < 5 {
		t.Fatalf("nur %d Kennungs-Eingaben gefunden — der Sammler greift vermutlich ins Leere", len(gesehen))
	}
	if len(maengel) > 0 {
		t.Errorf("Kennung kommt ungeprüft herein und endet bei einem Nicht-UUID-Wert als 500 (22P02):\n  %s\n"+
			"Fix: validate-Tag mit uuid_oder_leer bzw. uuidAusQuery — oder Eintrag in uuidEingabenAusnahmen mit Grund.",
			strings.Join(maengel, "\n  "))
	}
	for schluessel := range uuidEingabenAusnahmen {
		if !gesehen[schluessel] {
			t.Errorf("%s steht in uuidEingabenAusnahmen, kommt im Baum aber nicht mehr vor — streichen.", schluessel)
		}
	}
}

// Gegenprobe am Detektor: ein Sammler, der nichts findet, meldet ewig „alles gut".
func TestUUIDEingabenDetektorErkenntDieFormen(t *testing.T) {
	quelle := `package p
type ohneTag struct { SessionID string ` + "`json:\"session_id\"`" + ` }
type mitTag struct { SessionID string ` + "`json:\"session_id\" validate:\"omitempty,uuid_oder_leer\"`" + ` }
type liste struct { IDs []string ` + "`json:\"exemplar_ids\"`" + ` }
type kein struct { Name string ` + "`json:\"name\"`" + ` }
func aTag(w, r any) { var req ohneTag; DecodeAndValidate(w, r, &req) }
func bSauber(w, r any) { var req mitTag; DecodeAndValidate(w, r, &req) }
func cQuery(r any) { _ = r.URL.Query().Get("titel_id") }
func dDirekt(r any) { var req mitTag; json.NewDecoder(r.Body).Decode(&req) }
func eDirektGeprueft(r any) { var req liste; json.NewDecoder(r.Body).Decode(&req); uuid.Validate(req.IDs[0]) }
func fListe(w, r any) { var req liste; DecodeAndValidate(w, r, &req) }
func hHelfer(r any) { var req liste; json.NewDecoder(r.Body).Decode(&req); alleUUIDs(req.IDs) }
func gKeine(w, r any) { var req kein; DecodeAndValidate(w, r, &req) }
`
	datei, err := parser.ParseFile(token.NewFileSet(), "p/probe.go", quelle, 0)
	if err != nil {
		t.Fatal(err)
	}
	p := &uuidPaket{typen: map[string]ast.Expr{}}
	nimmUUIDDatei(p, datei)
	maengel, _ := pruefeUUIDEingaben(map[string]*uuidPaket{"p": p})
	erwartet := []string{
		"p.liste.IDs (fListe)",
		"p.ohneTag.SessionID (aTag)",
		"p:cQuery:query:titel_id (cQuery)",
		"p:dDirekt:ohne Validierung (dDirekt)",
	}
	if strings.Join(maengel, "|") != strings.Join(erwartet, "|") {
		t.Errorf("Detektor meldet\n  %s\nerwartet\n  %s", strings.Join(maengel, "\n  "), strings.Join(erwartet, "\n  "))
	}
}
