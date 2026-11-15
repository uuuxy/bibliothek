package repository

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"bibliothek/pkg/lmfplan"
	"bibliothek/pkg/schulzeit"
)

func tagIn(jahr int, monat time.Month, tag int) time.Time {
	return time.Date(jahr, monat, tag, 10, 0, 0, 0, schulzeit.Zone())
}

// Sommerferien der Tabelle: 2027 ab Montag, 28.06., 2028 ab 03.07., 2029 ab 16.07.; die
// Donnerstage davor sind der 24.06.2027, der 29.06.2028 und der 12.07.2029.
func TestLMFStichtagImSchuljahr(t *testing.T) {
	eigene := lmfplan.FerientabelleAus(`[{"jahr":2032,"von":"2032-06-28","bis":"2032-08-06"}]`)
	imAugust := lmfplan.FerientabelleAus(`[{"jahr":2032,"von":"2032-08-16","bis":"2032-09-24"}]`)
	faelle := []struct {
		name   string
		tag    time.Time
		klasse string
		ferien lmfplan.Ferientabelle
		will   string
	}{
		{"Klasse 5, im September gefragt", tagIn(2026, time.September, 5), "05G1", lmfplan.Hessen(), "2027-06-24"},
		{"Klasse 10, im Juni gefragt", tagIn(2027, time.June, 15), "10R1", lmfplan.Hessen(), "2027-06-24"},
		{"am 1. August gefragt", tagIn(2026, time.August, 1), "7H2", lmfplan.Hessen(), "2027-06-24"},
		{"nach dem Donnerstag gefragt: derselbe Tag", tagIn(2027, time.July, 20), "9H1", lmfplan.Hessen(), "2027-06-24"},
		{"Einführungsphase: Ende des Schuljahres", tagIn(2026, time.September, 5), "ET", lmfplan.Hessen(), "2027-07-31"},
		{"Oberstufe mit Ziffer", tagIn(2026, time.September, 5), "12T", lmfplan.Hessen(), "2027-07-31"},
		{"ohne Klasse", tagIn(2026, time.September, 5), "", lmfplan.Hessen(), "2027-07-31"},
		{"Klasse ohne Jahrgang", tagIn(2026, time.September, 5), "ABG", lmfplan.Hessen(), "2027-07-31"},
		{"Ferien nicht bekannt", tagIn(2031, time.September, 5), "05G1", lmfplan.Hessen(), "2032-07-31"},
		{"Ferien von der Schule eingetragen", tagIn(2031, time.September, 5), "05G1", eigene, "2032-06-24"},
		{"Ferien erst im August: Ende des Schuljahres", tagIn(2031, time.September, 5), "05G1", imAugust, "2032-07-31"},
	}
	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			got := LMFStichtagImSchuljahr(f.tag, f.klasse, f.ferien).Format(time.DateOnly)
			if got != f.will {
				t.Errorf("Stichtag für %q am %s = %s, erwartet %s", f.klasse, f.tag.Format(time.DateOnly), got, f.will)
			}
		})
	}
}

// Die zweite Frage: Die Frist einer neuen Ausleihe darf nicht in der Vergangenheit liegen, sonst
// mahnt das Programm das Buch am Tag der Ausgabe. Mit jedem Schuljahr rückt der Jahrgang vor.
func TestLMFStichtagAbTag(t *testing.T) {
	faelle := []struct {
		name        string
		tag         time.Time
		klasse      string
		zusatzJahre int
		will        string
	}{
		{"Klasse 5 im September", tagIn(2026, time.September, 5), "05G1", 0, "2027-06-24"},
		{"am Donnerstag ausgeliehen: heute, nicht in einem Jahr", tagIn(2027, time.June, 24), "05G1", 0, "2027-06-24"},
		{"am Freitag danach: nächstes Schuljahr", tagIn(2027, time.June, 25), "05G1", 0, "2028-06-29"},
		{"Oberstufe am Freitag: dieses Schuljahr", tagIn(2027, time.June, 25), "ET", 0, "2027-07-31"},
		{"Klasse 10 am Freitag: nächstes Schuljahr in der Oberstufe", tagIn(2027, time.June, 25), "10G1", 0, "2028-07-31"},
		{"Band bis Jahrgang 9 für Klasse 7", tagIn(2026, time.September, 5), "07G1", 2, "2029-07-12"},
		{"derselbe Band nach dem Donnerstag", tagIn(2027, time.June, 25), "07G1", 2, "2029-07-12"},
		{"Band der Oberstufe bis Jahrgang 13", tagIn(2026, time.September, 5), "ET", 2, "2029-07-31"},
		{"Band aus Klasse 10 in die Oberstufe", tagIn(2026, time.September, 5), "10G1", 1, "2028-07-31"},
	}
	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			got := LMFStichtagAbTag(f.tag, f.klasse, f.zusatzJahre, lmfplan.Hessen()).Format(time.DateOnly)
			if got != f.will {
				t.Errorf("Stichtag für %q (+%d) am %s = %s, erwartet %s",
					f.klasse, f.zusatzJahre, f.tag.Format(time.DateOnly), got, f.will)
			}
		})
	}
}

