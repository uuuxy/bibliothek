package main

import (
	"go/ast"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"golang.org/x/tools/go/packages"
)

// Groß geschrieben ist nur, was ein anderes Paket beim Namen nennt, auch in seinen Tests, und
// der Typ in der Signatur eines solchen Namens (STIL.md 6, nach Best Practices: „Every exported
// type increases the cognitive load for the reader."). Gemessen mit go/types über alle Pakete
// samt Tests.
//
// Ausgenommen sind Methoden, die eine Schnittstelle verlangt (auch die der Standardbibliothek),
// Unwrap, Is, As und Meldung der Fehlertypen, die Türen (STIL.md 5), Konstanten eines Blocks, von
// dem ein Glied draußen genutzt wird (eine Aufzählung ist als Ganzes öffentlich), und das
// erzeugte docs/docs.go. Blindheit: Aufrufe über Vorlagen und Reflexion und Schnittstellen, die
// erst in einer Funktion erklärt sind.
func TestSichtbarkeit_OeffentlichNurMitFremdemNutzer(t *testing.T) {
	pakete := ladePaketeMitTests(t)
	m := newSichtbarkeitsMessung(t)
	besuche(pakete, m.sammle)
	besuche(pakete, m.markiereSignaturen)
	besuche(pakete, m.markiereSchnittstellenMethoden)
	besuche(pakete, m.markiereAufzaehlungen)

	// Gegenprobe am Detektor: Ohne Typinformation fände er keine Definitionen und keine Nutzer.
	if len(m.definiert) < 1500 || len(m.fremdGenutzt) < 1000 {
		t.Fatalf("nur %d öffentliche Namen und %d fremd genutzte gefunden: Der Detektor greift ins Leere",
			len(m.definiert), len(m.fremdGenutzt))
	}
	var funde []string
	for name, stelle := range m.definiert {
		if !m.bleibtOeffentlich(name) {
			funde = append(funde, stelle+": "+name[strings.LastIndex(name, "/")+1:])
		}
	}
	sort.Strings(funde)
	for _, fund := range funde {
		t.Errorf("%s ist öffentlich, aber kein anderes Paket nennt es. Klein schreiben.", fund)
	}
}

// ladePaketeMitTests lädt alle Pakete des Moduls mit Syntax und Typinformation, je auch in der
// Fassung mit Tests.
func ladePaketeMitTests(t *testing.T) []*packages.Package {
	t.Helper()
	konfig := &packages.Config{
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedImports | packages.NeedDeps |
			packages.NeedTypes | packages.NeedTypesInfo | packages.NeedSyntax,
		Tests: true,
	}
	pakete, err := packages.Load(konfig, "./...")
	if err != nil {
		t.Fatalf("Pakete laden: %v", err)
	}
	if n := packages.PrintErrors(pakete); n > 0 {
		t.Fatalf("%d Fehler beim Laden der Pakete: Die Messung wäre unvollständig", n)
	}
	return pakete
}

// besuche ruft je Paket, auch je Abhängigkeit, einmal auf.
func besuche(pakete []*packages.Package, f func(*packages.Package)) {
	gesehen := map[string]bool{}
	packages.Visit(pakete, nil, func(p *packages.Package) {
		if p.Types == nil || gesehen[p.ID] {
			return
		}
		gesehen[p.ID] = true
		f(p)
	})
}

// sichtbarkeitsMessung hält, was über alle Pakete gesammelt wird; Schlüssel ist der volle Name,
// bei Methoden mit dem Typ (bibliothek/pdf.SchuleInfo.OrtDatum).
type sichtbarkeitsMessung struct {
	wurzel            string
	definiert         map[string]string // öffentlicher Name → Stelle der Definition
	fremdGenutzt      map[string]bool
	inSignatur        map[string]bool
	fuerSchnittstelle map[string]bool
	schnittstellen    []*types.Interface
}

func newSichtbarkeitsMessung(t *testing.T) *sichtbarkeitsMessung {
	t.Helper()
	wurzel, err := os.Getwd()
	if err != nil {
		t.Fatalf("Arbeitsverzeichnis: %v", err)
	}
	fehler, ok := types.Universe.Lookup("error").Type().Underlying().(*types.Interface)
	if !ok {
		t.Fatal("error ist keine Schnittstelle")
	}
	return &sichtbarkeitsMessung{
		wurzel:            filepath.ToSlash(wurzel),
		definiert:         map[string]string{},
		fremdGenutzt:      map[string]bool{},
		inSignatur:        map[string]bool{},
		fuerSchnittstelle: map[string]bool{},
		schnittstellen:    []*types.Interface{fehler},
	}
}

