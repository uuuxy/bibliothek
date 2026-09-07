package api

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// Jeder Schreibweg des LMF-Plans muss seine Änderung melden.
//
// Peter, 07.09.2026: „änderungen soll und kann man jederzeit eintragen und man sieht es
// live!" Getragen wird das von einem SSE-Ereignis (lmf_plan_live.go), das Portal und
// Planer zum Nachholen bringt. Ein Schreibweg, der es vergisst, fällt nicht auf: Die
// Antwort an den Aufrufer ist richtig, sein eigener Bildschirm zeigt den neuen Stand,
// und nur die ANDEREN — das Lehrerzimmer, der zweite Arbeitsplatz — bleiben auf dem
// alten. Das merkt niemand, bis jemand nach einem Termin fragt, den es so nicht mehr
// gibt.
//
// Deshalb an den ROUTEN aufgehängt, nicht an einer gepflegten Namensliste: Die Prüfung
// liest, welche Handler hinter einem schreibenden /api/lmf-plan hängen, und verlangt von
// jedem den Aufruf. Eine neue Route wächst damit von selbst in dieses Gate hinein.
func TestJederLmfPlanSchreibwegMeldetSeineAenderung(t *testing.T) {
	handler := lmfPlanSchreibHandler(t)
	if len(handler) < 3 {
		t.Fatalf("nur %d schreibende LMF-Plan-Routen gefunden (%v) — erwartet mindestens 3 "+
			"(speichern, verwerfen, veröffentlichen). Ist routes_books.go umformuliert? "+
			"Dann greift dieses Gate nicht mehr.", len(handler), handler)
	}

	// Datei für Datei statt parser.ParseDir: Das ist seit Go 1.25 abgekündigt (es
	// beachtet keine Build-Tags), und die anderen AST-Ratschen dieses Repos lesen
	// ebenfalls einzelne Dateien.
	pfade, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("api-Dateien auflisten: %v", err)
	}
	fset := token.NewFileSet()
	meldet := map[string]bool{}
	for _, pfad := range pfade {
		datei, err := parser.ParseFile(fset, pfad, nil, 0)
		if err != nil {
			t.Fatalf("%s parsen: %v", pfad, err)
		}
		for _, dekl := range datei.Decls {
			fn, ok := dekl.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}
			if ruftAuf(fn.Body, "meldeLmfPlanGeaendert") {
				meldet[fn.Name.Name] = true
			}
		}
	}

	for _, name := range handler {
		if !meldet[name] {
			t.Errorf("%s schreibt den LMF-Plan, meldet die Änderung aber nicht — offene "+
				"Portal-Seiten und der zweite Arbeitsplatz bleiben auf dem alten Stand. "+
				"`s.meldeLmfPlanGeaendert()` nach dem erfolgreichen Schreiben aufrufen.", name)
		}
	}
}

// lmfPlanSchreibHandler liest aus routes_books.go die Handler-Namen hinter schreibenden
// /api/lmf-plan-Routen (PUT, POST, DELETE, PATCH).
func lmfPlanSchreibHandler(t *testing.T) []string {
	t.Helper()
	inhalt, err := os.ReadFile("routes_books.go")
	if err != nil {
		t.Fatalf("routes_books.go lesen: %v", err)
	}
	// Zwei Schritte statt eines Musters: Die Zeile trägt zwei Aufrufe auf `s` — erst
	// RequirePermission, dann den Handler. Ein einziger Ausdruck fände nur den ersten.
	istSchreibroute := regexp.MustCompile(`mux\.Handle\("(?:PUT|POST|DELETE|PATCH) /api/lmf-plan`)
	aufrufe := regexp.MustCompile(`s\.([A-Za-z0-9_]+)\(`)
	gefunden := map[string]bool{}
	for _, zeile := range strings.Split(string(inhalt), "\n") {
		if !istSchreibroute.MatchString(zeile) {
			continue
		}
		for _, treffer := range aufrufe.FindAllStringSubmatch(zeile, -1) {
			if treffer[1] != "RequirePermission" {
				gefunden[treffer[1]] = true
			}
		}
	}
	namen := make([]string, 0, len(gefunden))
	for n := range gefunden {
		namen = append(namen, n)
	}
	sort.Strings(namen)
	return namen
}

// ruftAuf sagt, ob im Rumpf irgendwo `name(...)` steht — als Methode auf einem Empfänger
// (s.name()) oder frei.
func ruftAuf(rumpf *ast.BlockStmt, name string) bool {
	gefunden := false
	ast.Inspect(rumpf, func(n ast.Node) bool {
		aufruf, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		switch f := aufruf.Fun.(type) {
		case *ast.SelectorExpr:
			if f.Sel.Name == name {
				gefunden = true
			}
		case *ast.Ident:
			if f.Name == name {
				gefunden = true
			}
		}
		return !gefunden
	})
	return gefunden
}
