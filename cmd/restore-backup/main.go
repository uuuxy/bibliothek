// Command restore-backup entschlüsselt und entpackt eine Sicherung des täglichen Laufs
// (.sql.gz.enc) zurück zu reinem SQL.
//
// Die automatische Sicherung (jobs.BackupLauf) erzeugt mit AES-256-GCM verschlüsselte, mit
// gzip gepackte Dateien von pg_dump. Einfache Werkzeuge wie `zcat | psql` helfen dort nicht;
// dieses Werkzeug ist der vorgesehene Schritt der Entschlüsselung bei der Wiederherstellung.
//
// Aufruf:
//
//	BACKUP_ENCRYPTION_KEY=<passphrase> restore-backup <backup.sql.gz.enc> [ausgabe.sql]
//
// Ohne Ausgabedatei geht das SQL nach stdout und lässt sich direkt weiterleiten:
//
//	BACKUP_ENCRYPTION_KEY=… restore-backup backup_….sql.gz.enc | psql -U postgres -d bibliothek
package main

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"os"

	"bibliothek/internal/backupkrypto"
)

func main() {
	if err := fuehreAus(); err != nil {
		fmt.Fprintf(os.Stderr, "restore-backup: %v\n", err)
		os.Exit(1)
	}
}

// fuehreAus ist der ganze Ablauf: Datei lesen, entschlüsseln, entpacken, ausgeben.
func fuehreAus() error {
	argumente := os.Args[1:]
	if len(argumente) < 1 || len(argumente) > 2 {
		return fmt.Errorf("usage: BACKUP_ENCRYPTION_KEY=<passphrase> restore-backup <backup.sql.gz.enc> [ausgabe.sql]")
	}

	schluessel := os.Getenv("BACKUP_ENCRYPTION_KEY")
	if schluessel == "" {
		return fmt.Errorf("BACKUP_ENCRYPTION_KEY nicht gesetzt — ohne den Original-Schlüssel ist das Backup nicht entschlüsselbar")
	}

	eingabe := argumente[0]
	verschluesselt, err := os.ReadFile(eingabe) // #nosec G304 - Pfad ist ein bewusst übergebenes Argument des Betreibers
	if err != nil {
		return fmt.Errorf("backup-datei %q konnte nicht gelesen werden: %w", eingabe, err)
	}

	// 1. Mit AES-256-GCM entschlüsseln (derselbe Schlüssel wie beim Verschlüsseln).
	gepackt, err := backupkrypto.Entschluessele(schluessel, verschluesselt)
	if err != nil {
		return fmt.Errorf("entschlüsselung fehlgeschlagen (falscher Schlüssel oder beschädigte/manipulierte Datei?): %w", err)
	}

	// 2. gzip entpacken: reines SQL von pg_dump.
	entpacker, err := gzip.NewReader(bytes.NewReader(gepackt))
	if err != nil {
		return fmt.Errorf("gzip-header ungültig (kein gültiges Backup?): %w", err)
	}
	defer func() { _ = entpacker.Close() }() //nolint:errcheck

	// 3. Ziel bestimmen: Datei oder stdout (für `| psql`).
	var ziel io.Writer = os.Stdout
	if len(argumente) == 2 {
		datei, err := os.OpenFile(argumente[1], os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600) // #nosec G304 - Argument des Betreibers
		if err != nil {
			return fmt.Errorf("ausgabedatei %q konnte nicht erstellt werden: %w", argumente[1], err)
		}
		defer func() { _ = datei.Close() }() //nolint:errcheck
		ziel = datei
	}

	// Höchstens 100 GB entpacken, gegen eine Entpack-Bombe (G110).
	grenze := int64(100 << 30)
	begrenzt := io.LimitReader(entpacker, grenze)

	n, err := io.Copy(ziel, begrenzt)
	if err != nil {
		return fmt.Errorf("dekomprimierung fehlgeschlagen: %w", err)
	}
	if n == grenze {
		// Liegt dahinter noch etwas, ist das ein Fehler und kein stilles Abschneiden.
		rest := make([]byte, 1)
		if _, err := entpacker.Read(rest); err != io.EOF {
			return fmt.Errorf("dekomprimierung fehlgeschlagen: maximale größe (100GB) überschritten")
		}
	}

	if len(argumente) == 2 {
		fmt.Fprintf(os.Stderr, "restore-backup: SQL nach %q geschrieben. Einspielen mit: psql -U <user> -d <db> -f %s\n", argumente[1], argumente[1])
	}
	return nil
}
