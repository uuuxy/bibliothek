package docs

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
)

// Ein Ordner, zwei Skripte. In ./backups auf dem Host legen update.sh (vordeploy_…, 30 Tage)
// und scripts/backup.sh (bibliothek_backup_…, 7 Tage) ab; beide räumen denselben Ordner auf.
//
// Löschte scripts/backup.sh jede "*.sql.gz.enc", hielten die Vorab-Sicherungen von update.sh
// mit der Beispiel-Crontab aus resilience_and_recovery.md 7 statt der dokumentierten 30 Tage.
// Zählt eine Klartext-Warnung nur einen Namen, bleibt der unverschlüsselte Rest unter dem
// anderen ungemeldet.
//
// Die Regel: Eine verschlüsselte Sicherung löscht nur das Skript, das sie angelegt hat —
// die Fristen sind verschieden. Klartext hat im ganzen Ordner eine Frist, gleich von wem,
// und jedes Skript meldet jeden Klartext.
//
// Blindheit: Gelesen werden nur die -name-Muster der find-Zeilen über $BACKUP_DIR in den
// zwei Skripten. Ein drittes Skript, das denselben Ordner aufräumt, ein rm mit Glob und die
// -mtime-Werte sieht der Test nicht. Das Versprechen eines Löschens nach Zeit erkennt er nur in
// drei Wortformen („automatisch gelöscht", „verschwindet nach", „wird in/nach ${…} Tagen").

