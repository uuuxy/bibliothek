package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// backupSkriptAusfuehren kopiert backup.sh und backup_krypto.sh in ein leeres Verzeichnis und
// startet es mit einer Attrappe von docker. Sie meldet keinen laufenden Container des Backends,
// also läuft der Zweig im Klartext; pgDump ist, was `docker exec` ausgibt, oder bei leerem Text
// ein Abbruch wie bei einer Datenbank, die nicht antwortet.
func backupSkriptAusfuehren(t *testing.T, verzeichnis, pgDump string) ([]byte, error) {
	t.Helper()
	skripte := filepath.Join(verzeichnis, "scripts")
	binVerzeichnis := filepath.Join(verzeichnis, "bin")
	for _, ordner := range []string{skripte, binVerzeichnis} {
		if err := os.MkdirAll(ordner, 0o755); err != nil {
			t.Fatalf("os.MkdirAll: %v", err)
		}
	}
	for _, name := range []string{"backup.sh", "backup_krypto.sh"} {
		inhalt, err := os.ReadFile(filepath.Join("..", "..", "scripts", name))
		if err != nil {
			t.Fatalf("os.ReadFile: %v", err)
		}
		if err := os.WriteFile(filepath.Join(skripte, name), inhalt, 0o755); err != nil {
			t.Fatalf("os.WriteFile: %v", err)
		}
	}
	ausgabe := `echo "` + pgDump + `"`
	if pgDump == "" {
		ausgabe = "exit 1"
	}
	attrappe := "#!/bin/bash\ncase \"$1\" in\n  ps) exit 0 ;;\n  exec) " + ausgabe + " ;;\nesac\nexit 0\n"
	if err := os.WriteFile(filepath.Join(binVerzeichnis, "docker"), []byte(attrappe), 0o755); err != nil {
		t.Fatalf("os.WriteFile: %v", err)
	}
	befehl := exec.Command("bash", filepath.Join(skripte, "backup.sh"))
	befehl.Dir = verzeichnis
	befehl.Env = []string{"PATH=" + binVerzeichnis + string(os.PathListSeparator) + os.Getenv("PATH"), "HOME=" + verzeichnis}
	return befehl.CombinedOutput()
}

// Ein zweiter Lauf am selben Tag schnitt die Sicherung des Morgens ab, und scheiterte er, löschte
// er sie ganz. Jetzt schreibt er unter vorläufigem Namen: Scheitert er, steht die Sicherung des
// Morgens unverändert da; gelingt er, ersetzt er sie, und keine Zwischendatei bleibt liegen.
func TestBackupSkript_ZweiterLaufAmSelbenTag(t *testing.T) {
	verzeichnis := t.TempDir()
	sicherungen := filepath.Join(verzeichnis, "backups")
	if err := os.MkdirAll(sicherungen, 0o700); err != nil {
		t.Fatalf("os.MkdirAll: %v", err)
	}
	vonHeute := filepath.Join(sicherungen, "bibliothek_backup_"+time.Now().Format("2006-01-02")+".sql.gz")
	if err := os.WriteFile(vonHeute, []byte("Sicherung des Morgens"), 0o600); err != nil {
		t.Fatalf("os.WriteFile: %v", err)
	}

	if aus, err := backupSkriptAusfuehren(t, verzeichnis, ""); err == nil {
		t.Fatalf("backup.sh ohne Datenbank endete ohne Fehler: %s", aus)
	}
	if inhalt, err := os.ReadFile(vonHeute); err != nil || string(inhalt) != "Sicherung des Morgens" {
		t.Errorf("nach dem gescheiterten Lauf: %q, %v; erwartet die Sicherung des Morgens", inhalt, err)
	}

	if aus, err := backupSkriptAusfuehren(t, verzeichnis, "-- pg_dump am Nachmittag"); err != nil {
		t.Fatalf("backup.sh: %s: %v", aus, err)
	}
	if inhalt, err := os.ReadFile(vonHeute); err != nil || string(inhalt) == "Sicherung des Morgens" {
		t.Errorf("nach dem gelungenen Lauf: %q, %v; erwartet die neue Sicherung", inhalt, err)
	}
	eintraege, err := os.ReadDir(sicherungen)
	if err != nil {
		t.Fatalf("os.ReadDir: %v", err)
	}
	if len(eintraege) != 1 {
		t.Errorf("in backups liegt %v, erwartet nur die Sicherung von heute", eintraege)
	}
}

// Ein liegengebliebener Abzug im Klartext geht beim ersten Lauf nach zwei Tagen, wie es
// docs/resilience_and_recovery.md zusagt. `-mtime +2` zählt in ganzen Tagen und nahm ihn erst
// nach drei.
func TestBackupSkript_KlartextRestGehtNachZweiTagen(t *testing.T) {
	verzeichnis := t.TempDir()
	sicherungen := filepath.Join(verzeichnis, "backups")
	if err := os.MkdirAll(sicherungen, 0o700); err != nil {
		t.Fatalf("os.MkdirAll: %v", err)
	}
	lege := func(name string, alter time.Duration) string {
		t.Helper()
		pfad := filepath.Join(sicherungen, name)
		if err := os.WriteFile(pfad, []byte("Abzug"), 0o600); err != nil {
			t.Fatalf("os.WriteFile: %v", err)
		}
		zeit := time.Now().Add(-alter)
		if err := os.Chtimes(pfad, zeit, zeit); err != nil {
			t.Fatalf("os.Chtimes: %v", err)
		}
		return pfad
	}
	alt := lege("vordeploy_alt.sql.gz", 50*time.Hour)
	jung := lege("vordeploy_jung.sql.gz", 40*time.Hour)

	if aus, err := backupSkriptAusfuehren(t, verzeichnis, "-- pg_dump"); err != nil {
		t.Fatalf("backup.sh: %s: %v", aus, err)
	}
	if _, err := os.Stat(alt); !os.IsNotExist(err) {
		t.Errorf("der Abzug von vor 50 Stunden liegt noch da (%v), erwartet gelöscht", err)
	}
	if _, err := os.Stat(jung); err != nil {
		t.Errorf("der Abzug von vor 40 Stunden: %v, erwartet noch da", err)
	}
}
