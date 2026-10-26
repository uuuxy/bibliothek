package service

import (
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
)

// TestMeldungZeigtNurDieMeldung: Die Theke zeigt MeldungAus(err) (api/theke/eingabe.go). Stünde
// dort die Bezeichnung der Fehlerart vor der Meldung („Fehler: conflict: dieses Exemplar wurde
// soeben bereits ausgeliehen"), wäre es kein Satz für den Bildschirm (M3, Word choice: „UI text
// should be understandable by anyone, anywhere"). Der Text des Fehlers beginnt dagegen mit der
// Art, klein, wie ein Fehlertext (Decisions, „Error strings"). Die Art bleibt für errors.Is
// erhalten, auch eingewickelt, denn an ihr hängt der HTTP-Status.
func TestMeldungZeigtNurDieMeldung(t *testing.T) {
	for _, art := range []error{ErrUngueltigerZustand, ErrKonflikt, ErrNichtGefunden} {
		err := meldung(art, "Buchexemplar %s ist ausgesondert", "4711")
		const satz = "Buchexemplar 4711 ist ausgesondert"
		if got := MeldungAus(err); got != satz {
			t.Errorf("MeldungAus(%v) = %q, erwartet %q", art, got, satz)
		}
		if got := MeldungAus(fmt.Errorf("beim Ausleihen: %w", err)); got != satz {
			t.Errorf("MeldungAus(eingewickelt, %v) = %q, erwartet %q", art, got, satz)
		}
		if soll := art.Error() + ": " + satz; err.Error() != soll {
			t.Errorf("%v: Error() = %q, erwartet %q", art, err.Error(), soll)
		}
		if !errors.Is(err, art) || !errors.Is(fmt.Errorf("beim Ausleihen: %w", err), art) {
			t.Errorf("die Art %q muss erkennbar bleiben, auch eingewickelt", art)
		}
		if errors.Is(err, ErrGesperrt) {
			t.Errorf("%v darf nicht als Sperre gelten", art)
		}
	}
}

// TestSperrmeldungBeginntGross: Eine Sperre steht an der Theke als Satz da, groß begonnen
// (M3, UX writing: „Capitalize the first word of a sentence or phrase"), wie die übrigen
// Sätze zur Sperre (meldungVerlaengerungGesperrt). Der Fehlertext bleibt Wort für Wort, wie er
// war: klein (Decisions, „Error strings"), denn er steht im Log und in der Ablage einer Antwort.
func TestSperrmeldungBeginntGross(t *testing.T) {
	amLeser := SperreAmLeser(Sperre("Manuelle Sperre"))
	ehemalig := &SperrgrundFehler{Kern: SperreAmLeser(ErrGesperrt), Grund: "Abgang 2026"}
	for _, fall := range []struct {
		err        error
		meldung    string
		fehlertext string
	}{
		{ErrGesperrt, "Die Ausleihe ist gesperrt", "die ausleihe ist gesperrt"},
		{amLeser, "Die Ausleihe ist gesperrt: Manuelle Sperre", "die ausleihe ist gesperrt: Manuelle Sperre"},
		{&SperrgrundFehler{Kern: amLeser, Grund: "Ausweis verloren"},
			"Die Ausleihe ist gesperrt: Manuelle Sperre: Ausweis verloren",
			"die ausleihe ist gesperrt: Manuelle Sperre: Ausweis verloren"},
		{(&SperrgrundFehler{Kern: amLeser, Grund: "Ausweis verloren"}).OhneGrund(),
			"Die Ausleihe ist gesperrt: Manuelle Sperre — bitte an die Bibliotheksleitung wenden",
			"die ausleihe ist gesperrt: Manuelle Sperre — bitte an die Bibliotheksleitung wenden"},
		{ehemalig, "Die Ausleihe ist gesperrt: Abgang 2026", "die ausleihe ist gesperrt: Abgang 2026"},
		{ehemalig.OhneGrund(), "Die Ausleihe ist gesperrt — bitte an die Bibliotheksleitung wenden",
			"die ausleihe ist gesperrt — bitte an die Bibliotheksleitung wenden"},
		{UebergehbareSperre(Sperre("%d überfällige Medien vorhanden (Sperr-Automatik)", 2)),
			"Die Ausleihe ist gesperrt: 2 überfällige Medien vorhanden (Sperr-Automatik)",
			"die ausleihe ist gesperrt: 2 überfällige Medien vorhanden (Sperr-Automatik)"},
		{fmt.Errorf("beim Ausleihen: %w", Sperre("Gerät ist aktuell gesperrt")),
			"Die Ausleihe ist gesperrt: Gerät ist aktuell gesperrt",
			"beim Ausleihen: die ausleihe ist gesperrt: Gerät ist aktuell gesperrt"},
	} {
		if got := MeldungAus(fall.err); got != fall.meldung {
			t.Errorf("MeldungAus(%q) = %q, erwartet %q", fall.err, got, fall.meldung)
		}
		if got := fall.err.Error(); got != fall.fehlertext {
			t.Errorf("Error() = %q, erwartet %q", got, fall.fehlertext)
		}
		if !errors.Is(fall.err, ErrGesperrt) {
			t.Errorf("%q: errors.Is(…, ErrGesperrt) = false, der Status 403 ginge verloren", fall.err)
		}
	}
	// Die Merkmale überstehen OhneGrund: An ihnen wählt der Dialog der Theke, was er anbietet.
	if !IstSperreAmLeser((&SperrgrundFehler{Kern: amLeser, Grund: "x"}).OhneGrund()) {
		t.Error("OhneGrund verliert das Merkmal der Sperre am Leser")
	}
}

