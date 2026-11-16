package jobs

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// Die Sicherung entsteht unter einem Namen, den die Liste der Sicherungen nicht sieht, und steht
// erst nach dem Umbenennen unter ihrem eigenen, vollständig und nur für den Eigentümer lesbar.
func TestSchreibeSicherungsdatei_UeberVorlaeufigenNamen(t *testing.T) {
	verzeichnis := t.TempDir()
	ziel := filepath.Join(verzeichnis, backupPraefix+time.Now().UTC().Format(sicherungsStempel)+backupEndung)
	vorlaeufig := ziel + endungUnfertig

	// So sieht es nach einem Absturz beim Schreiben aus.
	if err := os.WriteFile(vorlaeufig, []byte("halb"), 0600); err != nil {
		t.Fatal(err)
	}
	if dateien, err := LeseBackupDateien(verzeichnis); err != nil || len(dateien) != 0 {
		t.Fatalf("LeseBackupDateien mit unfertiger Datei = %v, %v, erwartet keine Sicherung", dateien, err)
	}

	daten := []byte("vollständige Sicherung")
	if err := schreibeSicherungsdatei(ziel, daten); err != nil {
		t.Fatalf("schreibeSicherungsdatei: %v", err)
	}
	if got, err := os.ReadFile(ziel); err != nil || string(got) != string(daten) {
		t.Errorf("Inhalt = %q, %v, erwartet %q", got, err, daten)
	}
	if info, err := os.Stat(ziel); err != nil || info.Mode().Perm() != 0600 {
		t.Errorf("Rechte = %v, %v, erwartet 0600", info.Mode().Perm(), err)
	}
	if _, err := os.Stat(vorlaeufig); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("vorläufige Datei nach dem Schreiben: %v, erwartet umbenannt", err)
	}
}

// Nach einem gelungenen Lauf räumt die Aufbewahrung weg, was ein abgebrochener unter
// vorläufigem Namen hinterließ; die Sicherungen bleiben.
func TestRotiereBackups_LoeschtUnfertige(t *testing.T) {
	verzeichnis := t.TempDir()
	sicherung := filepath.Join(verzeichnis, backupPraefix+time.Now().UTC().Format(sicherungsStempel)+backupEndung)
	unfertig := filepath.Join(verzeichnis, backupPraefix+"2026-09-01T023000Z"+backupEndung+endungUnfertig)
	fremd := filepath.Join(verzeichnis, "notiz"+endungUnfertig)
	for _, pfad := range []string{sicherung, unfertig, fremd} {
		if err := os.WriteFile(pfad, []byte("x"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	rotiereBackups(verzeichnis, AufbewahrteNaechte, AufbewahrteWochen)
	if _, err := os.Stat(unfertig); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("unfertige Sicherung nach der Aufbewahrung: %v, erwartet gelöscht", err)
	}
	for _, pfad := range []string{sicherung, fremd} {
		if _, err := os.Stat(pfad); err != nil {
			t.Errorf("%s nach der Aufbewahrung: %v, erwartet vorhanden", filepath.Base(pfad), err)
		}
	}
}
