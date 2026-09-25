package docs

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Rasterfrage 10 (Rückweg) an den beiden Anleitungen, die man im Ernstfall abtippt.
//
// Bestands-Durchgang 10.09.2026, zwei Funde derselben Art — Anleitungen, die niemand
// prüfte, weil sie Text sind:
//
//  1. Die Restore-Anleitung wählte `ls -t backups/backup_*.sql.gz.enc` auf dem Host. Das
//     nächtliche Backup liegt aber im benannten Volume (/app/backups im Container,
//     jobs/backup.go), und in ./backups auf dem Host liegt nur die Vorab-Sicherung von
//     update.sh — unter DEMSELBEN Muster backup_<Zeit>.sql.gz.enc. Ein Restore nach
//     Anleitung spielte die letzte Deploy-Sicherung ein; alles seit dem letzten Deploy
//     war still weg. Auch die „manuelle Restore-Probe vor Go-Live" prüfte diese Datei.
//  2. Die Rollback-Anleitung von update.sh riet `git stash` (nach einem sauberen Pull
//     wirkungslos — der neue Code bleibt), spielte den Dump ohne dropdb und ohne
//     ON_ERROR_STOP in die schon migrierte Datenbank (psql endet trotz Fehlerflut mit 0)
//     und baute danach wieder den neuen Code.
//
// Neue Bugklasse „Zwei Ablagen, ein Dateiname": Zwei Erzeuger legen gleich benannte
// Artefakte an verschiedenen Orten ab; wer nach dem Muster statt nach der Herkunft
// wählt, greift die falsche Datei.
func TestRueckweg_RestoreNimmtDasNachtbackup(t *testing.T) {
	anleitung := lies(t, "resilience_and_recovery.md")
	if strings.Contains(anleitung, "ls -t backups/backup_") {
		t.Error("die Restore-Anleitung wählt das Backup per Host-Glob backups/backup_* — dort liegt nur die " +
			"Vorab-Sicherung von update.sh; das Nachtbackup liegt im Volume /app/backups des Containers")
	}
	skript := lies(t, "../update.sh")
	if strings.Contains(skript, `backup_${TIMESTAMP}`) {
		t.Error("update.sh benennt seine Vorab-Sicherung nach dem Muster des Nachtbackups (backup_<Zeit>) — " +
			"zwei Ablagen, ein Dateiname; die Vorab-Sicherung braucht ein eigenes Präfix")
	}
	nacht := lies(t, "../jobs/backup.go")
	if !strings.Contains(nacht, `"backup_%s.sql.gz.enc"`) {
		t.Fatal("jobs/backup.go schreibt das Nachtbackup nicht mehr unter dem Muster backup_<Zeit>.sql.gz.enc — Gate nachziehen")
	}
}

func TestRueckweg_RollbackFuehrtZurueck(t *testing.T) {
	skript := lies(t, "../update.sh")
	start := strings.Index(skript, "print_rollback_instructions() {")
	if start < 0 {
		t.Fatal("print_rollback_instructions nicht gefunden — Gate nachziehen")
	}
	ende := strings.Index(skript[start:], "\n}\n")
	if ende < 0 {
		t.Fatal("Ende von print_rollback_instructions nicht gefunden")
	}
	// Nur was GEDRUCKT wird zählt — die echo-Zeilen. Der erklärende Kommentar im Block
	// nennt das alte `git stash` beim Namen und hätte das Gate sonst selbst getroffen
	// (Memory „Lügende Ratsche durch Kommentar"; beim Bau dieses Gates genau so passiert).
	var gedruckt []string
	for _, zeile := range strings.Split(skript[start:start+ende], "\n") {
		if strings.Contains(zeile, "echo") {
			gedruckt = append(gedruckt, zeile)
		}
	}
	block := strings.Join(gedruckt, "\n")
	for _, muss := range []string{"git reset --hard", "dropdb", "createdb", "ON_ERROR_STOP=1"} {
		if !strings.Contains(block, muss) {
			t.Errorf("Rollback-Anleitung ohne %q — sie führt nicht auf den alten Stand zurück", muss)
		}
	}
	if strings.Contains(block, "git stash") {
		t.Error("Rollback-Anleitung rät `git stash` — nach einem sauberen Pull wirkungslos")
	}
}