// istModul unterscheidet die eigenen Pakete von fremden; ein Go-Paket, das eine Bibliothek der
// Oberfläche in node_modules mitbringt, ist fremd.
func istModul(p *packages.Package) bool {
	return (p.PkgPath == "bibliothek" || strings.HasPrefix(p.PkgPath, "bibliothek/")) &&
		!strings.Contains(p.PkgPath, "/node_modules/")
}

// sammle trägt die öffentlichen Namen der Programmdateien ein, die Nutzer aus anderen Paketen und
// die Schnittstellen aller Pakete, auch der fremden.
func (m *sichtbarkeitsMessung) sammle(p *packages.Package) {
	umfang := p.Types.Scope()
	for _, name := range umfang.Names() {
		if typname, ok := umfang.Lookup(name).(*types.TypeName); ok {
			if schnittstelle, ok := typname.Type().Underlying().(*types.Interface); ok && schnittstelle.NumMethods() > 0 {
				m.schnittstellen = append(m.schnittstellen, schnittstelle)
			}
		}
	}
	if !istModul(p) {
		return
	}
	for bezeichner, objekt := range p.TypesInfo.Defs {
		if objekt == nil || !objekt.Exported() || !zaehltAlsOeffentlich(objekt) {
			continue
		}
		stelle := p.Fset.Position(bezeichner.Pos())
		if strings.HasSuffix(stelle.Filename, "_test.go") || strings.HasSuffix(stelle.Filename, "docs/docs.go") {
			continue
		}
		if name := vollerName(objekt); name != "" {
			m.definiert[name] = strings.TrimPrefix(filepath.ToSlash(stelle.Filename), m.wurzel+"/") + ":" + strconv.Itoa(stelle.Line)
		}
	}
	for _, objekt := range p.TypesInfo.Uses {
		m.merkeNutzer(p, objekt)
	}
	for _, auswahl := range p.TypesInfo.Selections {
		m.merkeNutzer(p, auswahl.Obj())
	}
}

// merkeNutzer trägt einen Namen ein, den p nennt und ein anderes Paket erklärt. Das externe
// Testpaket (x_test) zählt als anderes Paket: Es sieht nur, was groß geschrieben ist.
func (m *sichtbarkeitsMessung) merkeNutzer(p *packages.Package, objekt types.Object) {
	if objekt == nil || objekt.Pkg() == nil || !objekt.Exported() {
		return
	}
	herkunft := objekt.Pkg().Path()
	if strings.TrimSuffix(herkunft, "_test") != strings.TrimSuffix(p.PkgPath, "_test") ||
		strings.HasSuffix(p.PkgPath, "_test") && !strings.HasSuffix(herkunft, "_test") {
		m.fremdGenutzt[vollerName(objekt)] = true
	}
}

// markiereSignaturen hält die Typen öffentlich, die ein fremd genutzter Name annimmt oder
// liefert: Wer das Ergebnis in der Hand hält, muss seinen Typ nachschlagen können.
func (m *sichtbarkeitsMessung) markiereSignaturen(p *packages.Package) {
	if !istModul(p) {
		return
	}
	for _, objekt := range p.TypesInfo.Defs {
		if objekt != nil && m.fremdGenutzt[vollerName(objekt)] {
			m.markiereTyp(objekt.Type(), 0)
		}
	}
}

func (m *sichtbarkeitsMessung) markiereTyp(typ types.Type, tiefe int) {
	if typ == nil || tiefe > 6 {
		return
	}
	switch t := typ.(type) {
	case *types.Alias:
		// Ein Alias steht unter seinem eigenen Namen in der Signatur (auth.Datenbank in
		// NewAnmeldung); wer sie liest, schlägt ihn nach.
		if name := vollerName(t.Obj()); !m.inSignatur[name] {
			m.inSignatur[name] = true
			m.markiereTyp(t.Rhs(), tiefe+1)
		}
	case *types.Named:
		name := vollerName(t.Obj())
		if m.inSignatur[name] {
			return
		}
		m.inSignatur[name] = true
		if struktur, ok := t.Underlying().(*types.Struct); ok {
			for i := range struktur.NumFields() {
				if struktur.Field(i).Exported() {
					m.markiereTyp(struktur.Field(i).Type(), tiefe+1)
				}
			}
		}
		for i := range t.TypeArgs().Len() {
			m.markiereTyp(t.TypeArgs().At(i), tiefe+1)
		}
	case *types.Pointer:
		m.markiereTyp(t.Elem(), tiefe+1)
	case *types.Slice:
		m.markiereTyp(t.Elem(), tiefe+1)
	case *types.Array:
		m.markiereTyp(t.Elem(), tiefe+1)
	case *types.Map:
		m.markiereTyp(t.Key(), tiefe+1)
		m.markiereTyp(t.Elem(), tiefe+1)
	case *types.Chan:
		m.markiereTyp(t.Elem(), tiefe+1)
	case *types.Signature:
		for i := range t.Params().Len() {
			m.markiereTyp(t.Params().At(i).Type(), tiefe+1)
		}
		for i := range t.Results().Len() {
			m.markiereTyp(t.Results().At(i).Type(), tiefe+1)
		}
	}
}

