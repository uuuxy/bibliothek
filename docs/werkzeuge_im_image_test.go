package docs

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// Gate: Jedes Werkzeug, das eine Anleitung IM CONTAINER aufruft, liegt auch im Image.
//
// Anlass (Register, Bestands-Durchgang 10.09.2026): docs/SCRIPTS.md nennt seit jeher
// `docker compose exec backend ./migrate-fotos` — das Dockerfile baute das Werkzeug aber
// nie, und der Schulserver hat kein Go. Statt dessen lag ein 14 MB großes Binary IM
// REPO, gebaut irgendwann auf irgendeinem Rechner. Beide Hälften sind Symptome desselben
// Lochs: Die Anleitung verspricht einen Weg, den das Image nicht kennt.
//
// Es ist nicht das erste Mal. Der Dockerfile-Kommentar über rotate-encryption-key hält
// denselben Fall vom 06.08.2026 fest („Ein Werkzeug, das man genau dort nicht starten
// kann, wo man es braucht, ist keines"), und restore-backup stand am 22.08. in derselben
// Lage. Dreimal dieselbe Klasse ist eine Ratsche wert.
//
// Reparatur bei Rot: Entweder das Werkzeug ins Image bauen (RUN … go build + COPY --from)
// oder die Anleitung auf den Weg ändern, den es wirklich gibt.
var werkzeugAufruf = regexp.MustCompile(`(?:compose exec|exec)(?: -[a-zA-Z]+)* (?:backend|bibliothek-backend) \./([a-zA-Z0-9_-]+)`)

func TestWerkzeugeDerAnleitungenLiegenImImage(t *testing.T) {
	dockerfile := lies(t, "../Dockerfile")

	quellen := map[string]string{}
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
			for _, m := range werkzeugAufruf.FindAllStringSubmatch(string(inhalt), -1) {
				if _, schon := quellen[m[1]]; !schon {
					quellen[m[1]] = pfad
				}
			}
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
