package docs

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// Gate: Jedes Werkzeug, das eine Anleitung im Container aufruft, liegt auch im Image. Der
// Schulserver hat kein Go; verspricht die Anleitung einen Weg, den das Image nicht kennt,
// steht man im Ernstfall ohne Werkzeug da: „Ein Werkzeug, das man genau dort nicht starten
// kann, wo man es braucht, ist keines" (Kommentar im Dockerfile über rotate-encryption-key).
//
// Reparatur bei Rot: Entweder das Werkzeug ins Image bauen (RUN … go build + COPY --from)
// oder die Anleitung auf den Weg ändern, den es wirklich gibt.
//
// Blindheit: Ein Aufruf in einer anderen Form als den beiden unten (etwa über eine andere
// Variable für den Container) bleibt ungeprüft.
//
// werkzeugAufrufe sind die beiden Formen, in denen Anleitungen und Skripte ein Werkzeug im
// Container starten: exec am laufenden Container, in den Skripten über ${BACKEND_CONTAINER},
// und ein Wegwerf-Container mit eigenem Einstieg (compose run --entrypoint), den der
// Totalverlust und die Meldungen der Skripte nennen.
var werkzeugAufrufe = []*regexp.Regexp{
	regexp.MustCompile(`(?:compose exec|exec)(?: -[a-zA-Z]+)* (?:backend|bibliothek-backend|"\$\{BACKEND_CONTAINER\}") \./([a-zA-Z0-9_-]+)`),
	regexp.MustCompile(`--entrypoint \./([a-zA-Z0-9_-]+) backend`),
}

func TestWerkzeugeDerAnleitungenLiegenImImage(t *testing.T) {
	dockerfile := lies(t, "../Dockerfile")

	quellen := map[string]string{}
	trefferJeForm := make([]int, len(werkzeugAufrufe))
	for _, muster := range []string{"*.md", "../*.sh", "../scripts/*.sh"} {
		treffer, err := filepath.Glob(muster)
		if err != nil {
			t.Fatalf("Glob %q: %v", muster, err)
		}
		for _, pfad := range treffer {
			inhalt, err := os.ReadFile(pfad) //nolint:gosec // Testdateien aus dem Repo
			if err != nil {
				t.Fatalf("%s lesen: %v", pfad, err)
			}
			for i, form := range werkzeugAufrufe {
				for _, m := range form.FindAllStringSubmatch(string(inhalt), -1) {
					trefferJeForm[i]++
					if _, schon := quellen[m[1]]; !schon {
						quellen[m[1]] = pfad
					}
				}
			}
		}
	}
	for i, n := range trefferJeForm {
		if n == 0 {
			t.Fatalf("die Form %q findet keinen Aufruf — greift sie ins Leere, prüft dieses Gate einen Weg nicht mehr",
				werkzeugAufrufe[i])
		}
	}
	if len(quellen) < 3 {
		t.Fatalf("nur %d Werkzeug-Aufrufe gefunden — der Sammler greift ins Leere, dieses Gate wäre still grün", len(quellen))
	}

	namen := make([]string, 0, len(quellen))
	for name := range quellen {
		namen = append(namen, name)
	}
	sort.Strings(namen)

	for _, name := range namen {
		// `main` ist der Server selbst; er kommt über dieselbe COPY-Zeile ins Image.
		if !strings.Contains(dockerfile, "COPY --from=backend-builder /app/"+name+" ") {
			t.Errorf("%s ruft `./%s` im Container auf, das Dockerfile kopiert es aber nicht ins Image — "+
				"entweder bauen (RUN … go build -o %s ./cmd/%s + COPY --from) oder die Anleitung ändern",
				quellen[name], name, name, name)
		}
	}
}
