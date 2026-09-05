package main

// Bugklasse „Parameter-Struktur mit vergessenem Feld" (Rasterdurchgang 05.09.2026 abends,
// Frage 2 „Spezialwert"): Zehn Jules-PRs formten lange Parameterlisten in Strukturen um
// (1359408b). Der Gewinn ist echt — benachbarte Argumente desselben Typs lassen sich nicht
// mehr stumm vertauschen. Der Preis steht in derselben Commit-Botschaft: Aus einem
// Pflichtargument wird ein Feld, und ein VERGESSENES Feld ist kein Compilerfehler, sondern
// der Nullwert. Bei DueDateOptions wären das 0 Tage Frist, bei den Zusammenführ-Parametern
// eine Rückweg-Spur ohne Bezug.
//
// Diese Ratsche verlangt für genau diese Strukturen, dass JEDES Literal JEDES Feld setzt.
// Sie gilt bewusst nicht für alle Strukturen des Baums: Ein Options-Objekt mit
// Vorgabewerten darf teilbefüllt sein. Wer eine neue Parameter-Struktur einführt, trägt
// sie hier ein — die Liste ist die Aussage, nicht der Zufall.
//
// Positionale Literale (`coverBox{x, y, b, h}`) zwingt der Compiler bereits zur
// Vollständigkeit; sie werden deshalb nicht bemängelt.

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// parameterStrukturen: Typname → warum ein fehlendes Feld hier schadet.
var parameterStrukturen = map[string]string{
	"DueDateOptions":                       "fehlendes Feld = 0 Tage Frist und damit eine falsche Mahnung",
	"BarcodePruefOptionen":                 "fehlendes Feld = leere Ausschluss-ID oder leere Meldung",
	"BulkReceiveParams":                    "fehlendes Feld = Wareneingang ohne Bearbeiter oder ohne IP im Protokoll",
	"Kuerzung":                             "fehlendes Feld = Übernahme-Protokoll ohne Quelle, Kennung oder Wert",
	"rueckwegEintragParams":                "fehlendes Feld = Rückweg-Spur des Zusammenführens ohne Bezug",
	"schreibeZusammengefuehrtesZielParams": "fehlendes Feld = Zielsatz ohne Herkunft",
	"titelLookup":                          "fehlende Map = Import ordnet Exemplare keinem oder dem falschen Titel zu",
	"klassifizierungsLauf":                 "fehlendes Feld = LUSD-Klassifizierung ohne Ergebnis- oder Gesehen-Liste",
	"bestellBerichtOpts":                   "fehlendes Feld = Bericht über den falschen Zeitraum oder ohne Preise",
	"coverBox":                             "fehlendes Maß = Cover an Position 0 oder mit Größe 0",
}

func TestParameterStrukturen_JedesLiteralSetztJedesFeld(t *testing.T) {
	felder, literale := sammleParameterStrukturen(t, "")
	for name := range parameterStrukturen {
		if _, da := felder[name]; !da {
			t.Errorf("Struktur %q steht in der Liste, existiert im Baum aber nicht (mehr) — "+
				"Eintrag entfernen oder Namen korrigieren; sonst prüft die Ratsche hier nichts.", name)
		}
	}
	if len(literale) == 0 {
		t.Fatal("kein einziges Literal gefunden — greift der Sammler noch?")
	}
	var maengel []string
	for _, l := range literale {
		if fehlt := fehlendeFelder(felder[l.typ], l.gesetzt); len(fehlt) > 0 {
			maengel = append(maengel, fmt.Sprintf("%s: %s ohne %s (%s)",
				l.ort, l.typ, strings.Join(fehlt, ", "), parameterStrukturen[l.typ]))
		}
	}
	sort.Strings(maengel)
	if len(maengel) > 0 {
		t.Errorf("Parameter-Struktur unvollständig befüllt — ein vergessenes Feld ist hier kein "+
			"Compilerfehler, sondern der Nullwert:\n  %s", strings.Join(maengel, "\n  "))
	}
}

