package docs

import (
	"os"
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

func lies(t *testing.T, pfad string) string {
	t.Helper()
	b, err := os.ReadFile(pfad)
	if err != nil {
		t.Fatalf("%s lesen: %v", pfad, err)
	}
	return string(b)
}