var backupSkripte = map[string]struct {
	erzeugt string   // die Zeile, die den Dateinamen bildet — hält die Beispielnamen ehrlich
	eigene  []string // Beispielnamen dieses Skripts, ohne Endung .enc
}{
	"../update.sh": {
		erzeugt: `vordeploy_${TIMESTAMP}.sql.gz`,
		// backup_…: Vorab-Sicherungen unter dem alten Namen; update.sh räumt auch sie auf.
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

// findeMuster liefert die -name-Muster der find-Aufrufe eines Skripts, getrennt nach
// Aufräumen (-delete, rm) und Zählen, und jeweils nach verschlüsselt und Klartext.
func findeMuster(t *testing.T, skript string) (loeschtEnc, loeschtKlar, zaehltKlar []string) {
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
		loeschtEnc, _, _ := findeMuster(t, skript)
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
		_, loeschtKlar, zaehltKlar := findeMuster(t, lies(t, pfad))
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

// Gelöscht wird nur, wenn eines der Skripte läuft. Wer einer Zusage wie „wird in 2 Tagen
// automatisch gelöscht" glaubt, lässt die ganze Datenbank im Klartext liegen, bis zum nächsten
// Lauf, der Wochen später kommen kann.
func TestBackupAblage_KeineLoeschUhrVersprochen(t *testing.T) {
	versprechen := regexp.MustCompile(`(?i)automatisch gelöscht|verschwindet nach|wird (in|nach) \$\{[A-Z_]+\} Tagen`)
	for pfad := range backupSkripte {
		for i, zeile := range strings.Split(lies(t, pfad), "\n") {
			if versprechen.MatchString(zeile) {
				t.Errorf("%s:%d verspricht ein Löschen nach Zeit, gelöscht wird aber nur bei einem Lauf: %s",
					pfad, i+1, strings.TrimSpace(zeile))
			}
		}
	}
}

// Die Uhr am Server ruft scripts/backup.sh --nur-aufraeumen (DEPLOYMENT.md §6). Sie räumt nach
// denselben Fristen auf wie ein Lauf und legt keine Sicherung an: docker ruft sie nie.
func TestBackupAblage_UhrRaeumtNurAuf(t *testing.T) {
	wurzel := t.TempDir()
	skripte := filepath.Join(wurzel, "scripts")
	ablage := filepath.Join(wurzel, "backups")
	bin := filepath.Join(wurzel, "bin")
	for _, ordner := range []string{skripte, ablage, bin} {
		if err := os.Mkdir(ordner, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range []string{"backup.sh", "backup_krypto.sh"} {
		inhalt := lies(t, "../scripts/"+name)
		if err := os.WriteFile(filepath.Join(skripte, name), []byte(inhalt), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	protokoll := filepath.Join(wurzel, "docker.log")
	stellvertreter := "#!/bin/sh\necho \"$*\" >> \"" + protokoll + "\"\nexit 1\n"
	if err := os.WriteFile(filepath.Join(bin, "docker"), []byte(stellvertreter), 0o755); err != nil {
		t.Fatal(err)
	}

	alt := time.Now().Add(-10 * 24 * time.Hour)
	dateien := []struct {
		name        string
		alt, bleibt bool
	}{
		{"bibliothek_backup_2026-09-01.sql.gz.enc", true, false},
		{"vordeploy_20260901_000000.sql.gz.enc", true, true}, // update.sh hält sie 30 Tage
		{"vordeploy_20260901_000000.sql.gz", true, false},    // Klartext-Rest eines Updates
		{"bibliothek_backup_2026-10-21.sql.gz", false, true},
	}
	for _, d := range dateien {
		pfad := filepath.Join(ablage, d.name)
		if err := os.WriteFile(pfad, nil, 0o600); err != nil {
			t.Fatal(err)
		}
		if d.alt {
			if err := os.Chtimes(pfad, alt, alt); err != nil {
				t.Fatal(err)
			}
		}
	}

	// Logs ersetzter Container legt update.sh unter logs/ ab; die Uhr löscht sie nach 14 Tagen mit.
	logs := filepath.Join(wurzel, "logs")
	logAlt, logJung := "backend_20260901_000000.log", "backend_20261010_000000.log"
	schreibeLog(t, logs, logAlt, 20*24*time.Hour)
	schreibeLog(t, logs, logJung, 10*24*time.Hour)

	lauf := func(angabe string) (string, error) {
		befehl := exec.Command("bash", filepath.Join(skripte, "backup.sh"), angabe)
		befehl.Env = append(os.Environ(), "PATH="+bin+string(os.PathListSeparator)+os.Getenv("PATH"))
		ausgabe, err := befehl.CombinedOutput()
		return string(ausgabe), err
	}
	ausgabe, err := lauf("--nur-aufraeumen")
	if err != nil {
		t.Fatalf("--nur-aufraeumen: %v\n%s", err, ausgabe)
	}
	for _, d := range dateien {
		_, fehler := os.Stat(filepath.Join(ablage, d.name))
		if da := fehler == nil; da != d.bleibt {
			t.Errorf("%s: nach dem Aufräumen da = %v, erwartet %v", d.name, da, d.bleibt)
		}
	}
	if !strings.Contains(ausgabe, "WARNUNG: 1 unverschlüsselte") {
		t.Errorf("der frische Klartext wird nicht gemeldet:\n%s", ausgabe)
	}
	for name, bleibt := range map[string]bool{logAlt: false, logJung: true} {
		_, fehler := os.Stat(filepath.Join(logs, name))
		if da := fehler == nil; da != bleibt {
			t.Errorf("%s: nach dem Aufräumen da = %v, erwartet %v", name, da, bleibt)
		}
	}

	ausgabe, err = lauf("--nur-aufraumen")
	var ende *exec.ExitError
	if !errors.As(err, &ende) || ende.ExitCode() != 2 {
		t.Errorf("eine unbekannte Angabe endet nicht mit 2: %v\n%s", err, ausgabe)
	}
	// Ohne Aufruf legt der Stellvertreter kein Protokoll an.
	aufrufe, err := os.ReadFile(protokoll)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		t.Fatal(err)
	}
	if len(aufrufe) > 0 {
		t.Errorf("die Uhr oder die unbekannte Angabe rief docker:\n%s", aufrufe)
	}
}
