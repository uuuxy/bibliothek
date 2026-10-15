package jobs

import (
	"os"
	"path/filepath"
	"testing"
)

// Die Tests für das Lesen der Sicherungen und die Aufbewahrung. Löscht die Aufbewahrung eine
// Datei zu viel, wird TestRotiereBackups_BehaeltDieJuengsten rot.

func legeDateiAn(t *testing.T, verzeichnis, name string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(verzeichnis, name), []byte("x"), 0600); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}

func namen(dateien []BackupDatei) []string {
	ausgabe := make([]string, 0, len(dateien))
	for _, d := range dateien {
		ausgabe = append(ausgabe, d.Name)
	}
	return ausgabe
}

func TestLeseBackupDateien_NurSicherungenUndNachNamenSortiert(t *testing.T) {
	verzeichnis := t.TempDir()
	legeDateiAn(t, verzeichnis, "backup_2026-09-02T02-30.sql.gz.enc")
	legeDateiAn(t, verzeichnis, "backup_2026-09-01T02-30.sql.gz.enc")
	legeDateiAn(t, verzeichnis, "notiz.txt")                                                       // Fremddatei
	legeDateiAn(t, verzeichnis, "backup_2026-09-03T02-30.sql.gz")                                  // Klartext-Dump: keine Sicherung
	if err := os.Mkdir(filepath.Join(verzeichnis, "backup_ordner.sql.gz.enc"), 0750); err != nil { // passender Name, aber Verzeichnis
		t.Fatal(err)
	}

	dateien, err := LeseBackupDateien(verzeichnis)
	if err != nil {
		t.Fatalf("LeseBackupDateien: %v", err)
	}
	got := namen(dateien)
	want := []string{"backup_2026-09-01T02-30.sql.gz.enc", "backup_2026-09-02T02-30.sql.gz.enc"}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("Liste = %v, erwartet %v (nur Sicherungen, älteste zuerst)", got, want)
	}
}

func TestLeseBackupDateien_FehlendesVerzeichnisIstKeinFehler(t *testing.T) {
	dateien, err := LeseBackupDateien(filepath.Join(t.TempDir(), "gibt-es-nicht"))
	if err != nil || len(dateien) != 0 {
		t.Errorf("fehlendes Verzeichnis = leere Liste ohne Fehler, bekam %v / %v", dateien, err)
	}
}

func TestRotiereBackups_BehaeltDieJuengsten(t *testing.T) {
	verzeichnis := t.TempDir()
	for _, n := range []string{
		"backup_2026-09-01T02-30.sql.gz.enc",
		"backup_2026-09-02T02-30.sql.gz.enc",
		"backup_2026-09-03T02-30.sql.gz.enc",
		"backup_2026-09-04T02-30.sql.gz.enc",
	} {
		legeDateiAn(t, verzeichnis, n)
	}
	legeDateiAn(t, verzeichnis, "notiz.txt")

	rotiereBackups(verzeichnis, 10, 0) // weniger als die Nächte: nichts passiert
	vorher, err := LeseBackupDateien(verzeichnis)
	if err != nil {
		t.Fatal(err)
	}
	if len(vorher) != 4 {
		t.Fatalf("unter der Zahl der Nächte darf nichts gelöscht werden, übrig %v", namen(vorher))
	}

	rotiereBackups(verzeichnis, 2, 0)
	d, err := LeseBackupDateien(verzeichnis)
	if err != nil {
		t.Fatal(err)
	}
	got := namen(d)
	if len(got) != 2 || got[0] != "backup_2026-09-03T02-30.sql.gz.enc" || got[1] != "backup_2026-09-04T02-30.sql.gz.enc" {
		t.Errorf("nach Rotation auf 2 müssen die zwei jüngsten bleiben, übrig %v", got)
	}
	if _, err := os.Stat(filepath.Join(verzeichnis, "notiz.txt")); err != nil {
		t.Errorf("Fremddateien rührt die Rotation nicht an: %v", err)
	}
}

// Ein Symlink im Backup-Verzeichnis, der nach draußen zeigt, verschwindet bei der Aufbewahrung
// als Link; sein Ziel bleibt unberührt. Das ist der Grund für os.OpenRoot.
func TestRotiereBackups_LoeschtSymlinkNichtSeinZiel(t *testing.T) {
	verzeichnis := t.TempDir()
	draussen := filepath.Join(t.TempDir(), "wichtig.txt")
	if err := os.WriteFile(draussen, []byte("bleibt"), 0600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(verzeichnis, "backup_2026-08-31T02-30.sql.gz.enc") // sortiert vor allen anderen
	if err := os.Symlink(draussen, link); err != nil {
		t.Skipf("Symlinks hier nicht möglich: %v", err)
	}
	legeDateiAn(t, verzeichnis, "backup_2026-09-01T02-30.sql.gz.enc")
	legeDateiAn(t, verzeichnis, "backup_2026-09-02T02-30.sql.gz.enc")

	rotiereBackups(verzeichnis, 2, 0)

	if _, err := os.Lstat(link); !os.IsNotExist(err) {
		t.Errorf("der Link ist der älteste Eintrag und muss weg sein: %v", err)
	}
	if inhalt, err := os.ReadFile(draussen); err != nil || string(inhalt) != "bleibt" {
		t.Errorf("das Ziel des Links darf die Rotation nicht anfassen: %v %q", err, inhalt)
	}
}
