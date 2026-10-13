package repository_test

import (
	"errors"
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"go/types"
	"io/fs"
	"path"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"bibliothek/repository"
)

// Wer die Einstellungen liest, behandelt den Lesefehler.
//
// Das Lesen liefert bei einem Fehler die Vorgaben des Programms zusammen mit dem Fehler. Wer
// den Fehler mit `_` verwirft, arbeitet mit den Vorgaben weiter, als wären sie eingestellt: An
// sechs Türen entstanden so Blätter und eine Bestellmail ohne die Angaben der Schule
// (Bestellbericht, Bestellmail, Auskunft, Mahnbrief, Elternbrief, Ersatzforderung). Der Linter
// meldet die Stelle, eine Zeile `//nolint:errcheck` stellt ihn ab; diese Prüfung lässt sich so
// nicht abstellen.
//
// Erkannt wird das Lesen am Ergebnis und nicht am Namen: jeder Aufruf, der die Einstellungen
// zusammen mit einem zweiten Wert liefert. Eine eigene Funktion um das Lesen, die den Fehler
// weiterreicht, fällt damit auch darunter.
//
// Blind für: einen Fehler, der einer Variable zugewiesen und danach nicht geprüft wird, und
// für eine eigene Funktion um das Lesen, die den Fehler selbst schluckt.
func TestEinstellungenLesefehlerWirdNichtVerworfen(t *testing.T) {
	einstellungen := reflect.TypeOf(repository.SystemEinstellungen{})
	lader := newTypLader(strings.TrimSuffix(einstellungen.PkgPath(), "/repository"), "..")
	gelesen, verworfen := 0, []string{}

	for _, paket := range lader.ladeAlle(t) {
		for _, datei := range paket.dateien {
			ast.Inspect(datei, func(n ast.Node) bool {
				zuweisung, ok := n.(*ast.AssignStmt)
				if !ok || len(zuweisung.Rhs) != 1 || len(zuweisung.Lhs) != 2 {
					return true
				}
				aufruf, ok := zuweisung.Rhs[0].(*ast.CallExpr)
				if !ok || !liefertZeigerAuf(paket.info.Types[aufruf].Type, einstellungen) {
					return true
				}
				gelesen++
				if fehler, ok := zuweisung.Lhs[1].(*ast.Ident); ok && fehler.Name == "_" {
					verworfen = append(verworfen, lader.fset.Position(zuweisung.Pos()).String())
				}
				return true
			})
		}
	}

	// Gegenprobe am Detektor: Findet er die Leser überhaupt?
	if gelesen < 20 {
		t.Fatalf("nur %d Stellen gefunden, die die Einstellungen lesen: Der Detektor greift ins Leere", gelesen)
	}
	t.Logf("%d Stellen lesen die Einstellungen", gelesen)
	for _, stelle := range verworfen {
		t.Errorf("%s verwirft den Fehler beim Lesen der Einstellungen und arbeitet mit den Vorgaben weiter. "+
			"Den Fehler behandeln: die Antwort verweigern oder die Vorgabe ausdrücklich wählen.", stelle)
	}
}

// liefertZeigerAuf meldet, ob ein Aufruf zwei Werte liefert und der erste ein Zeiger auf den
// genannten Typ ist.
func liefertZeigerAuf(ergebnis types.Type, ziel reflect.Type) bool {
	werte, ok := ergebnis.(*types.Tuple)
	if !ok || werte.Len() != 2 {
		return false
	}
	zeiger, ok := werte.At(0).Type().(*types.Pointer)
	if !ok {
		return false
	}
	benannt, ok := zeiger.Elem().(*types.Named)
	if !ok || benannt.Obj().Pkg() == nil {
		return false
	}
	return benannt.Obj().Name() == ziel.Name() && benannt.Obj().Pkg().Path() == ziel.PkgPath()
}

// typPaket ist ein Paket des Projekts mit seinen Dateien und den Typen seiner Ausdrücke.
type typPaket struct {
	dateien []*ast.File
	info    *types.Info
}

// typLader prüft die Typen des Programmcodes aus dem Quelltext, ohne gebaute Pakete. Ein Paket
// von außerhalb des Projekts liefert er leer: Ein Ausdruck mit einem Typ von dort bleibt ohne
// Typ, und die Fehler dazu verwirft er. Was ein Aufruf aus dem Projekt zurückgibt, steht
// trotzdem fest, sobald der Typ dem Projekt gehört.
type typLader struct {
	fset   *token.FileSet
	modul  string
	wurzel string
	typen  map[string]*types.Package
	pakete map[string]*typPaket
}

func newTypLader(modul, wurzel string) *typLader {
	return &typLader{
		fset: token.NewFileSet(), modul: modul, wurzel: wurzel,
		typen: map[string]*types.Package{}, pakete: map[string]*typPaket{},
	}
}

// Import erfüllt types.Importer: ein Paket des Projekts aus dem Quelltext, jedes andere leer.
func (l *typLader) Import(pfad string) (*types.Package, error) {
	if p, ok := l.typen[pfad]; ok {
		return p, nil
	}
	if pfad != l.modul && !strings.HasPrefix(pfad, l.modul+"/") {
		leer := types.NewPackage(pfad, path.Base(pfad))
		leer.MarkComplete()
		l.typen[pfad] = leer
		return leer, nil
	}
	ordner := filepath.Join(l.wurzel, filepath.FromSlash(strings.TrimPrefix(strings.TrimPrefix(pfad, l.modul), "/")))
	gebaut, err := build.Default.ImportDir(ordner, 0)
	if err != nil {
		return nil, err
	}
	paket := &typPaket{info: &types.Info{Types: map[ast.Expr]types.TypeAndValue{}}}
	for _, name := range gebaut.GoFiles {
		datei, err := parser.ParseFile(l.fset, filepath.Join(ordner, name), nil, parser.SkipObjectResolution)
		if err != nil {
			return nil, err
		}
		paket.dateien = append(paket.dateien, datei)
	}
	pruefung := types.Config{Importer: l, Error: func(error) {}, FakeImportC: true}
	// Der Fehler ist der erste der verworfenen: Namen aus den leeren Paketen fehlen. Er zählt
	// nur, wenn gar kein Paket entstand.
	typen, err := pruefung.Check(pfad, l.fset, paket.dateien, paket.info)
	if typen == nil {
		return nil, err
	}
	l.typen[pfad] = typen
	l.pakete[pfad] = paket
	return typen, nil
}

// ladeAlle lädt jedes Paket des Projekts, das Programmcode trägt.
func (l *typLader) ladeAlle(t *testing.T) map[string]*typPaket {
	t.Helper()
	err := filepath.WalkDir(l.wurzel, func(pfad string, d fs.DirEntry, err error) error {
		if err != nil || !d.IsDir() {
			return err
		}
		// Versteckte Ordner tragen keinen Quelltext des Programms, wohl aber Arbeitskopien.
		if name := d.Name(); name == "node_modules" || name == "frontend" || name == "testdata" ||
			(strings.HasPrefix(name, ".") && pfad != l.wurzel) {
			return filepath.SkipDir
		}
		relativ, err := filepath.Rel(l.wurzel, pfad)
		if err != nil {
			return err
		}
		var ohneGo *build.NoGoError
		if _, err := l.Import(path.Join(l.modul, filepath.ToSlash(relativ))); err != nil && !errors.As(err, &ohneGo) {
			return err
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Baum lesen: %v", err)
	}
	return l.pakete
}
