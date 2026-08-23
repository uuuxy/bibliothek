package jobs

// restore_probe_hilfen.go — Datei- und Entschlüsselungs-Helfer der Restore-Probe
// (restore_probe.go): jüngstes Backup finden, AES-GCM + gzip öffnen, psql einspielen,
// Gegenprobe zählen.

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	"bibliothek/internal/backupkrypto"
)

// juengsteBackupDatei liefert die neueste backup_*.sql.gz.enc im Verzeichnis.
func juengsteBackupDatei(dir string) (string, error) {
	treffer, err := filepath.Glob(filepath.Join(dir, "backup_*.sql.gz.enc"))
	if err != nil || len(treffer) == 0 {
		return "", fmt.Errorf("keine Backup-Datei in %s gefunden", dir)
	}
	juengste, juengsteZeit := "", time.Time{}
	for _, pfad := range treffer {
		info, statErr := os.Stat(pfad)
		if statErr == nil && info.ModTime().After(juengsteZeit) {
			juengste, juengsteZeit = pfad, info.ModTime()
		}
	}
	if juengste == "" {
		return "", fmt.Errorf("keine lesbare Backup-Datei in %s", dir)
	}
	return juengste, nil
}

// entschluesseleBackup entschlüsselt (AES-256-GCM) und entpackt (gzip) ein Backup.
func entschluesseleBackup(encKey string, roh []byte) ([]byte, error) {
	klar, err := backupkrypto.EntschluesseleBackup(encKey, roh)
	if err != nil {
		return nil, fmt.Errorf("entschlüsselung fehlgeschlagen (falscher Schlüssel oder beschädigte Datei): %w", err)
	}
	leser, err := gzip.NewReader(bytes.NewReader(klar))
	if err != nil {
		return nil, fmt.Errorf("entpacken fehlgeschlagen: %w", err)
	}
	defer leser.Close() //nolint:errcheck
	sqlText, err := io.ReadAll(leser)
	if err != nil {
		return nil, fmt.Errorf("entpacken fehlgeschlagen: %w", err)
	}
	return sqlText, nil
}

// spieleDumpEin schiebt den Dump per psql in die Wegwerf-Datenbank. ON_ERROR_STOP=1
// ist Pflicht: ohne den Schalter endet psql auch nach gescheiterten Statements mit 0,
// und eine halbe Wiederherstellung sähe erfolgreich aus (siehe CI-Drill).
func spieleDumpEin(ctx context.Context, dsn string, sqlText []byte) error {
	config, err := pgconn.ParseConfig(dsn)
	if err != nil {
		return fmt.Errorf("DSN unlesbar: %w", err)
	}
	config.Database = restoreProbeDBName
	passDatei, err := createPgPassFile(config)
	if err != nil {
		return err
	}
	defer os.Remove(passDatei) //nolint:errcheck

	port := fmt.Sprintf("%d", config.Port)
	if port == "0" {
		port = "5432"
	}
	// #nosec G204 - Argumente stammen aus dem geparsten DSN der eigenen Konfiguration
	psql := exec.CommandContext(ctx, "psql",
		"--host="+config.Host, "--port="+port, "--username="+config.User,
		"--dbname="+restoreProbeDBName, "--no-password", "--no-psqlrc", "--quiet",
		"-v", "ON_ERROR_STOP=1")
	psql.Env = append(os.Environ(), "PGPASSFILE="+passDatei)
	psql.Stdin = bytes.NewReader(sqlText)
	var stderr bytes.Buffer
	psql.Stderr = &stderr
	if err := psql.Run(); err != nil {
		return fmt.Errorf("psql-Wiederherstellung fehlgeschlagen: %v — %s", err, kuerzeFehlertext(stderr.String()))
	}
	return nil
}

// zaehleProbeTabellen verbindet sich mit der Wegwerf-Datenbank und zählt die Tabellen.
func zaehleProbeTabellen(ctx context.Context, dsn string) (int, error) {
	config, err := pgconn.ParseConfig(dsn)
	if err != nil {
		return 0, err
	}
	config.Database = restoreProbeDBName
	conn, err := pgconn.ConnectConfig(ctx, config)
	if err != nil {
		return 0, err
	}
	defer conn.Close(ctx) //nolint:errcheck
	res := conn.ExecParams(ctx,
		`SELECT count(*)::text FROM information_schema.tables WHERE table_schema = 'public'`,
		nil, nil, nil, nil).Read()
	if res.Err != nil || len(res.Rows) == 0 {
		return 0, fmt.Errorf("tabellen nicht zählbar: %v", res.Err)
	}
	var anzahl int
	if _, err := fmt.Sscanf(string(res.Rows[0][0]), "%d", &anzahl); err != nil {
		return 0, err
	}
	return anzahl, nil
}
