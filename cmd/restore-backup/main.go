// Command restore-backup entschlüsselt und dekomprimiert ein vom täglichen
// Backup-Job erzeugtes .sql.gz.enc-Backup zurück zu reinem SQL.
//
// Das automatische Backup (jobs.RunDatabaseBackup) erzeugt AES-256-GCM-
// verschlüsselte, gzip-komprimierte pg_dump-Dateien. Plain-Tools wie
// `zcat | psql` funktionieren darauf NICHT — dieses Tool ist der unterstützte
// Entschlüsselungsschritt der Disaster-Recovery.
//
// Verwendung:
//
//	BACKUP_ENCRYPTION_KEY=<passphrase> restore-backup <backup.sql.gz.enc> [ausgabe.sql]
//
// Ohne Ausgabedatei wird das SQL nach stdout geschrieben, sodass direkt
// weitergeleitet werden kann:
//
//	BACKUP_ENCRYPTION_KEY=… restore-backup backup_….sql.gz.enc | psql -U postgres -d bibliothek
package main

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"os"

	"bibliothek/jobs"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "restore-backup: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	args := os.Args[1:]
	if len(args) < 1 || len(args) > 2 {
		return fmt.Errorf("usage: BACKUP_ENCRYPTION_KEY=<passphrase> restore-backup <backup.sql.gz.enc> [ausgabe.sql]")
	}

	encKey := os.Getenv("BACKUP_ENCRYPTION_KEY")
	if encKey == "" {
		return fmt.Errorf("BACKUP_ENCRYPTION_KEY nicht gesetzt — ohne den Original-Schlüssel ist das Backup nicht entschlüsselbar")
	}

	inputPath := args[0]
	ciphertext, err := os.ReadFile(inputPath) // #nosec G304 - Pfad ist ein bewusst übergebenes Operator-Argument
	if err != nil {
		return fmt.Errorf("backup-datei %q konnte nicht gelesen werden: %w", inputPath, err)
	}

	// 1. AES-256-GCM entschlüsseln (identische Schlüsselableitung wie beim Backup).
	compressed, err := jobs.DecryptBackup(encKey, ciphertext)
	if err != nil {
		return fmt.Errorf("entschlüsselung fehlgeschlagen (falscher Schlüssel oder beschädigte/manipulierte Datei?): %w", err)
	}

	// 2. gzip dekomprimieren → reines pg_dump-SQL.
	gz, err := gzip.NewReader(bytes.NewReader(compressed))
	if err != nil {
		return fmt.Errorf("gzip-header ungültig (kein gültiges Backup?): %w", err)
	}
	defer func() { _ = gz.Close() }()

	// 3. Ziel bestimmen: Datei oder stdout (für `| psql`).
	var out io.Writer = os.Stdout
	if len(args) == 2 {
		f, err := os.OpenFile(args[1], os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600) // #nosec G304 - Operator-Argument
		if err != nil {
			return fmt.Errorf("ausgabedatei %q konnte nicht erstellt werden: %w", args[1], err)
		}
		defer func() { _ = f.Close() }()
		out = f
	}

	// #nosec G110
	//
	if _, err := io.Copy(out, gz); err != nil {
		return fmt.Errorf("dekomprimierung fehlgeschlagen: %w", err)
	}

	if len(args) == 2 {
		fmt.Fprintf(os.Stderr, "restore-backup: SQL nach %q geschrieben. Einspielen mit: psql -U <user> -d <db> -f %s\n", args[1], args[1])
	}
	return nil
}
