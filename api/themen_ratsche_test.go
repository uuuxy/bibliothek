package api

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// Die Türen stehen je Thema in einem Paket unter api/. Ein Thema bindet aus diesem Baum nur
// api/tuer ein: Was zwei Themen brauchen, steht dort oder in einer Schicht darunter. Der
// Compiler weist nur einen Kreis ab; diese Prüfung weist auch die Einbindung in eine Richtung
// ab.
//
// Blindheit: Sie liest Einbindungen. Greift ein Thema über repository/ in die Tabellen eines
// anderen, sieht sie es nicht.

const (
	paketAPI  = "bibliothek/api"
	paketTuer = "bibliothek/api/tuer"
)

// einbindungErlaubt sagt, ob eine Datei im Ordner eines Themas das Paket einbinden darf. Ein
// Testpaket (package thema_test) prüft über den ganzen Router und darf dafür api/ und das
// eigene Thema einbinden.
func einbindungErlaubt(thema, ziel string, testpaket bool) bool {
	if ziel != paketAPI && !strings.HasPrefix(ziel, paketAPI+"/") {
		return true
	}
	switch ziel {
	case paketTuer:
		return thema != "tuer" || testpaket
	case paketAPI, paketAPI + "/" + thema:
		return testpaket
	}
	return false
}

