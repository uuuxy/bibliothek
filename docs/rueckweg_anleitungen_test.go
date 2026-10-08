package docs

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
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
	// (Bugklasse „Lügende Ratsche", docs/sweeps.md; beim Bau dieses Gates genau so passiert).
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

// Jede Zeile der Wiederherstellungs-Anleitung, die einen Dump einspielt, bricht beim ersten
// Fehler ab. Ohne ON_ERROR_STOP endet psql trotz Fehlerflut mit 0 — so lief der Rückweg von
// update.sh bis zum 10.09.2026 und die Anleitung für den Ernstfall bis zum 28.09.2026, an vier
// Stellen (2a, 2b, 2c, 2e; gefunden in der Generalprobe).
//
// Blindheit: Erkannt wird `psql … -f …` und `| psql` auf einer Zeile, auch hinter
// `docker compose exec`, nur in dieser Datei. Ein Einspielen mit `<` oder über mehrere Zeilen
// mit dem psql-Aufruf vor dem Umbruch sieht der Test nicht.
func TestRueckweg_EinspielenBrichtBeimErstenFehlerAb(t *testing.T) {
	einspielen := regexp.MustCompile(`psql\b.*\s-f\s|\|\s*(?:docker compose exec(?:\s+-T)?\s+\S+\s+)?psql\b`)
	gesehen := 0
	for i, zeile := range strings.Split(lies(t, "resilience_and_recovery.md"), "\n") {
		if !einspielen.MatchString(zeile) {
			continue
		}
		gesehen++
		if !strings.Contains(zeile, "-v ON_ERROR_STOP=1") {
			t.Errorf("resilience_and_recovery.md:%d spielt ohne ON_ERROR_STOP ein — psql endet dann trotz Fehler mit 0: %s",
				i+1, strings.TrimSpace(zeile))
		}
	}
	if gesehen < 5 {
		t.Fatalf("nur %d Einspiel-Zeilen gefunden, erwartet mindestens 5 (2a, 2b, 2c, 2e, 2f) — der Detektor sieht nichts", gesehen)
	}
}

// Beim Totalverlust (2f) kommt das Werkzeug aus dem Image des Backends, und seine Ausgabe geht
// in die Datenbank. Baut erst dieser Aufruf das Image, läuft die Ausgabe des Baus mit hinein,
// und das Einspielen bricht an ihrer ersten Zeile ab (so geschehen beim Durchspielen am
// 08.10.2026). Deshalb steht der Bau als eigener Schritt davor.
func TestRueckweg_TotalverlustBautVorDemEinspielen(t *testing.T) {
	anleitung := lies(t, "resilience_and_recovery.md")
	einspielen := strings.Index(anleitung, "docker compose run --rm --no-deps -T --entrypoint ./restore-backup backend")
	if einspielen < 0 {
		t.Fatal("die Einspiel-Zeile des Totalverlusts (docker compose run … restore-backup) fehlt — Anleitung oder Gate nachziehen")
	}
	bau := strings.Index(anleitung, "\ndocker compose build backend\n")
	if bau < 0 || bau > einspielen {
		t.Error("vor dem Einspielen beim Totalverlust fehlt der eigene Schritt `docker compose build backend` — " +
			"baut erst der Einspiel-Aufruf das Image, läuft die Ausgabe des Baus in die Datenbank")
	}
}

// Was ein Skript als Befehl druckt, wird abgetippt. Ein Platzhalter in spitzen Klammern darin
// lässt sich wörtlich einfügen (so am 06.08.2026 mit einer Anleitung geschehen), und der Hinweis
// zum Entschlüsseln nannte bis zum 08.10.2026 einen Aufruf, der die Datei im Container suchte:
// Die Sicherungen von update.sh und scripts/backup.sh liegen auf dem Host. Die Datei geht
// deshalb über die Standardeingabe in das Werkzeug, in einem Wegwerf-Container: Beim
// Wiederherstellen ist das Backend angehalten, docker exec liefe dann nicht.
//
// Blindheit: nur Zeilen mit echo, printf oder log_ in update.sh und scripts/*.sh; ein Befehl,
// der über eine Variable oder ein Here-Dokument gedruckt wird, bleibt unsichtbar.
func TestSkripte_GedruckteBefehleOhnePlatzhalter(t *testing.T) {
	// ausgenommen: Datei → Platzhalter, mit Grund.
	ausgenommen := map[string]string{
		// Den Namen des Caddy-Containers kennt das Skript nicht; er hängt am Server.
		"../scripts/deploy.sh": "<caddy_container>",
	}
	dateien, err := filepath.Glob("../scripts/*.sh")
	if err != nil {
		t.Fatal(err)
	}
	dateien = append(dateien, "../update.sh")
	druckt := regexp.MustCompile(`\b(echo|printf|log_[a-z]+)\b`)
	platzhalter := regexp.MustCompile(`<[A-Za-zäöüÄÖÜß_.-]+>`)
	gedruckt, oeffnet := 0, 0
	benutzt := map[string]bool{}
	for _, datei := range dateien {
		for i, zeile := range strings.Split(lies(t, datei), "\n") {
			if strings.HasPrefix(strings.TrimSpace(zeile), "#") || !druckt.MatchString(zeile) {
				continue
			}
			gedruckt++
			for _, fund := range platzhalter.FindAllString(zeile, -1) {
				if ausgenommen[datei] == fund {
					benutzt[datei] = true
					continue
				}
				t.Errorf("%s:%d druckt einen Befehl mit dem Platzhalter %s — den Wert einsetzen oder abfragen (read): %s",
					datei, i+1, fund, strings.TrimSpace(zeile))
			}
			if strings.Contains(zeile, "restore-backup") {
				oeffnet++
				if !strings.Contains(zeile, "docker compose run --rm --no-deps -T --entrypoint ./restore-backup backend /dev/stdin <") {
					t.Errorf("%s:%d nennt restore-backup nicht in der erprobten Form (Wegwerf-Container, Datei über die "+
						"Standardeingabe; docs/resilience_and_recovery.md 2f): %s", datei, i+1, strings.TrimSpace(zeile))
				}
			}
		}
	}
	if gedruckt < 100 {
		t.Fatalf("nur %d druckende Zeilen gefunden — der Detektor sieht die Skripte nicht", gedruckt)
	}
	if oeffnet < 3 {
		t.Fatalf("nur %d gedruckte Hinweise auf restore-backup gefunden, erwartet mindestens 3 (update.sh zweimal, scripts/backup.sh)", oeffnet)
	}
	for datei := range ausgenommen {
		if !benutzt[datei] {
			t.Errorf("die Ausnahme für %s greift nicht mehr — austragen", datei)
		}
	}
}

// umgebungOhneGit liefert die Umgebung ohne die Variablen, mit denen git sein Repository
// findet. Unter einem Hook in einer verknüpften Arbeitskopie zeigt GIT_DIR auf das echte
// Repository — die Wegwerf-Repositories des Tests schrieben sonst dorthin.
func umgebungOhneGit() []string {
	var sauber []string
	for _, eintrag := range os.Environ() {
		if !strings.HasPrefix(eintrag, "GIT_") {
			sauber = append(sauber, eintrag)
		}
	}
	return sauber
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
