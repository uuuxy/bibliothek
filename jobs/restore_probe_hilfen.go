package jobs

// Die Helfer der Restore-Probe (restore_probe.go) für Datei und Entschlüsselung: jüngste
// Sicherung finden, AES-GCM und gzip öffnen, mit psql einspielen, zur Gegenprobe zählen.

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/jackc/pgx/v5/pgconn"

	"bibliothek/internal/backupkrypto"
)

// juengsteBackupDatei liefert die neueste backup_*.sql.gz.enc im Verzeichnis, nach
// BackupDatei.SicherungsZeit: Nach dem Zurückspielen des Ordners sagt die Änderungszeit nichts
// über das Alter einer Sicherung.
func juengsteBackupDatei(verzeichnis string) (string, error) {
	dateien, err := LeseBackupDateien(verzeichnis)
	if err != nil {
		return "", fmt.Errorf("keine Backup-Datei lesbar in %s: %w", verzeichnis, err)
	}
	if len(dateien) == 0 {
		return "", fmt.Errorf("keine Backup-Datei in %s gefunden", verzeichnis)
	}
	juengste := dateien[0]
	for _, d := range dateien[1:] {
		if d.SicherungsZeit().After(juengste.SicherungsZeit()) {
			juengste = d
		}
	}
	return filepath.Join(verzeichnis, juengste.Name), nil
}

// entschluesseleBackup entschlüsselt (AES-256-GCM) ein Backup und liefert den gepackten Dump.
// Den Kopf von gzip prüft es schon hier, damit eine Datei, die kein gzip ist, vor dem Anlegen der
// Wegwerf-Datenbank scheitert.
func entschluesseleBackup(schluessel string, roh []byte) ([]byte, error) {
	gepackt, err := backupkrypto.Entschluessele(schluessel, roh)
	if err != nil {
		return nil, fmt.Errorf("entschlüsselung fehlgeschlagen (falscher Schlüssel oder beschädigte Datei): %w", err)
	}
	if _, err := gzip.NewReader(bytes.NewReader(gepackt)); err != nil {
		return nil, fmt.Errorf("entpacken fehlgeschlagen: %w", err)
	}
	return gepackt, nil
}

// spieleDumpEin entpackt den Dump im Fluss in psql und spielt ihn so in die Wegwerf-Datenbank ein:
// Ganz entpackt im Speicher brauchte die Probe mit 1.900 Passbildern 1,3 GB. Scheitert das
// Entpacken mittendrin, meldet psql.Run den Fehler auch dann, wenn psql selbst mit 0 endet.
// ON_ERROR_STOP=1 ist Pflicht: Ohne den Schalter endet psql auch nach gescheiterten Anweisungen
// mit 0, und eine halbe Wiederherstellung sähe gelungen aus (wie in der Übung der CI).
func spieleDumpEin(ctx context.Context, dsn string, gepackt []byte) error {
	sqlLeser, err := gzip.NewReader(bytes.NewReader(gepackt))
	if err != nil {
		return fmt.Errorf("entpacken fehlgeschlagen: %w", err)
	}
	defer sqlLeser.Close() //nolint:errcheck

	konfig, err := pgconn.ParseConfig(dsn)
	if err != nil {
		return fmt.Errorf("DSN unlesbar: %w", err)
	}
	konfig.Database = probeDatenbank
	pgpassDatei, err := erzeugePgpassDatei(konfig)
	if err != nil {
		return err
	}
	defer os.Remove(pgpassDatei) //nolint:errcheck

	port := fmt.Sprintf("%d", konfig.Port)
	if port == "0" {
		port = "5432"
	}
	// #nosec G204 - Argumente stammen aus dem geparsten DSN der eigenen Konfiguration
	psql := exec.CommandContext(ctx, "psql",
		"--host="+konfig.Host, "--port="+port, "--username="+konfig.User,
		"--dbname="+probeDatenbank, "--no-password", "--no-psqlrc", "--quiet",
		"-v", "ON_ERROR_STOP=1")
	psql.Env = append(os.Environ(), "PGPASSFILE="+pgpassDatei)
	psql.Stdin = sqlLeser
	var stderr bytes.Buffer
	psql.Stderr = &stderr
	if err := psql.Run(); err != nil {
		return fmt.Errorf("psql-Wiederherstellung fehlgeschlagen: %w — %s", err, kuerzeFehlertext(stderr.String()))
	}
	return nil
}

// zaehleProbeTabellen verbindet sich mit der Wegwerf-Datenbank und zählt die Tabellen.
func zaehleProbeTabellen(ctx context.Context, dsn string) (int, error) {
	konfig, err := pgconn.ParseConfig(dsn)
	if err != nil {
		return 0, err
	}
	konfig.Database = probeDatenbank
	verbindung, err := pgconn.ConnectConfig(ctx, konfig)
	if err != nil {
		return 0, err
	}
	defer verbindung.Close(ctx) //nolint:errcheck
	ergebnis := verbindung.ExecParams(ctx,
		`SELECT count(*)::text FROM information_schema.tables WHERE table_schema = 'public'`,
		nil, nil, nil, nil).Read()
	if ergebnis.Err != nil {
		return 0, fmt.Errorf("tabellen nicht zählbar: %w", ergebnis.Err)
	}
	if len(ergebnis.Rows) == 0 {
		return 0, errors.New("tabellen nicht zählbar: keine Zeile")
	}
	var anzahl int
	if _, err := fmt.Sscanf(string(ergebnis.Rows[0][0]), "%d", &anzahl); err != nil {
		return 0, err
	}
	return anzahl, nil
}
