package jobs

// Der Wechsel des BACKUP_ENCRYPTION_KEY am echten Weg (docs/SECURITY.md,
// „BACKUP_ENCRYPTION_KEY wechseln"): Der Wechsel schlüsselt nichts um. Eine Nachtsicherung von
// vor dem Wechsel öffnet nur der alte Schlüssel, und mit ihm lässt sie sich einspielen. Die
// Probe am Sonntag nimmt die jüngste Sicherung und läuft erst wieder durch, wenn eine mit dem
// neuen Schlüssel entstanden ist.
//
// Nutzt die Wegwerf-Datenbanken und Helfer der CI-Drill (backup_drill_pg_test.go).

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestSchluesselwechsel_AeltereSicherungBrauchtDenAltenSchluessel(t *testing.T) {
	adminDSN := pruefeVoraussetzungen(t)
	const (
		alt = "alter-schluessel-mit-mehr-als-32-zeichen-0001"
		neu = "neuer-schluessel-mit-mehr-als-32-zeichen-0002"
	)

	_, quellDSN := legeProbeDatenbankAn(t, adminDSN, "wechsel_src")
	_, zielDSN := legeProbeDatenbankAn(t, adminDSN, "wechsel_dst")
	befuelleQuelle(t, quellDSN)
	erwartet := zaehleAlleTabellen(t, quellDSN)

	backupDir := t.TempDir()
	t.Setenv("DATABASE_URL", quellDSN)
	t.Setenv("BACKUP_DIR", backupDir)
	t.Setenv("S3_ENDPOINT", "")

	// Die Nachtsicherung von gestern, mit dem Schlüssel von gestern. Name und Änderungszeit
	// einen Tag zurück: Der Name trägt die Sekunde, und die Probe wählt nach der Änderungszeit.
	t.Setenv("BACKUP_ENCRYPTION_KEY", alt)
	(&BackupJob{}).RunDatabaseBackup()
	gestern := time.Now().UTC().Add(-24 * time.Hour)
	vorher := filepath.Join(backupDir, backupPraefix+gestern.Format(sicherungsStempel)+backupEndung)
	if err := os.Rename(einzigeSicherung(t, backupDir), vorher); err != nil {
		t.Fatalf("Sicherung umbenennen: %v", err)
	}
	if err := os.Chtimes(vorher, gestern, gestern); err != nil {
		t.Fatalf("Änderungszeit setzen: %v", err)
	}
	rohVorher, err := os.ReadFile(vorher) // #nosec G304 - Pfad aus t.TempDir()
	if err != nil {
		t.Fatalf("Sicherung lesen: %v", err)
	}

	// Der Wechsel: Ab hier gilt im Programm der neue Schlüssel.
	t.Setenv("BACKUP_ENCRYPTION_KEY", neu)

	pool, err := pgxpool.New(context.Background(), quellDSN)
	if err != nil {
		t.Fatalf("Pool auf der Quelle: %v", err)
	}
	t.Cleanup(pool.Close)
	s := &Scheduler{db: pool}

	t.Run("vor der ersten neuen Sicherung scheitert die Probe am Schlüssel", func(t *testing.T) {
		s.RunRestoreProbe()
		e := leseProbeErgebnis(t, pool)
		if e.Erfolg {
			t.Fatal("die Probe meldet Erfolg für eine Sicherung, die der laufende Schlüssel nicht öffnet")
		}
		if !strings.Contains(e.Fehler, "falscher Schlüssel") {
			t.Errorf("der Befund nennt den Schlüssel nicht: %s", e.Fehler)
		}
	})

	t.Run("mit der ersten neuen Sicherung besteht die Probe wieder", func(t *testing.T) {
		(&BackupJob{}).RunDatabaseBackup()
		s.RunRestoreProbe()
		e := leseProbeErgebnis(t, pool)
		if !e.Erfolg {
			t.Fatalf("die Probe meldet Fehlschlag: %s", e.Fehler)
		}
		if e.BackupDatei == filepath.Base(vorher) {
			t.Errorf("die Probe prüfte die Sicherung von vor dem Wechsel (%s)", e.BackupDatei)
		}
	})

	t.Run("der laufende Schlüssel öffnet die ältere Sicherung nicht", func(t *testing.T) {
		if _, err := entschluesseleBackup(neu, rohVorher); err == nil {
			t.Fatal("der neue Schlüssel öffnet eine Sicherung von vor dem Wechsel")
		}
	})

	t.Run("mit dem alten Schlüssel lässt sie sich einspielen", func(t *testing.T) {
		spieleInZielEin(t, zielDSN, entschluesseleUndPacke(t, alt, rohVorher))
		vergleicheBestand(t, erwartet, zaehleAlleTabellen(t, zielDSN))
	})
}

// einzigeSicherung liefert die eine Nachtsicherung im Verzeichnis.
func einzigeSicherung(t *testing.T, dir string) string {
	t.Helper()
	treffer, err := filepath.Glob(filepath.Join(dir, backupPraefix+"*"+backupEndung))
	if err != nil || len(treffer) != 1 {
		t.Fatalf("genau eine Sicherung erwartet, gefunden: %v (err=%v)", treffer, err)
	}
	return treffer[0]
}
