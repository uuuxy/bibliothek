package docs

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Gate gegen eine veraltete Swagger-Datei.
//
// Anlass (05.08.2026): docs.go war generiert worden, als es 23 annotierte Endpunkte gab,
// und danach nie wieder. Im Code standen inzwischen 49 `@Router`-Annotationen, und
// umgekehrt beschrieb die Datei mit `/transactions/recent` einen Endpunkt, den es nicht
// mehr gab. Beides fiel niemandem auf, weil eine generierte Datei aussieht, als pflege
// sie sich selbst.
//
// Das Gate prüft beide Richtungen. Es prüft NICHT, ob jeder Endpunkt annotiert ist — das
// wäre eine andere Entscheidung (das vollständige Verzeichnis aller Routen steht in
// api_inventar.md). Es hält nur fest: Was annotiert ist, muss auch in der ausgelieferten
// Datei stehen, und was dort steht, muss es noch geben.
//
// Reparatur bei Rot: `swag init -g main.go -o docs` und das Ergebnis mitcommitten.
func TestSwaggerNichtVeraltet(t *testing.T) {
	annotationen := sammleRouterAnnotationen(t)
	if len(annotationen) < 20 {
		t.Fatalf("nur %d @Router-Annotationen gefunden — der Scanner greift vermutlich nicht mehr", len(annotationen))
	}

	rohdaten, err := os.ReadFile("docs.go")
	if err != nil {
		t.Fatalf("docs.go lesen: %v", err)
	}
	generiert := sammleGenerierteePfade(string(rohdaten))

	for pfad := range annotationen {
		if _, ok := generiert[pfad]; !ok {
			t.Errorf("Endpunkt %q ist annotiert, fehlt aber in docs.go — die Datei ist veraltet.\n"+
				"→ `swag init -g main.go -o docs` ausführen und das Ergebnis mitcommitten.", pfad)
		}
	}
	for pfad := range generiert {
		if _, ok := annotationen[pfad]; !ok {
			t.Errorf("docs.go beschreibt %q, wozu es keine @Router-Annotation (mehr) gibt — "+
				"die Doku beschreibt einen Endpunkt, den es nicht mehr gibt.\n"+
				"→ `swag init -g main.go -o docs` ausführen.", pfad)
		}
	}
}

var routerAnnotation = regexp.MustCompile(`//\s*@Router\s+(\S+)`)

// sammleRouterAnnotationen liest alle @Router-Pfade aus dem Quelltext des Projekts.
func sammleRouterAnnotationen(t *testing.T) map[string]bool {
	t.Helper()
	pfade := map[string]bool{}

	wurzel := ".."
	err := filepath.WalkDir(wurzel, func(pfad string, eintrag os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		// node_modules und .git enthalten keine Go-Handler, kosten aber Laufzeit.
		if eintrag.IsDir() {
			name := eintrag.Name()
			if name == "node_modules" || name == ".git" || name == "dist" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(pfad, ".go") || strings.HasSuffix(pfad, "docs.go") {
			return nil
		}
		inhalt, err := os.ReadFile(pfad)
		if err != nil {
			return err
		}
		for _, treffer := range routerAnnotation.FindAllStringSubmatch(string(inhalt), -1) {
			pfade[treffer[1]] = true
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Quelltext durchsuchen: %v", err)
	}
	return pfade
}

// sammleGenerierteePfade liest die Pfad-Schlüssel aus dem "paths"-Block von docs.go.
// Sie stehen dort als einzige Einträge mit genau acht führenden Leerzeichen.
func sammleGenerierteePfade(inhalt string) map[string]bool {
	pfade := map[string]bool{}
	schluessel := regexp.MustCompile(`(?m)^        "(/[^"]*)": \{`)
	for _, treffer := range schluessel.FindAllStringSubmatch(inhalt, -1) {
		pfade[treffer[1]] = true
	}
	return pfade
}