// Paar-Gate: Solange der Stichtag des Schuljahres noch bevorsteht, müssen Ausleihe
// (LMFStichtagAbTag) und Rückweg des Plans (LMFStichtagImSchuljahr) denselben Tag nennen. Ohne
// diese Einigkeit bekäme dasselbe Buch je nach Weg eine um ein Jahr verschiedene Frist.
func TestLMFStichtag_AusleiheUndRueckwegEinig(t *testing.T) {
	ferien := lmfplan.Hessen()
	for _, klasse := range []string{"05G1", "9H1", "10R1", "ET", "12T", "", "ABG"} {
		for tag := tagIn(2026, time.August, 1); tag.Before(tagIn(2027, time.August, 1)); tag = tag.AddDate(0, 0, 1) {
			imSchuljahr := LMFStichtagImSchuljahr(tag, klasse, ferien)
			if imSchuljahr.Before(time.Date(tag.Year(), tag.Month(), tag.Day(), 0, 0, 0, 0, schulzeit.Zone())) {
				continue // vorbei — hier darf die Ausleihe bewusst weiterrücken
			}
			if abTag := LMFStichtagAbTag(tag, klasse, 0, ferien); !imSchuljahr.Equal(abTag) {
				t.Errorf("%q am %s: Ausleihe sagt %s, Rückweg sagt %s", klasse, tag.Format(time.DateOnly),
					abTag.Format(time.DateOnly), imSchuljahr.Format(time.DateOnly))
			}
		}
	}
}

// --- Ratsche: die Rechnung darf nicht wieder ein zweites Mal entstehen ---

// lmfPlanAnker ist die zweite Stelle, die den Donnerstag vor den Ferien liest: der Rahmen, mit dem
// ein neuer Rückgabe-Plan beginnt. Er ist ein Vorschlag im Planer, keine Frist.
var lmfPlanAnker = filepath.Join("internal", "service", "lmf_plan_vorgabe.go")

// stichtagsVerstoesse findet in einer Go-Quelle die zwei Formen, in denen eine zweite Fassung der
// Rechnung entsteht: einen Stichtag als Text („07-31", wie die frühere Einstellung) und einen
// eigenen Aufruf von lmfplan.DonnerstagVor. Gearbeitet wird am AST, nicht am Text: Ein Kommentar,
// der das Literal erklärt, ist kein Verstoß.
func stichtagsVerstoesse(t *testing.T, name, quelle string, donnerstagErlaubt bool) []string {
	t.Helper()
	baum, err := parser.ParseFile(token.NewFileSet(), name, quelle, 0)
	if err != nil {
		t.Fatalf("%s parsen: %v", name, err)
	}
	var funde []string
	ast.Inspect(baum, func(n ast.Node) bool {
		switch k := n.(type) {
		case *ast.BasicLit:
			if k.Kind == token.STRING && k.Value == `"07-31"` {
				funde = append(funde, "Stichtag als Text "+k.Value)
			}
		case *ast.SelectorExpr:
			paket, ok := k.X.(*ast.Ident)
			if ok && paket.Name == "lmfplan" && k.Sel.Name == "DonnerstagVor" && !donnerstagErlaubt {
				funde = append(funde, "eigene Rechnung über lmfplan.DonnerstagVor")
			}
		}
		return true
	})
	return funde
}

func TestLMFStichtag_NurEineRechnung(t *testing.T) {
	// Selbstprobe zuerst: Ein Muster, das nichts fasst, meldet ewig „alles gut".
	probe := `package p

import "bibliothek/pkg/lmfplan"

func f(beginn time.Time) {
	_ = "07-31"
	_ = lmfplan.DonnerstagVor(beginn)
}
`
	if funde := stichtagsVerstoesse(t, "probe.go", probe, false); len(funde) != 2 {
		t.Fatalf("Selbstprobe: der Detektor fasst die verbotenen Formen nicht (%d von 2): %v", len(funde), funde)
	}

	wurzel := ".."
	eigen := filepath.Join("repository", "lmf_stichtag.go")
	var verstoesse []string
	dateien := 0
	err := filepath.Walk(wurzel, func(pfad string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if name := info.Name(); name == "node_modules" || name == "vendor" || name == ".git" || name == "frontend" {
				return filepath.SkipDir
			}
			return nil
		}
		rel, err := filepath.Rel(wurzel, pfad)
		if err != nil {
			return err
		}
		if !strings.HasSuffix(pfad, ".go") || strings.HasSuffix(pfad, "_test.go") || rel == eigen {
			return nil
		}
		inhalt, err := os.ReadFile(pfad)
		if err != nil {
			return err
		}
		dateien++
		for _, fund := range stichtagsVerstoesse(t, pfad, string(inhalt), rel == lmfPlanAnker) {
			verstoesse = append(verstoesse, rel+": "+fund)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Baum lesen: %v", err)
	}
	if dateien < 100 {
		t.Fatalf("nur %d Go-Dateien gesehen — der Lauf greift ins Leere und dieses Gate wäre still grün", dateien)
	}
	if len(verstoesse) > 0 {
		t.Errorf("Zweite Fassung der Stichtags-Rechnung:\n  %s\n\n"+
			"Der Stichtag wird an einer Stelle gerechnet (repository/lmf_stichtag.go): "+
			"LMFStichtagImSchuljahr für den Rückweg des Plans, LMFStichtagAbTag für eine neue "+
			"Ausleihe, LMFStichtagNachJahren für den Büchertausch. Mit zwei Fassungen bekäme "+
			"dasselbe Schulbuch je nach Weg eine andere Frist.", strings.Join(verstoesse, "\n  "))
	}
}
