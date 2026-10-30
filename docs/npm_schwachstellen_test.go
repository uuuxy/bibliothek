package docs

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// npm audit hat eine Regel, und sie steht in scripts/npm-audit-gate.sh.
//
// Die Sicherheitsprüfung der CI wertet nur HIGH und CRITICAL im ausgelieferten Programm. Ein Hook
// mit eigenem Aufruf sagte nicht voraus, was sie rot macht, und warnte über das, was sie bewusst
// nicht wertet. Beide rufen deshalb das Skript, und keiner führt npm audit selbst.
//
// Sieht nicht: einen dritten Ort, der npm audit aufruft (er steht nicht in der Liste), und einen
// Aufruf, der anders geschrieben ist als in den Mustern, etwa über eine Variable.
func TestNpmSchwachstellen_EineRegelFuerCIUndHook(t *testing.T) {
	if !strings.Contains(ohneShellKommentare(t, "../scripts/npm-audit-gate.sh"), "npm audit --audit-level=high --omit=dev") {
		t.Errorf("scripts/npm-audit-gate.sh ruft npm audit nicht mehr mit --audit-level=high --omit=dev — " +
			"die Regel der Sicherheitsprüfung steht nur dort")
	}
	aufrufe := []struct {
		pfad   string
		aufruf *regexp.Regexp
	}{
		{"../scripts/git-hooks/pre-push", regexp.MustCompile(`(?m)^\s*\./scripts/npm-audit-gate\.sh\b`)},
		{"../.github/workflows/security-scan.yml", regexp.MustCompile(`(?m)^\s*VERSUCHE=3 \.\./scripts/npm-audit-gate\.sh\b`)},
	}
	// Ein Aufruf als Befehl, nicht als Wort in einer Meldung („Führe npm audit … aus").
	selbst := regexp.MustCompile(`(?m)(^|\$\(|\bif\s+|&&|\|\|)\s*npm audit\b`)
	for _, a := range aufrufe {
		text := ohneShellKommentare(t, a.pfad)
		if !a.aufruf.MatchString(text) {
			t.Errorf("%s ruft scripts/npm-audit-gate.sh nicht (mehr) auf", a.pfad)
		}
		if selbst.MatchString(text) {
			t.Errorf("%s führt npm audit wieder selbst aus — die Regel steht einmal, in "+
				"scripts/npm-audit-gate.sh", a.pfad)
		}
	}
}

// Das Gate kennt drei Ausgänge, und Hook wie CI hängen an den Zahlen: 0 kein Befund, 1 ein Befund
// HIGH oder CRITICAL, 2 der Audit-Endpunkt war in allen Anläufen nicht erreichbar. Ein nachgestelltes
// npm zählt die Anläufe und meldet die ersten FAKE_AUSFAELLE als Ausfall der Registry.
//
// Sieht nicht: ob das echte npm einen Befund noch mit Exit 1 meldet und einen Ausfall noch mit
// einer der Meldungen, die das Skript kennt; das zeigt erst ein Lauf gegen die Registry.
func TestNpmSchwachstellen_GateKenntDreiAusgaenge(t *testing.T) {
	ordner := t.TempDir()
	nachgestellt := `#!/bin/sh
echo "$*" >> "$AUFRUFE"
if [ "$(wc -l < "$AUFRUFE")" -le "$FAKE_AUSFAELLE" ]; then
	echo "npm error audit endpoint returned an error"
	exit 1
fi
printf '%s\n' "$FAKE_AUSGABE"
exit "$FAKE_STATUS"
`
	if err := os.WriteFile(filepath.Join(ordner, "npm"), []byte(nachgestellt), 0o700); err != nil { //nolint:gosec // ausführbar ist der Zweck
		t.Fatal(err)
	}
	faelle := []struct {
		name                              string
		ausfaelle, status                 int
		ausgabe, versuche                 string
		erwarteterExit, erwarteteAnlaeufe int
	}{
		{"kein Befund", 0, 0, "found 0 vulnerabilities", "1", 0, 1},
		{"Befund HIGH", 0, 1, "1 high severity vulnerability", "1", 1, 1},
		{"Endpunkt nicht erreichbar", 1, 0, "found 0 vulnerabilities", "1", 2, 1},
		{"Endpunkt kurz weg, zweiter Anlauf ohne Befund", 1, 0, "found 0 vulnerabilities", "3", 0, 2},
		{"Endpunkt in allen Anläufen weg", 3, 0, "", "3", 2, 3},
		{"Befund nach einem Ausfall", 1, 1, "1 critical severity vulnerability", "3", 1, 2},
	}
	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			aufrufe := filepath.Join(t.TempDir(), "aufrufe")
			befehl := exec.Command("bash", "../scripts/npm-audit-gate.sh")
			befehl.Env = append(os.Environ(),
				"PATH="+ordner+string(os.PathListSeparator)+os.Getenv("PATH"),
				"AUFRUFE="+aufrufe,
				"FAKE_AUSFAELLE="+strconv.Itoa(f.ausfaelle),
				"FAKE_STATUS="+strconv.Itoa(f.status),
				"FAKE_AUSGABE="+f.ausgabe,
				"VERSUCHE="+f.versuche,
				"NPM_AUDIT_PAUSE=0",
			)
			ausgabe, err := befehl.CombinedOutput()
			exit := 0
			var ende *exec.ExitError
			if errors.As(err, &ende) {
				exit = ende.ExitCode()
			} else if err != nil {
				t.Fatalf("Gate starten: %v", err)
			}
			if exit != f.erwarteterExit {
				t.Errorf("Exit %d, erwartet %d; Ausgabe:\n%s", exit, f.erwarteterExit, ausgabe)
			}
			roh, err := os.ReadFile(aufrufe) //nolint:gosec // Datei aus t.TempDir
			if err != nil {
				t.Fatalf("das nachgestellte npm wurde nicht aufgerufen: %v", err)
			}
			anlaeufe := strings.Split(strings.TrimSpace(string(roh)), "\n")
			if len(anlaeufe) != f.erwarteteAnlaeufe {
				t.Errorf("%d Anläufe, erwartet %d", len(anlaeufe), f.erwarteteAnlaeufe)
			}
			for _, a := range anlaeufe {
				if a != "audit --audit-level=high --omit=dev" {
					t.Errorf("npm gerufen mit %q, erwartet „audit --audit-level=high --omit=dev“", a)
				}
			}
		})
	}
}

// ohneShellKommentare liest eine Datei ohne ihre Kommentarzeilen: Ein Aufruf, der nur noch im
// Kommentar steht, prüft nichts.
func ohneShellKommentare(t *testing.T, pfad string) string {
	t.Helper()
	inhalt, err := os.ReadFile(pfad) //nolint:gosec // Dateien aus dem Repo
	if err != nil {
		t.Fatalf("%s lesen: %v", pfad, err)
	}
	var zeilen []string
	for _, zeile := range strings.Split(string(inhalt), "\n") {
		if !strings.HasPrefix(strings.TrimSpace(zeile), "#") {
			zeilen = append(zeilen, zeile)
		}
	}
	return strings.Join(zeilen, "\n")
}