// Der Rückweg zeigt auf den Stand, der VOR dem Update lief — auch wenn `git pull` vorher
// von Hand lief, wie DEPLOYMENT.md §2.4 es verlangt.
//
// Nachgestellt am 25.09.2026 (Durchsicht des Pflegekonzepts): update.sh las den Rückweg als
// `git rev-parse HEAD` unmittelbar vor seinem eigenen Pull. Nach dem vorgezogenen Pull ist
// das schon der NEUE Stand: Die Anleitung druckte `git reset --hard <neu>` und baute in
// ihrem Schritt 4 wieder den neuen Code. ORIG_HEAD taugt ebenso wenig — der zweite Pull,
// der nichts mehr findet, setzt es auf den neuen Stand.
//
// Der Test fährt das echte Skript: eine Kopie in einem Wegwerf-Repo, der Pull von Hand
// vorab, `docker` als Stellvertreter im PATH. Er meldet für das laufende Image den alten
// Commit und lässt den Bau in Schritt 3 scheitern. Geprüft wird die gedruckte Zeile.
func TestRueckweg_RollbackNenntDenLaufendenStand(t *testing.T) {
	for _, werkzeug := range []string{"bash", "git", "gzip"} {
		if _, err := exec.LookPath(werkzeug); err != nil {
			t.Fatalf("%s fehlt: %v", werkzeug, err)
		}
	}
	skript := lies(t, "../update.sh")
	wurzel := t.TempDir()
	gitUmgebung := append(os.Environ(),
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

	quelle := filepath.Join(wurzel, "quelle.git")
	arbeit := filepath.Join(wurzel, "arbeit")
	server := filepath.Join(wurzel, "server")
	git(wurzel, "init", "-q", "--bare", "-b", "main", quelle)
	git(wurzel, "clone", "-q", quelle, arbeit)
	schreibe(filepath.Join(arbeit, "update.sh"), skript, 0o755)
	git(arbeit, "add", "update.sh")
	git(arbeit, "commit", "-qm", "alt")
	git(arbeit, "push", "-q", "origin", "HEAD:main")
	alt := git(arbeit, "rev-parse", "HEAD")
	git(wurzel, "clone", "-q", quelle, server)

	schreibe(filepath.Join(arbeit, "neu.txt"), "neu\n", 0o644)
	git(arbeit, "add", "neu.txt")
	git(arbeit, "commit", "-qm", "neu")
	git(arbeit, "push", "-q", "origin", "HEAD:main")
	neu := git(arbeit, "rev-parse", "HEAD")
	git(server, "pull", "-q") // der vorgezogene Pull aus DEPLOYMENT.md §2.4

	bin := filepath.Join(wurzel, "bin")
	if err := os.Mkdir(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	schreibe(filepath.Join(bin, "docker"), `#!/bin/sh
case "$1" in
  ps) echo bibliothek-db ;;
  exec)
    case "$*" in
      *"printenv GIT_COMMIT"*) echo `+alt+` ;;
      *pg_dump*) echo "-- Probe-Dump" ;;
    esac ;;
  compose) echo "Bau gescheitert (Probe)" >&2; exit 1 ;;
esac
exit 0
`, 0o755)

	lauf := exec.Command("bash", filepath.Join(server, "update.sh"))
	lauf.Dir = server
	lauf.Env = append(gitUmgebung, "PATH="+bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	roh, err := lauf.CombinedOutput()
	ausgabe := string(roh)
	// Positivkontrolle: Ohne den gescheiterten Bau gäbe es keine Anleitung, und die
	// Prüfungen unten wären grün, ohne etwas gesehen zu haben.
	if err == nil || !strings.Contains(ausgabe, "ROLLBACK-ANLEITUNG") {
		t.Fatalf("update.sh hätte in Schritt 3 mit der Rollback-Anleitung abbrechen sollen (err=%v):\n%s", err, ausgabe)
	}
	var reset []string
	for _, zeile := range strings.Split(ausgabe, "\n") {
		if strings.Contains(zeile, "git reset --hard") {
			reset = append(reset, strings.TrimSpace(zeile))
		}
	}
	if len(reset) != 1 || reset[0] != "git reset --hard "+alt {
		t.Errorf("Die Rollback-Anleitung führt nicht auf den laufenden Stand zurück.\n"+
			"  laufend (alt): %s\n  neu:           %s\n  gedruckt:      %q\n"+
			"→ update.sh muss den Rückweg aus dem laufenden Image lesen (GIT_COMMIT), nicht aus HEAD.",
			alt, neu, reset)
	}
}

func lies(t *testing.T, pfad string) string {
	t.Helper()
	b, err := os.ReadFile(pfad)
	if err != nil {
		t.Fatalf("%s lesen: %v", pfad, err)
	}
	return string(b)
}