// markiereSchnittstellenMethoden nimmt die Methoden eines Typs aus, die eine Schnittstelle
// verlangt, etwa Error, String oder ServeHTTP.
func (m *sichtbarkeitsMessung) markiereSchnittstellenMethoden(p *packages.Package) {
	if !istModul(p) {
		return
	}
	umfang := p.Types.Scope()
	for _, name := range umfang.Names() {
		typname, ok := umfang.Lookup(name).(*types.TypeName)
		if !ok {
			continue
		}
		benannt, ok := typname.Type().(*types.Named)
		if !ok {
			continue
		}
		for _, schnittstelle := range m.schnittstellen {
			if types.Implements(benannt, schnittstelle) || types.Implements(types.NewPointer(benannt), schnittstelle) {
				for i := range schnittstelle.NumMethods() {
					m.fuerSchnittstelle[p.PkgPath+"."+benannt.Obj().Name()+"."+schnittstelle.Method(i).Name()] = true
				}
			}
		}
	}
}

// markiereAufzaehlungen hält einen Konstantenblock als Ganzes öffentlich, sobald ein Glied oder
// sein Typ draußen genutzt wird (STIL.md 6).
func (m *sichtbarkeitsMessung) markiereAufzaehlungen(p *packages.Package) {
	if !istModul(p) {
		return
	}
	for _, datei := range p.Syntax {
		for _, dekl := range datei.Decls {
			block, ok := dekl.(*ast.GenDecl)
			if !ok || block.Tok != token.CONST {
				continue
			}
			var namen []string
			offen := false
			for _, spez := range block.Specs {
				werte, ok := spez.(*ast.ValueSpec)
				if !ok {
					continue
				}
				for _, bezeichner := range werte.Names {
					objekt := p.TypesInfo.Defs[bezeichner]
					if objekt == nil || !objekt.Exported() {
						continue
					}
					namen = append(namen, vollerName(objekt))
					offen = offen || m.fremdGenutzt[vollerName(objekt)]
					if benannt, ok := objekt.Type().(*types.Named); ok {
						typ := vollerName(benannt.Obj())
						offen = offen || m.fremdGenutzt[typ] || m.inSignatur[typ]
					}
				}
			}
			if offen {
				for _, name := range namen {
					m.fremdGenutzt[name] = true
				}
			}
		}
	}
}

func (m *sichtbarkeitsMessung) bleibtOeffentlich(name string) bool {
	if m.fremdGenutzt[name] || m.inSignatur[name] || m.fuerSchnittstelle[name] {
		return true
	}
	// Türen sind nach STIL.md 5 öffentlich; Unwrap, Is, As und Meldung sind Methoden, nach denen
	// errors und apierrors fragen.
	for _, endung := range []string{".Unwrap", ".Is", ".As", ".Meldung"} {
		if strings.HasSuffix(name, endung) {
			return true
		}
	}
	return strings.Contains(name, ".Tueren.") || strings.HasSuffix(name, ".main")
}

// zaehltAlsOeffentlich nimmt aus, was keinen Namen der Paketebene trägt: Felder, Typparameter
// und Namen, die eine Funktion für sich erklärt.
func zaehltAlsOeffentlich(objekt types.Object) bool {
	switch o := objekt.(type) {
	case *types.Func:
		return true
	case *types.TypeName:
		_, istParameter := o.Type().(*types.TypeParam)
		return !istParameter && o.Parent() == o.Pkg().Scope()
	case *types.Const:
		return o.Parent() == o.Pkg().Scope()
	case *types.Var:
		return !o.IsField() && o.Parent() == o.Pkg().Scope()
	}
	return false
}

// vollerName nennt Paket und Namen, bei einer Methode auch den Typ; leer für Methoden, deren
// Empfänger kein benannter Typ ist.
func vollerName(objekt types.Object) string {
	if objekt.Pkg() == nil {
		return ""
	}
	if funktion, ok := objekt.(*types.Func); ok {
		if empfaenger := funktion.Signature().Recv(); empfaenger != nil {
			typ := empfaenger.Type()
			if zeiger, ok := typ.(*types.Pointer); ok {
				typ = zeiger.Elem()
			}
			benannt, ok := typ.(*types.Named)
			if !ok {
				return ""
			}
			return objekt.Pkg().Path() + "." + benannt.Obj().Name() + "." + objekt.Name()
		}
	}
	return objekt.Pkg().Path() + "." + objekt.Name()
}
