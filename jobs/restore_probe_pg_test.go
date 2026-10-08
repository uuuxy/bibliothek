package jobs

// Gate für die wöchentliche Restore-Probe (restore_probe.go): einmal der gute Weg
// (echtes Backup → Probe meldet Erfolg), einmal der schlechte (korrumpierte Datei →
// Probe meldet Fehlschlag). Der zweite Fall ist der wichtigere: Eine Probe, die
// auch kaputte Backups als Erfolg meldet, wäre gefährlicher als gar keine.
//
// Nutzt die Wegwerf-Datenbanken und Voraussetzungs-Prüfung der CI-Drill
// (backup_drill_pg_test.go) — gleiche Sicherheitsbremse, gleiche Werkzeuge.

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestRunRestoreProbe_EchterRoundtripUndKorrupteDatei(t *testing.T) {
	adminDSN := pruefeVoraussetzungen(t)
	ctx := context.Background()

	_, quellDSN := legeProbeDatenbankAn(t, adminDSN, "probe_src")
	befuelleQuelle(t, quellDSN)

	backupDir := t.TempDir()
	t.Setenv("BACKUP_ENCRYPTION_KEY", "probe-passphrase-mit-mehr-als-32-zeichen")
	t.Setenv("DATABASE_URL", quellDSN)
	t.Setenv("BACKUP_DIR", backupDir)
	t.Setenv("S3_ENDPOINT", "")

	(&BackupJob{}).RunDatabaseBackup()
	sicherung := einzigeSicherung(t, backupDir)

	pool, err := pgxpool.New(ctx, quellDSN)
	if err != nil {
		t.Fatalf("Pool auf der Quelle: %v", err)
	}
	t.Cleanup(pool.Close)
	s := &Scheduler{db: pool}

	t.Run("echtes Backup wird wiederhergestellt", func(t *testing.T) {
		s.RunRestoreProbe()
		e := leseProbeErgebnis(t, pool)
		if !e.Erfolg {
			t.Fatalf("Probe meldet Fehlschlag: %s", e.Fehler)
		}
		if e.Tabellen < restoreProbeMinTabellen {
			t.Errorf("nur %d Tabellen — das ist kein vollständiges Schema", e.Tabellen)
		}
		if e.BackupDatei != filepath.Base(sicherung) {
			t.Errorf("Probe prüfte %q statt %q", e.BackupDatei, filepath.Base(sicherung))
		}
	})

	t.Run("Wegwerf-Datenbank wird abgeräumt", func(t *testing.T) {
		var existiert bool
		if err := pool.QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = $1)`,
			restoreProbeDBName).Scan(&existiert); err != nil {
			t.Fatalf("pg_database nicht lesbar: %v", err)
		}
		if existiert {
			t.Errorf("die Wegwerf-Datenbank %s blieb stehen", restoreProbeDBName)
		}
	})

	t.Run("korrumpierte Datei wird als Fehlschlag gemeldet", func(t *testing.T) {
		roh, err := os.ReadFile(sicherung) // #nosec G304 - Pfad aus t.TempDir()
		if err != nil {
			t.Fatalf("Backup-Datei nicht lesbar: %v", err)
		}
		// Ein Byte mitten im Ciphertext kippen — GCM muss die Manipulation erkennen.
		roh[len(roh)/2] ^= 0xFF
		if err := os.WriteFile(sicherung, roh, 0o600); err != nil {
			t.Fatalf("Korrumpieren fehlgeschlagen: %v", err)
		}

		s.RunRestoreProbe()
		e := leseProbeErgebnis(t, pool)
		if e.Erfolg {
			t.Fatal("Probe meldet ERFOLG für eine korrumpierte Backup-Datei — damit wäre sie gefährlicher als keine Probe")
		}
		if !strings.Contains(e.Fehler, "entschlüsselung") {
			t.Errorf("Fehler nennt die Entschlüsselung nicht: %s", e.Fehler)
		}
	})
}

// leseProbeErgebnis liest das Ergebnis des letzten Probelaufs dort, wo die
// Betriebsbereitschaft es liest.
func leseProbeErgebnis(t *testing.T, pool *pgxpool.Pool) RestoreProbeErgebnis {
	t.Helper()
	var wert string
	if err := pool.QueryRow(context.Background(),
		`SELECT wert FROM system_einstellungen WHERE schluessel = $1`,
		RestoreProbeSchluessel).Scan(&wert); err != nil {
		t.Fatalf("Ergebnis nicht gespeichert: %v", err)
	}
	var e RestoreProbeErgebnis
	if err := json.Unmarshal([]byte(wert), &e); err != nil {
		t.Fatalf("Ergebnis kein JSON: %v — %s", err, wert)
	}
	return e
}
