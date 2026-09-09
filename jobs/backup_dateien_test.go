package jobs

import (
	"os"
	"path/filepath"
	"testing"
)

// Die ersten Tests für Auflistung und Rotation — beide Funktionen hatten bis zum
// 09.09.2026 keinen. Rot bewiesen: Löscht die Rotation eine Datei zu viel, fällt
// TestRotateBackups_BehaeltDieJuengsten (siehe Commit-Text).

func legeDateiAn(t *testing.T, dir, name string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0600); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}

func namen(dateien []BackupDatei) []string {
	out := make([]string, 0, len(dateien))
	for _, d := range dateien {
		out = append(out, d.Name)
	}
	return out
}

func TestBackupDateien_NurSicherungenUndNachNamenSortiert(t *testing.T) {
	dir := t.TempDir()
	legeDateiAn(t, dir, "backup_2026-09-02T02-30.sql.gz.enc")
	legeDateiAn(t, dir, "backup_2026-09-01T02-30.sql.gz.enc")
	legeDateiAn(t, dir, "notiz.txt")                                                       // Fremddatei
	legeDateiAn(t, dir, "backup_2026-09-03T02-30.sql.gz")                                  // Klartext-Dump: keine Sicherung
	if err := os.Mkdir(filepath.Join(dir, "backup_ordner.sql.gz.enc"), 0750); err != nil { // passender Name, aber Verzeichnis
		t.Fatal(err)
	}

	dateien, err := BackupDateien(dir)
	if err != nil {
		t.Fatalf("BackupDateien: %v", err)
	}
	got := namen(dateien)
	want := []string{"backup_2026-09-01T02-30.sql.gz.enc", "backup_2026-09-02T02-30.sql.gz.enc"}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("Liste = %v, erwartet %v (nur Sicherungen, älteste zuerst)", got, want)
	}
}

func TestBackupDateien_FehlendesVerzeichnisIstKeinFehler(t *testing.T) {
	dateien, err := BackupDateien(filepath.Join(t.TempDir(), "gibt-es-nicht"))
	if err != nil || len(dateien) != 0 {
		t.Errorf("fehlendes Verzeichnis = leere Liste ohne Fehler, bekam %v / %v", dateien, err)
	}
}

func TestRotateBackups_BehaeltDieJuengsten(t *testing.T) {
	dir := t.TempDir()
	for _, n := range []string{
		"backup_2026-09-01T02-30.sql.gz.enc",
		"backup_2026-09-02T02-30.sql.gz.enc",
		"backup_2026-09-03T02-30.sql.gz.enc",
		"backup_2026-09-04T02-30.sql.gz.enc",
	} {
		legeDateiAn(t, dir, n)
	}
	legeDateiAn(t, dir, "notiz.txt")

	rotateBackups(dir, 10) // weniger als maxKeep: nichts passiert
	vorher, err := BackupDateien(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(vorher) != 4 {
		t.Fatalf("unter maxKeep darf nichts gelöscht werden, übrig %v", namen(vorher))
	}

	rotateBackups(dir, 2)
	d, err := BackupDateien(dir)
	if err != nil {
		t.Fatal(err)
	}
	got := namen(d)
	if len(got) != 2 || got[0] != "backup_2026-09-03T02-30.sql.gz.enc" || got[1] != "backup_2026-09-04T02-30.sql.gz.enc" {
		t.Errorf("nach Rotation auf 2 müssen die zwei jüngsten bleiben, übrig %v", got)
	}
	if _, err := os.Stat(filepath.Join(dir, "notiz.txt")); err != nil {
		t.Errorf("Fremddateien rührt die Rotation nicht an: %v", err)
	}
}

// Ein Symlink im Backup-Verzeichnis, der nach draußen zeigt, verschwindet bei der
// Rotation als LINK — sein Ziel bleibt unberührt. Das ist der Grund für os.OpenRoot.
func TestRotateBackups_LoeschtSymlinkNichtSeinZiel(t *testing.T) {
	dir := t.TempDir()
	draussen := filepath.Join(t.TempDir(), "wichtig.txt")
	if err := os.WriteFile(draussen, []byte("bleibt"), 0600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "backup_2026-08-31T02-30.sql.gz.enc") // sortiert vor allen anderen
	if err := os.Symlink(draussen, link); err != nil {
		t.Skipf("Symlinks hier nicht möglich: %v", err)
	}
	legeDateiAn(t, dir, "backup_2026-09-01T02-30.sql.gz.enc")
	legeDateiAn(t, dir, "backup_2026-09-02T02-30.sql.gz.enc")

	rotateBackups(dir, 2)

	if _, err := os.Lstat(link); !os.IsNotExist(err) {
		t.Errorf("der Link ist der älteste Eintrag und muss weg sein: %v", err)
	}
	if inhalt, err := os.ReadFile(draussen); err != nil || string(inhalt) != "bleibt" {
		t.Errorf("das Ziel des Links darf die Rotation nicht anfassen: %v %q", err, inhalt)
	}
}
