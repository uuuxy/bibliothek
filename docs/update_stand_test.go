package docs

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// update.sh spielt am Schulserver nur Releases ein und folgt am Testserver main (OFFEN 5.31),
// mit frischem Datenbank-Image und frischen Basis-Images in beiden Fällen. Die Tests fahren das
// echte Skript in einem Wegwerf-Repo: drei Commits auf main, die ersten zwei als v1.0.0 und
// v1.1.0 getaggt, der Server auf dem ersten. docker ist ein Stellvertreter im PATH, der jeden
// Aufruf mitschreibt und nach dem Bau dessen Commit als den des laufenden Images meldet.

// updateLauf ist das Ergebnis eines Laufs: Ausgabe, Fehler, der Commit des Servers danach, die
// Aufrufe von docker und die Commits des Ursprungs (alt, Release v1.1.0, Spitze von main).
type updateLauf struct {
	ausgabe, kopf, docker string
	err                   error
	alt, release, spitze  string
}

func fahreUpdate(t *testing.T, envZeile string, mitTags, serverVorn bool) updateLauf {
	t.Helper()
	for _, werkzeug := range []string{"bash", "git", "gzip"} {
		if _, err := exec.LookPath(werkzeug); err != nil {
			t.Fatalf("%s fehlt: %v", werkzeug, err)
		}
	}
	skript := lies(t, "../update.sh")
	wurzel := t.TempDir()
	gitUmgebung := append(umgebungOhneGit(),
		"GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_NOSYSTEM=1",
		"GIT_AUTHOR_NAME=Probe", "GIT_AUTHOR_EMAIL=probe@example.invalid",
		"GIT_COMMITTER_NAME=Probe", "GIT_COMMITTER_EMAIL=probe@example.invalid")
	git := func(ordner string, argumente ...string) string {
		t.Helper()
		befehl := exec.Command("git", argumente...)
		befehl.Dir = ordner
		befehl.Env = gitUmgebung
		ausgabe, err := befehl.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v\n%s", argumente, err, ausgabe)
		}
		return strings.TrimSpace(string(ausgabe))
	}
	schreibe := func(pfad, inhalt string, modus os.FileMode) {
		t.Helper()
		if err := os.WriteFile(pfad, []byte(inhalt), modus); err != nil {
			t.Fatal(err)
		}
	}

	var lauf updateLauf
	quelle := filepath.Join(wurzel, "quelle.git")
	arbeit := filepath.Join(wurzel, "arbeit")
	server := filepath.Join(wurzel, "server")
	git(wurzel, "init", "-q", "--bare", "-b", "main", quelle)
	git(wurzel, "clone", "-q", quelle, arbeit)
	schreibe(filepath.Join(arbeit, "update.sh"), skript, 0o755)
	git(arbeit, "add", "update.sh")
	git(arbeit, "commit", "-qm", "alt")
	lauf.alt = git(arbeit, "rev-parse", "HEAD")
	if mitTags {
		git(arbeit, "tag", "v1.0.0")
	}
	git(arbeit, "push", "-q", "--tags", "origin", "HEAD:main")
	git(wurzel, "clone", "-q", quelle, server)

	for _, schritt := range []string{"release", "spitze"} {
		schreibe(filepath.Join(arbeit, schritt+".txt"), schritt+"\n", 0o644)
		git(arbeit, "add", schritt+".txt")
		git(arbeit, "commit", "-qm", schritt)
		if schritt == "release" {
			lauf.release = git(arbeit, "rev-parse", "HEAD")
			if mitTags {
				git(arbeit, "tag", "v1.1.0")
			}
		}
	}
	lauf.spitze = git(arbeit, "rev-parse", "HEAD")
	git(arbeit, "push", "-q", "--tags", "origin", "HEAD:main")
	if serverVorn {
		git(server, "pull", "-q")
	}
	if envZeile != "" {
		schreibe(filepath.Join(server, ".env"), envZeile+"\n", 0o600)
	}

	bin := filepath.Join(wurzel, "bin")
	if err := os.Mkdir(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	protokoll := filepath.Join(wurzel, "docker.log")
	gebaut := filepath.Join(wurzel, "gebaut")
	schreibe(filepath.Join(bin, "docker"), `#!/bin/sh
echo "$*" >> "`+protokoll+`"
case "$1" in
  ps) echo bibliothek-db ;;
  exec)
    case "$*" in
      *"printenv GIT_COMMIT"*) cat "`+gebaut+`" 2>/dev/null || echo alt ;;
      *pg_dump*) echo "-- Probe-Dump" ;;
    esac ;;
  inspect) echo healthy ;;
  compose)
    case "$*" in *build*) printf '%s' "$GIT_COMMIT" > "`+gebaut+`" ;; esac ;;
esac
exit 0
`, 0o755)

	befehl := exec.Command("bash", filepath.Join(server, "update.sh"))
	befehl.Dir = server
	befehl.Env = append(gitUmgebung, "PATH="+bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	roh, err := befehl.CombinedOutput()
	lauf.ausgabe, lauf.err = string(roh), err
	lauf.kopf = git(server, "rev-parse", "HEAD")
	if aufrufe, lesefehler := os.ReadFile(protokoll); lesefehler == nil {
		lauf.docker = string(aufrufe)
	}
	return lauf
}

// pruefeFrischeImages: das Datenbank-Image geholt, gebaut mit --pull, dann gestartet, in dieser
// Reihenfolge.
func pruefeFrischeImages(t *testing.T, lauf updateLauf) {
	t.Helper()
	letzte := -1
	for _, schritt := range []string{"pull postgres-db", "build --pull", "up -d"} {
		stelle := strings.Index(lauf.docker, "docker-compose.yml "+schritt)
		if stelle < 0 || stelle < letzte {
			t.Errorf("docker compose %s fehlt oder steht außer der Reihe:\n%s", schritt, lauf.docker)
		}
		letzte = stelle
	}
}

func TestUpdate_ReleaseNimmtDasNeuesteTagStattMain(t *testing.T) {
	lauf := fahreUpdate(t, "UPDATE_STAND=release", true, false)
	if lauf.err != nil {
		t.Fatalf("update.sh: %v\n%s", lauf.err, lauf.ausgabe)
	}
	if lauf.kopf != lauf.release {
		t.Errorf("Server steht auf %s, erwartet v1.1.0 (%s), nicht die Spitze von main (%s)", lauf.kopf, lauf.release, lauf.spitze)
	}
	if !strings.Contains(lauf.ausgabe, "Stand: Release v1.1.0") {
		t.Errorf("die Ausgabe nennt das Release nicht:\n%s", lauf.ausgabe)
	}
	pruefeFrischeImages(t, lauf)
}

func TestUpdate_OhneEintragFolgtMain(t *testing.T) {
	lauf := fahreUpdate(t, "", true, false)
	if lauf.err != nil {
		t.Fatalf("update.sh: %v\n%s", lauf.err, lauf.ausgabe)
	}
	if lauf.kopf != lauf.spitze {
		t.Errorf("Server steht auf %s, erwartet die Spitze von main (%s)", lauf.kopf, lauf.spitze)
	}
	pruefeFrischeImages(t, lauf)
}

// Ohne Release bricht das Update ab, bevor es etwas tut: keine Sicherung, kein Wechsel, kein Bau.
func TestUpdate_OhneReleaseBrichtVorDerSicherungAb(t *testing.T) {
	lauf := fahreUpdate(t, "UPDATE_STAND=release", false, false)
	if lauf.err == nil || !strings.Contains(lauf.ausgabe, "Kein Release gefunden") {
		t.Fatalf("update.sh hätte ohne Release abbrechen sollen (err=%v):\n%s", lauf.err, lauf.ausgabe)
	}
	if lauf.kopf != lauf.alt {
		t.Errorf("Server steht auf %s, erwartet unverändert %s", lauf.kopf, lauf.alt)
	}
	if strings.Contains(lauf.docker, "pg_dump") || strings.Contains(lauf.docker, "compose") {
		t.Errorf("vor dem Abbruch lief schon etwas:\n%s", lauf.docker)
	}
}

// Ein Tippfehler im Wert fällt nicht still auf main zurück; am Schulserver spielte das den
// ungeprüften Zweig ein.
func TestUpdate_UnbekannterStandBrichtAb(t *testing.T) {
	lauf := fahreUpdate(t, "UPDATE_STAND=Release", true, false)
	if lauf.err == nil || !strings.Contains(lauf.ausgabe, "UPDATE_STAND=Release in der .env ist unbekannt") {
		t.Fatalf("update.sh hätte den Wert ablehnen sollen (err=%v):\n%s", lauf.err, lauf.ausgabe)
	}
	if lauf.kopf != lauf.alt || strings.Contains(lauf.docker, "pg_dump") {
		t.Errorf("vor dem Abbruch lief schon etwas (Server %s, docker: %s)", lauf.kopf, lauf.docker)
	}
}

// Steht der Server schon weiter als das neueste Release, wechselt das Update nicht zurück:
// Migrationen laufen nur vorwärts. Es bricht ab, bevor es etwas tut.
func TestUpdate_ReleaseHinterDemStandBrichtAb(t *testing.T) {
	lauf := fahreUpdate(t, "UPDATE_STAND=release", true, true)
	if lauf.err == nil || !strings.Contains(lauf.ausgabe, "liegt nicht nach dem Stand im Arbeitsverzeichnis") {
		t.Fatalf("update.sh hätte den Rückschritt ablehnen sollen (err=%v):\n%s", lauf.err, lauf.ausgabe)
	}
	if lauf.kopf != lauf.spitze || strings.Contains(lauf.docker, "pg_dump") {
		t.Errorf("vor dem Abbruch lief schon etwas (Server %s, docker: %s)", lauf.kopf, lauf.docker)
	}
}
