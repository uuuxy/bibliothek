package jobs

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Der Start probt nur nach einem gespeicherten Fehlschlag erneut (cron.go). Geprüft am Grund
// der Regel: Nach der Behebung stellt ein Neustart den Befund richtig, und ein gesunder Stack
// spielt beim Start keine Sicherung ein.
func TestProbeBeimStart_NurNachFehlschlag(t *testing.T) {
	adminDSN := pruefeVoraussetzungen(t)
	ctx := context.Background()

	_, quellDSN := legeProbeDatenbankAn(t, adminDSN, "probe_start")
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

	// Ein Name, den kein echter Lauf vergibt: Steht er nach dem Start noch da, lief keine Probe.
	const marke = "frueherer-lauf.sql.gz.enc"
	frueher := time.Date(2026, time.January, 4, 3, 30, 0, 0, time.UTC)

	t.Run("noch nie geprobt: der Start probt nicht", func(t *testing.T) {
		s.probeBeimStartNachFehlschlag()
		var n int
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM system_einstellungen WHERE schluessel = $1`,
			RestoreProbeSchluessel).Scan(&n); err != nil {
			t.Fatalf("Ergebnis zählen: %v", err)
		}
		if n != 0 {
			t.Errorf("ohne früheren Lauf steht ein Ergebnis da — der Start hat geprobt")
		}
	})

	t.Run("letzter Lauf gelungen: der Start probt nicht", func(t *testing.T) {
		s.speichereRestoreProbe(RestoreProbeErgebnis{Zeitpunkt: frueher, Erfolg: true, BackupDatei: marke, Tabellen: 40})
		s.probeBeimStartNachFehlschlag()
		e := leseProbeErgebnis(t, pool)
		if e.BackupDatei != marke || !e.Zeitpunkt.Equal(frueher) {
			t.Errorf("nach gelungenem Lauf steht ein neues Ergebnis da (%s, %s) — der Start hat geprobt",
				e.BackupDatei, e.Zeitpunkt)
		}
	})

	t.Run("letzter Lauf fehlgeschlagen: der Start probt erneut", func(t *testing.T) {
		s.speichereRestoreProbe(RestoreProbeErgebnis{Zeitpunkt: frueher, BackupDatei: marke, Fehler: "entschlüsselung fehlgeschlagen"})
		s.probeBeimStartNachFehlschlag()
		e := leseProbeErgebnis(t, pool)
		if !e.Erfolg {
			t.Fatalf("nach dem Start steht weiter ein Fehlschlag da: %s", e.Fehler)
		}
		if e.BackupDatei != filepath.Base(sicherung) {
			t.Errorf("der Start prüfte %q statt %q", e.BackupDatei, filepath.Base(sicherung))
		}
	})
}
