package docs

import (
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Ein Ordner, zwei Skripte. In ./backups auf dem Host legen update.sh (vordeploy_…, 30 Tage)
// und scripts/backup.sh (bibliothek_backup_…, 7 Tage) ab; beide räumen denselben Ordner auf.
//
// Nachgesehen am 28.09.2026, zwei Funde:
//
//  1. scripts/backup.sh löschte mit -name "*.sql.gz.enc" -mtime +7 auch die Vorab-Sicherungen
//     von update.sh. Mit der Beispiel-Crontab aus resilience_and_recovery.md hielten sie
//     damit 7 statt der dokumentierten 30 Tage.
//  2. Die Klartext-Warnung von update.sh zählte nur backup_*.sql.gz. Seit der Umbenennung
//     der Vorab-Sicherung in vordeploy_ (10.09.2026) meldete sie einen unverschlüsselten
//     Rest nach einem misslungenen Update nicht mehr.
//
// Die Regel: Eine verschlüsselte Sicherung löscht nur das Skript, das sie angelegt hat —
// die Fristen sind verschieden. Klartext hat im ganzen Ordner eine Frist, gleich von wem,
// und jedes Skript meldet jeden Klartext.
//
// Blindheit: Gelesen werden nur die -name-Muster der find-Zeilen über $BACKUP_DIR in den
// zwei Skripten. Ein drittes Skript, das denselben Ordner aufräumt, ein rm mit Glob und die
// -mtime-Werte sieht der Test nicht.

var backupSkripte = map[string]struct {
	erzeugt string   // die Zeile, die den Dateinamen bildet — hält die Beispielnamen ehrlich
	eigene  []string // Beispielnamen dieses Skripts, ohne Endung .enc
}{
	"../update.sh": {
		erzeugt: `vordeploy_${TIMESTAMP}.sql.gz`,
		// backup_…: Vorab-Sicherungen vor dem 10.09.2026, update.sh räumt sie weiter auf.
		eigene: []string{"vordeploy_20260910_000000.sql.gz", "backup_20260901_000000.sql.gz"},
	},
	"../scripts/backup.sh": {
		erzeugt: `bibliothek_backup_$TIMESTAMP.sql.gz`,
		eigene:  []string{"bibliothek_backup_2026-09-10.sql.gz"},
	},
}

var (
	findZeile  = regexp.MustCompile(`find\s+"\$\{?BACKUP_DIR\}?"[^\n]*`)
	nameMuster = regexp.MustCompile(`-name\s+"([^"]+)"`)
)

// findMuster liefert die -name-Muster der find-Aufrufe eines Skripts, getrennt nach
// Aufräumen (-delete, rm) und Zählen, und jeweils nach verschlüsselt und Klartext.
func findMuster(t *testing.T, skript string) (loeschtEnc, loeschtKlar, zaehltKlar []string) {
	t.Helper()
	for _, zeile := range findZeile.FindAllString(skript, -1) {
		loescht := strings.Contains(zeile, "-delete") || strings.Contains(zeile, "rm -f")
		for _, m := range nameMuster.FindAllStringSubmatch(zeile, -1) {
			enc := strings.HasSuffix(m[1], ".enc")
			switch {
			case loescht && enc:
				loeschtEnc = append(loeschtEnc, m[1])
			case loescht:
				loeschtKlar = append(loeschtKlar, m[1])
			case !enc:
				zaehltKlar = append(zaehltKlar, m[1])
			}
		}
	}
	return
}

func trifftEines(t *testing.T, muster []string, name string) bool {
	t.Helper()
	for _, m := range muster {
		ok, err := filepath.Match(m, name)
		if err != nil {
			t.Fatalf("Muster %q: %v", m, err)
		}
		if ok {
			return true
		}
	}
	return false
}

func TestBackupAblage_JedesSkriptLoeschtNurSeineVerschluesselten(t *testing.T) {
	for pfad, eigen := range backupSkripte {
		skript := lies(t, pfad)
		if !strings.Contains(skript, eigen.erzeugt) {
			t.Fatalf("%s bildet seinen Dateinamen nicht mehr über %q — Beispielnamen im Test nachziehen", pfad, eigen.erzeugt)
		}
		loeschtEnc, _, _ := findMuster(t, skript)
		if len(loeschtEnc) == 0 {
			t.Fatalf("%s: kein find-Aufruf, der verschlüsselte Sicherungen löscht — der Detektor sieht nichts", pfad)
		}
		for _, name := range eigen.eigene {
			if !trifftEines(t, loeschtEnc, name+".enc") {
				t.Errorf("%s räumt seine eigene Sicherung %s.enc nicht auf (Muster %v)", pfad, name, loeschtEnc)
			}
		}
		for fremdPfad, fremd := range backupSkripte {
			if fremdPfad == pfad {
				continue
			}
			for _, name := range fremd.eigene {
				if trifftEines(t, loeschtEnc, name+".enc") {
					t.Errorf("%s löscht die Sicherung %s.enc von %s mit eigener Frist (Muster %v)",
						pfad, name, fremdPfad, loeschtEnc)
				}
			}
		}
	}
}

func TestBackupAblage_KlartextHatEineFristUndWirdGemeldet(t *testing.T) {
	var alleKlar []string
	for _, eigen := range backupSkripte {
		alleKlar = append(alleKlar, eigen.eigene...)
	}
	for pfad := range backupSkripte {
		_, loeschtKlar, zaehltKlar := findMuster(t, lies(t, pfad))
		if len(loeschtKlar) == 0 || len(zaehltKlar) == 0 {
			t.Fatalf("%s: Klartext wird nicht gelöscht (%v) oder nicht gezählt (%v) — der Detektor sieht nichts",
				pfad, loeschtKlar, zaehltKlar)
		}
		for _, name := range alleKlar {
			if !trifftEines(t, loeschtKlar, name) {
				t.Errorf("%s löscht den Klartext-Rest %s nicht (Muster %v)", pfad, name, loeschtKlar)
			}
			if !trifftEines(t, zaehltKlar, name) {
				t.Errorf("%s meldet den Klartext-Rest %s nicht (Muster %v)", pfad, name, zaehltKlar)
			}
		}
	}
}