// ohneBezeichnung sind die Fehlerarten, deren Text eine interne Bezeichnung ist. ErrGesperrt fehlt
// mit Absicht: „die ausleihe ist gesperrt" ist der Kopf jeder Sperrmeldung (fehler.go).
var ohneBezeichnung = map[string]bool{"ErrUngueltigerZustand": true, "ErrKonflikt": true, "ErrNichtGefunden": true}

// TestFehlerartenNurUeberMeldung: Kein fmt.Errorf im Bestand wickelt eine dieser Arten ein —
// sonst stünde ihre Bezeichnung wieder vor der Meldung an der Theke.
//
// Blindheit: Gesehen wird fmt.Errorf mit der Art als Argument, in Go-Dateien außer Tests (im
// Paket service als Name, anderswo als service.Name). errors.Join oder ein eigener Wrapper
// fallen nicht auf.
func TestFehlerartenNurUeberMeldung(t *testing.T) {
	fset := token.NewFileSet()
	gesehen := 0
	err := filepath.WalkDir("../..", func(pfad string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch d.Name() {
			case ".git", "node_modules", "frontend", "testdata":
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
		imPaket := datei.Name.Name == "service"
		ast.Inspect(datei, func(n ast.Node) bool {
			aufruf, ok := n.(*ast.CallExpr)
			if !ok || !istFmtErrorf(aufruf.Fun) {
				return true
			}
			gesehen++
			for _, arg := range aufruf.Args[1:] {
				if art := fehlerart(arg, imPaket); ohneBezeichnung[art] {
					t.Errorf("%s: fmt.Errorf wickelt %s ein — die Theke zeigte dann dessen Bezeichnung vor "+
						"der Meldung; meldung(%s, …) benutzen", fset.Position(aufruf.Pos()), art, art)
				}
			}
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if gesehen < 500 {
		t.Fatalf("nur %d fmt.Errorf-Aufrufe gesehen (am 28.09.2026: rund 690) — der Detektor läuft ins Leere", gesehen)
	}
}

func istFmtErrorf(fun ast.Expr) bool {
	sel, ok := fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "Errorf" {
		return false
	}
	paket, ok := sel.X.(*ast.Ident)
	return ok && paket.Name == "fmt"
}

// fehlerart nennt den Namen eines Sentinels aus diesem Paket, sonst "".
func fehlerart(e ast.Expr, imPaket bool) string {
	switch x := e.(type) {
	case *ast.Ident:
		if imPaket {
			return x.Name
		}
	case *ast.SelectorExpr:
		if paket, ok := x.X.(*ast.Ident); ok && paket.Name == "service" {
			return x.Sel.Name
		}
	}
	return ""
}