// Selbstprobe: Der Sammler muss eine Lücke auch wirklich finden — in beiden Schreibweisen
// (Feld fehlt ganz, Feld nur im Kommentar) und ohne bei vollständigen Literalen anzuschlagen.
func TestParameterStrukturen_DetektorFindetLuecke(t *testing.T) {
	quelle := `package p
type DueDateOptions struct {
	IstLernmittel   bool
	Medientyp       string
	LmfStichtag     string
	FristBuchTage   int
	FristMedienTage int
	AdditionalYears int
}
type coverBox struct{ x, y, breite, hoehe float64 }
func vollstaendig() DueDateOptions {
	return DueDateOptions{IstLernmittel: true, Medientyp: "Buch", LmfStichtag: "07-31",
		FristBuchTage: 21, FristMedienTage: 7, AdditionalYears: 0}
}
func luecke() DueDateOptions {
	// FristMedienTage: 7,
	return DueDateOptions{IstLernmittel: true, Medientyp: "Buch", LmfStichtag: "07-31",
		FristBuchTage: 21, AdditionalYears: 0}
}
func positional() coverBox { return coverBox{1, 2, 3, 4} }
`
	felder, literale := sammleParameterStrukturen(t, quelle)
	if len(felder["coverBox"]) != 4 {
		t.Errorf("gruppierte Felder (x, y, breite, hoehe) nicht erkannt: %v — genau diese Form "+
			"war beim ersten Anlauf am 05.09.2026 der blinde Fleck", felder["coverBox"])
	}
	var gemeldet []string
	for _, l := range literale {
		if fehlt := fehlendeFelder(felder[l.typ], l.gesetzt); len(fehlt) > 0 {
			gemeldet = append(gemeldet, l.typ+":"+strings.Join(fehlt, ","))
		}
	}
	if len(gemeldet) != 1 || gemeldet[0] != "DueDateOptions:FristMedienTage" {
		t.Errorf("Selbstprobe: erwartet genau eine Lücke (DueDateOptions:FristMedienTage), "+
			"gemeldet %v — ein auskommentiertes Feld darf nicht als gesetzt gelten, ein "+
			"positionales Literal nicht als lückenhaft", gemeldet)
	}
}

type strukturLiteral struct {
	typ     string
	ort     string
	gesetzt map[string]bool
}

// sammleParameterStrukturen liest Felder und Literale der gelisteten Typen. Ist `quelle`
// gesetzt, wird nur dieser Text geparst (Selbstprobe), sonst der ganze Baum.
func sammleParameterStrukturen(t *testing.T, quelle string) (map[string][]string, []strukturLiteral) {
	t.Helper()
	fset := token.NewFileSet()
	felder := map[string][]string{}
	var literale []strukturLiteral

	lies := func(pfad string, dateiQuelle any) error {
		datei, err := parser.ParseFile(fset, pfad, dateiQuelle, 0)
		if err != nil {
			return fmt.Errorf("%s parsen: %w", pfad, err)
		}
		ast.Inspect(datei, func(n ast.Node) bool {
			switch k := n.(type) {
			case *ast.TypeSpec:
				st, ok := k.Type.(*ast.StructType)
				if !ok || !istParameterStruktur(k.Name.Name, quelle) {
					return true
				}
				for _, f := range st.Fields.List {
					for _, name := range f.Names {
						felder[k.Name.Name] = append(felder[k.Name.Name], name.Name)
					}
				}
			case *ast.CompositeLit:
				ident, ok := k.Type.(*ast.Ident)
				if !ok || !istParameterStruktur(ident.Name, quelle) || len(k.Elts) == 0 {
					return true
				}
				gesetzt := map[string]bool{}
				positional := false
				for _, e := range k.Elts {
					kv, ok := e.(*ast.KeyValueExpr)
					if !ok {
						positional = true // der Compiler verlangt hier Vollständigkeit
						continue
					}
					if key, ok := kv.Key.(*ast.Ident); ok {
						gesetzt[key.Name] = true
					}
				}
				if positional {
					return true
				}
				literale = append(literale, strukturLiteral{
					typ: ident.Name, gesetzt: gesetzt,
					ort: fmt.Sprintf("%s:%d", filepath.ToSlash(pfad), fset.Position(k.Pos()).Line),
				})
			}
			return true
		})
		return nil
	}

	if quelle != "" {
		if err := lies("selbstprobe.go", quelle); err != nil {
			t.Fatal(err)
		}
		return felder, literale
	}
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
		if !strings.HasSuffix(pfad, ".go") {
			return nil
		}
		return lies(pfad, nil)
	})
	if err != nil {
		t.Fatal(err)
	}
	return felder, literale
}

// istParameterStruktur: im Selbstproben-Modus zählt jede Struktur der Quelle, sonst die
// Liste oben.
func istParameterStruktur(name, quelle string) bool {
	if quelle != "" {
		return name == "DueDateOptions" || name == "coverBox"
	}
	_, da := parameterStrukturen[name]
	return da
}

func fehlendeFelder(alle []string, gesetzt map[string]bool) []string {
	var fehlt []string
	for _, f := range alle {
		if !gesetzt[f] {
			fehlt = append(fehlt, f)
		}
	}
	return fehlt
}
