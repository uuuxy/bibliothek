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

	"bibliothek/pkg/schulzeit"
)

func tagIn(jahr int, monat time.Month, tag int) time.Time {
	return time.Date(jahr, monat, tag, 10, 0, 0, 0, schulzeit.Zone())
}

// Die Rechnung selbst — als Tabelle, damit der Fall, der die beiden Fassungen
// auseinandergehen ließ (Stichtag ab August), nicht wieder nur „theoretisch" ist.
func TestLmfStichtagImSchuljahr(t *testing.T) {
	faelle := []struct {
		name     string
		tag      time.Time
		stichtag string
		will     string
	}{
		{"Vorgabe, im September gefragt", tagIn(2026, time.September, 5), "07-31", "2027-07-31"},
		{"Vorgabe, im Juni gefragt", tagIn(2027, time.June, 15), "07-31", "2027-07-31"},
		{"Vorgabe, am 1. August gefragt", tagIn(2026, time.August, 1), "07-31", "2027-07-31"},
		{"Vorgabe, am Stichtag selbst", tagIn(2027, time.July, 31), "07-31", "2027-07-31"},
		// Der Fall, an dem die zwei alten Fassungen ein Jahr auseinanderlagen:
		{"Stichtag im September, im September gefragt", tagIn(2026, time.September, 5), "09-30", "2026-09-30"},
		{"Stichtag im September, im Juni gefragt", tagIn(2027, time.June, 15), "09-30", "2026-09-30"},
		{"Stichtag im Dezember", tagIn(2026, time.September, 5), "12-24", "2026-12-24"},
		{"Stichtag im Januar", tagIn(2026, time.September, 5), "01-15", "2027-01-15"},
		{"unlesbarer Stichtag fällt auf die Vorgabe", tagIn(2026, time.September, 5), "Juli", "2027-07-31"},
		{"Monat 13 fällt auf die Vorgabe", tagIn(2026, time.September, 5), "13-01", "2027-07-31"},
	}
	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			got := LmfStichtagImSchuljahr(f.tag, f.stichtag).Format("2006-01-02")
			if got != f.will {
				t.Errorf("Stichtag %q am %s: got %s, will %s",
					f.stichtag, f.tag.Format("2006-01-02"), got, f.will)
			}
		})
	}
}

// Die zweite Frage: die Frist einer NEUEN Ausleihe darf nicht in der Vergangenheit
// liegen — sonst mahnt das System das Buch am Tag der Ausgabe.
func TestLmfStichtagAbTag(t *testing.T) {
	faelle := []struct {
		name     string
		tag      time.Time
		stichtag string
		will     string
	}{
		{"Vorgabe: immer das Ende dieses Schuljahres", tagIn(2026, time.September, 5), "07-31", "2027-07-31"},
		{"am Stichtag ausgeliehen: heute, nicht in einem Jahr", tagIn(2027, time.July, 31), "07-31", "2027-07-31"},
		{"Stichtag im September, davor ausgeliehen", tagIn(2026, time.September, 5), "09-30", "2026-09-30"},
		{"Stichtag im September, danach ausgeliehen: nächstes Schuljahr", tagIn(2026, time.October, 15), "09-30", "2027-09-30"},
	}
	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			got := LmfStichtagAbTag(f.tag, f.stichtag).Format("2006-01-02")
			if got != f.will {
				t.Errorf("Stichtag %q am %s: got %s, will %s",
					f.stichtag, f.tag.Format("2006-01-02"), got, f.will)
			}
		})
	}
}