func TestKeinThemaBindetEinAnderesEin(t *testing.T) {
	fset := token.NewFileSet()
	gelesen := map[string]int{}  // Dateien je Ordner unter api/
	mitTuer := map[string]bool{} // Ordner, in denen eine Einbindung von api/tuer steht
	err := filepath.WalkDir(".", func(pfad string, eintrag fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		name := eintrag.Name()
		if eintrag.IsDir() {
			if pfad != "." && (strings.HasPrefix(name, ".") || name == "testdata" || name == "uploads") {
				return filepath.SkipDir
			}
			return nil
		}
		teile := strings.Split(filepath.ToSlash(pfad), "/")
		// Die Dateien von api/ selbst binden die Themen ein: Dort stehen Router und Zugänge.
		if !strings.HasSuffix(name, ".go") || len(teile) < 2 {
			return nil
		}
		thema := teile[0]
		datei, err := parser.ParseFile(fset, pfad, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		gelesen[thema]++
		testpaket := strings.HasSuffix(datei.Name.Name, "_test")
		for _, imp := range datei.Imports {
			ziel, err := strconv.Unquote(imp.Path.Value)
			if err != nil {
				return err
			}
			if ziel == paketTuer {
				mitTuer[thema] = true
			}
			if !einbindungErlaubt(thema, ziel, testpaket) {
				t.Errorf("api/%s bindet %s ein. Ein Thema bindet aus api/ nur api/tuer ein; was zwei "+
					"Themen brauchen, gehört nach api/tuer oder in eine Schicht darunter "+
					"(docs/ARCHITEKTUR.md 5.2.2).", filepath.ToSlash(pfad), ziel)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("api/ nicht lesbar: %v", err)
	}

	// Die Prüfung misst nur, wenn sie jedes Thema gelesen hat, zu dem der Server einen Zugang
	// erklärt, und dort die Einbindungen sieht: Jedes Thema bindet api/tuer ein.
	themen := erklaerteZugaenge(t)
	if len(themen) == 0 {
		t.Fatal("kein Zugang zu einem Thema gefunden — die Prüfung misst nichts")
	}
	for _, thema := range themen {
		if gelesen[thema] == 0 {
			t.Errorf("zum Zugang %s gibt es keinen Ordner api/%s mit Go-Dateien — die Prüfung "+
				"liest dieses Thema nicht", thema, thema)
		} else if !mitTuer[thema] {
			t.Errorf("in api/%s steht keine Einbindung von api/tuer — die Prüfung liest die "+
				"Einbindungen dieses Themas offenbar nicht", thema)
		}
	}
}

// Eine Tür ist eine Fabrik (func (t *Tueren) X() http.HandlerFunc), und die Routenliste ruft sie:
// s.katalog().TitelListe(). Eine Methode, die selbst Antwort und Anfrage nimmt, wäre eine zweite
// Form derselben Sache (docs/ARCHITEKTUR.md 5.2.2).
//
// Blindheit: Sie liest Methoden am Typ Tueren. Eine Funktion ohne Empfänger mit derselben
// Signatur sieht sie nicht.
func TestTuerenSindFabriken(t *testing.T) {
	fset := token.NewFileSet()
	fabriken := map[string]int{} // Türen je Thema, die einen Handler liefern
	err := filepath.WalkDir(".", func(pfad string, eintrag fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		name := eintrag.Name()
		if eintrag.IsDir() {
			if pfad != "." && (strings.HasPrefix(name, ".") || name == "testdata" || name == "uploads") {
				return filepath.SkipDir
			}
			return nil
		}
		teile := strings.Split(filepath.ToSlash(pfad), "/")
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") || len(teile) < 2 {
			return nil
		}
		datei, err := parser.ParseFile(fset, pfad, nil, 0)
		if err != nil {
			return err
		}
		for _, erklaerung := range datei.Decls {
			methode, ok := erklaerung.(*ast.FuncDecl)
			// Nur eine exportierte Methode erreicht die Routenliste; eine klein geschriebene mit
			// Antwort und Anfrage ist der Rumpf, an den eine Fabrik weiterreicht.
			if !ok || methode.Recv == nil || !methode.Name.IsExported() ||
				!strings.HasSuffix(types.ExprString(methode.Recv.List[0].Type), "Tueren") {
				continue
			}
			switch {
			case typen(methode.Type.Params) == "http.ResponseWriter, *http.Request":
				t.Errorf("api/%s: %s nimmt selbst Antwort und Anfrage. Eine Tür ist eine Fabrik, "+
					"func (t *Tueren) %s() http.HandlerFunc (docs/ARCHITEKTUR.md 5.2.2).",
					filepath.ToSlash(pfad), methode.Name.Name, methode.Name.Name)
			case typen(methode.Type.Results) == "http.HandlerFunc" || typen(methode.Type.Results) == "http.Handler":
				fabriken[teile[0]]++
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("api/ nicht lesbar: %v", err)
	}
	for _, thema := range erklaerteZugaenge(t) {
		if fabriken[thema] == 0 {
			t.Errorf("in api/%s keine Tür gefunden, die einen Handler liefert — die Prüfung liest "+
				"dieses Thema nicht", thema)
		}
	}
}

// typen nennt die Typen einer Liste von Parametern oder Ergebnissen, durch Komma getrennt.
func typen(liste *ast.FieldList) string {
	if liste == nil {
		return ""
	}
	var aus []string
	for _, feld := range liste.List {
		for range max(1, len(feld.Names)) {
			aus = append(aus, types.ExprString(feld.Type))
		}
	}
	return strings.Join(aus, ", ")
}

func TestEinbindungErlaubt_UnterscheidetDieFaelle(t *testing.T) {
	faelle := []struct {
		was       string
		thema     string
		ziel      string
		testpaket bool
		erlaubt   bool
	}{
		{"was jede Tür braucht", "leser", "bibliothek/api/tuer", false, true},
		{"eine Schicht darunter", "leser", "bibliothek/repository", false, true},
		{"ein Paket mit ähnlichem Namen", "leser", "bibliothek/apierrors", false, true},
		{"die Standardbibliothek", "leser", "net/http", false, true},
		{"ein anderes Thema", "leser", "bibliothek/api/theke", false, false},
		{"ein anderes Thema aus einem Test", "leser", "bibliothek/api/theke", true, false},
		{"ein Ordner unter einem anderen Thema", "leser", "bibliothek/api/theke/tiefer", false, false},
		{"der Router aus dem Paket des Themas", "leser", "bibliothek/api", false, false},
		{"der Router aus dem Testpaket", "leser", "bibliothek/api", true, true},
		{"das eigene Thema aus dem Testpaket", "leser", "bibliothek/api/leser", true, true},
		{"das eigene Thema aus dem Paket selbst", "leser", "bibliothek/api/leser", false, false},
		{"api/tuer bindet sich nicht selbst ein", "tuer", "bibliothek/api/tuer", false, false},
		{"api/tuer bindet kein Thema ein", "tuer", "bibliothek/api/leser", false, false},
		{"das Testpaket von api/tuer bindet api/tuer ein", "tuer", "bibliothek/api/tuer", true, true},
	}
	for _, f := range faelle {
		if got := einbindungErlaubt(f.thema, f.ziel, f.testpaket); got != f.erlaubt {
			t.Errorf("%s (%s bindet %s ein, Testpaket %v): erlaubt=%v, erwartet %v",
				f.was, f.thema, f.ziel, f.testpaket, got, f.erlaubt)
		}
	}
}