// Paar-Gate: Solange der Stichtag des Schuljahres noch bevorsteht, MÜSSEN Ausleihe
// (LmfStichtagAbTag) und Rückweg des Plans (LmfStichtagImSchuljahr) denselben Tag
// nennen. Genau diese Einigkeit fehlte bis zum 06.09.2026 — dasselbe Buch hätte je
// nach Weg eine um ein Jahr verschiedene Frist bekommen.
func TestLmfStichtag_AusleiheUndRueckwegEinig(t *testing.T) {
	for _, stichtag := range []string{"07-31", "06-15", "09-30", "12-24", "01-15", "08-01"} {
		for monat := time.January; monat <= time.December; monat++ {
			tag := tagIn(2026, monat, 10)
			imSchuljahr := LmfStichtagImSchuljahr(tag, stichtag)
			if imSchuljahr.Before(time.Date(tag.Year(), tag.Month(), tag.Day(), 0, 0, 0, 0, schulzeit.Zone())) {
				continue // vorbei — hier darf die Ausleihe bewusst weiterrücken
			}
			abTag := LmfStichtagAbTag(tag, stichtag)
			if !imSchuljahr.Equal(abTag) {
				t.Errorf("Stichtag %q am %s: Ausleihe sagt %s, Rückweg sagt %s",
					stichtag, tag.Format("2006-01-02"),
					abTag.Format("2006-01-02"), imSchuljahr.Format("2006-01-02"))
			}
		}
	}
}

// --- Ratsche: die Rechnung darf nicht wieder ein zweites Mal entstehen ---

// stichtagsVerstoesse findet in einer Go-Quelle die zwei Formen, in denen eine zweite
// Fassung der Rechnung entsteht: das Vorgabe-Literal „07-31" und ein eigenes Zerlegen
// eines Stichtag-Werts an „-". Gearbeitet wird am AST, nicht am Text — ein Kommentar,
// der das Literal ERKLÄRT, ist kein Verstoß (Bugklasse „Lügende Ratsche durch
// Kommentar"), ein echtes Literal dagegen schon.
func stichtagsVerstoesse(t *testing.T, name, quelle string) []string {
	t.Helper()
	fset := token.NewFileSet()
	baum, err := parser.ParseFile(fset, name, quelle, 0)
	if err != nil {
		t.Fatalf("%s parsen: %v", name, err)
	}
	var funde []string
	ast.Inspect(baum, func(n ast.Node) bool {
		switch k := n.(type) {
		case *ast.BasicLit:
			if k.Kind == token.STRING && k.Value == `"`+StandardLmfStichtag+`"` {
				funde = append(funde, "Vorgabe-Literal "+k.Value)
			}
		case *ast.CallExpr:
			ruf, ok := k.Fun.(*ast.SelectorExpr)
			if !ok || len(k.Args) < 2 {
				return true
			}
			paket, ok := ruf.X.(*ast.Ident)
			if !ok || paket.Name != "strings" {
				return true
			}
			if ruf.Sel.Name != "Split" && ruf.Sel.Name != "SplitN" {
				return true
			}
			erstes := quelleVon(fset, k.Args[0])
			if strings.Contains(strings.ToLower(erstes), "stichtag") {
				funde = append(funde, "eigenes Zerlegen: strings."+ruf.Sel.Name+"("+erstes+", …)")
			}
		}
		return true
	})
	return funde
}

func quelleVon(fset *token.FileSet, n ast.Node) string {
	pos := fset.Position(n.Pos())
	ende := fset.Position(n.End())
	if pos.Filename != ende.Filename {
		return ""
	}
	return quellAusschnitt[pos.Filename][pos.Offset:ende.Offset]
}

var quellAusschnitt = map[string]string{}

func TestLmfStichtag_NurEineRechnung(t *testing.T) {
	// Selbstprobe zuerst: Ein Muster, das nichts fasst, meldet ewig „alles gut".
	probe := `package p

import "strings"

func f(einst struct{ LmfStichtag string }) {
	_ = "07-31"
	_ = strings.SplitN(einst.LmfStichtag, "-", 2)
}
`
	quellAusschnitt["probe.go"] = probe
	if funde := stichtagsVerstoesse(t, "probe.go", probe); len(funde) != 2 {
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
		quellAusschnitt[pfad] = string(inhalt)
		for _, fund := range stichtagsVerstoesse(t, pfad, string(inhalt)) {
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
			"Der Stichtag wird an EINER Stelle gerechnet (repository/lmf_stichtag.go): "+
			"LmfStichtagImSchuljahr für den Rückweg des Plans, LmfStichtagAbTag für eine neue "+
			"Ausleihe, StandardLmfStichtag für die Vorgabe. Zwei Fassungen waren sich zuletzt "+
			"nur bei der Vorgabe einig.", strings.Join(verstoesse, "\n  "))
	}
}
