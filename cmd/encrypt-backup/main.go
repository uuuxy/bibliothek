// Command encrypt-backup verschlüsselt einen gepackten pg_dump von der Standardeingabe in das
// Format der Sicherung (`.sql.gz.enc`) auf die Standardausgabe.
//
// Es ist das Gegenstück zu cmd/restore-backup. Die Sicherungen der Shell entstehen nicht im
// nächtlichen Lauf: `scripts/backup.sh` und die Sicherung vor einem Update in `update.sh`
// lägen sonst als `.sql.gz` im Klartext, mit jedem Namen, jeder Adresse und jeder Ausleihe,
// 7 oder 30 Tage lang. Ein Skript kann das Format nicht selbst erzeugen; hier ist es.
//
// Aufruf (der Schlüssel kommt aus der Umgebung, nicht aus einem Argument, sonst stünde er in
// der Prozessliste für jeden lesbar):
//
//	pg_dump … | gzip | BACKUP_ENCRYPTION_KEY=<passphrase> encrypt-backup > backup.sql.gz.enc
//
// Im Betrieb läuft es im Container des Backends, der den Schlüssel ohnehin gesetzt hat:
//
//	docker exec -i bibliothek-backend ./encrypt-backup < dump.sql.gz > backup.sql.gz.enc
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
	if err := fuehreAus(os.Args[1:], os.Stdin, os.Stdout, os.Stderr, os.Getenv); err != nil {
		fmt.Fprintf(os.Stderr, "encrypt-backup: %v\n", err)
		os.Exit(1)
	}
}

// fuehreAus ist der ganze Ablauf, mit Ein- und Ausgabe und Umgebung als Parameter; so prüft
// ihn der Test ohne echte Dateien und ohne echte Umgebungsvariablen.
func fuehreAus(argumente []string, stdin io.Reader, stdout, stderr io.Writer, umgebung func(string) string) error {
	if len(argumente) > 0 {
		return fmt.Errorf("dieses Werkzeug nimmt keine Argumente; Aufruf: … | BACKUP_ENCRYPTION_KEY=<passphrase> encrypt-backup > <datei>.sql.gz.enc")
	}

	schluessel := umgebung("BACKUP_ENCRYPTION_KEY")
	if schluessel == "" {
		return fmt.Errorf("BACKUP_ENCRYPTION_KEY nicht gesetzt — ohne Schlüssel kann nicht verschlüsselt werden")
	}
	// Warnen, nicht abbrechen: dieselbe Abwägung wie im nächtlichen Lauf (jobs/backup.go). Eine
	// Sicherung mit kurzer Passphrase ist viel besser als gar keine, und der Aufrufer ist hier
	// ein Skript des Deploys, dem der Abbruch die Sicherung nähme.
	if len(schluessel) < mindestLaenge {
		// Ein Fehler beim Warnen wird verworfen: Ein nicht schreibbares stderr darf die
		// Sicherung nicht verhindern.
		fmt.Fprintf(stderr, "encrypt-backup: WARNUNG — BACKUP_ENCRYPTION_KEY ist nur %d Zeichen lang (empfohlen: >= %d)\n", len(schluessel), mindestLaenge) //nolint:errcheck
	}

	klartext, err := io.ReadAll(stdin)
	if err != nil {
		return fmt.Errorf("eingabe konnte nicht gelesen werden: %w", err)
	}
	if err := pruefeGzip(klartext); err != nil {
		return err
	}

	verschluesselt, err := backupkrypto.Verschluessele(schluessel, klartext)
	if err != nil {
		return fmt.Errorf("verschlüsselung fehlgeschlagen: %w", err)
	}

	// Der Rückweg wird belegt, bevor ein einziges Byte hinausgeht.
	//
	// Der Aufrufer wirft den Dump im Klartext nach diesem Aufruf weg; genau das ist der Zweck.
	// Wäre die Ausgabe unlesbar (falsch abgeleiteter Schlüssel, halb geschriebener Puffer),
	// stünde am Ende eine Datei, die wie eine Sicherung aussieht und keine ist; bemerkt würde
	// das erst beim Wiederherstellen, im schlechtesten Moment. Die Gegenprobe kostet einen
	// zweiten Durchlauf von scrypt (etwa 100 ms); gemessen am Risiko ist das nichts.
	zurueck, err := backupkrypto.Entschluessele(schluessel, verschluesselt)
	if err != nil {
		return fmt.Errorf("gegenprobe fehlgeschlagen — die verschlüsselte Ausgabe ließe sich nicht wiederherstellen: %w", err)
	}
	if !bytes.Equal(zurueck, klartext) {
		return fmt.Errorf("gegenprobe fehlgeschlagen — entschlüsselter Inhalt weicht vom Eingang ab (%d statt %d Bytes)", len(zurueck), len(klartext))
	}

	if _, err := stdout.Write(verschluesselt); err != nil {
		return fmt.Errorf("ausgabe konnte nicht geschrieben werden: %w", err)
	}
	return nil
}

// mindestLaenge entspricht jobs.MinBackupSchluesselLaenge. Eine eigene Konstante, damit das
// Werkzeug ohne das Paket jobs baut, das über die Cover cgo braucht; die Verschlüsselung selbst
// steht in internal/backupkrypto.
const mindestLaenge = 32

// pruefeGzip lehnt ab, was kein gzip-Strom mit Inhalt ist.
//
// Ohne die Prüfung käme der stumme Ausfall zurück, gegen den die Skripte `pipefail` setzen:
// Bricht pg_dump ab, liefert die Leitung entweder nichts oder ein gzip mit einer Fehlermeldung
// darin. Verschlüsselt sähe beides aus wie eine Sicherung, mit derselben Endung und plausibler
// Größe, unlesbar erst im Ernstfall. Das vollständige Entpacken (statt nur der ersten Bytes)
// fängt zusätzlich den abgeschnittenen Strom ab: Bei ihm fehlt am Ende die Prüfsumme.
func pruefeGzip(daten []byte) error {
	if len(daten) == 0 {
		return fmt.Errorf("leere eingabe — es gibt nichts zu verschlüsseln (ist pg_dump fehlgeschlagen?)")
	}
	leser, err := gzip.NewReader(bytes.NewReader(daten))
	if err != nil {
		return fmt.Errorf("eingabe ist kein gültiger gzip-Strom (erwartet: pg_dump | gzip): %w", err)
	}
	defer func() { _ = leser.Close() }() //nolint:errcheck

	// Höchstens 100 GB entpacken, gegen eine Entpack-Bombe (G110).
	grenze := int64(100 << 30)
	begrenzt := io.LimitReader(leser, grenze)
	n, err := io.Copy(io.Discard, begrenzt)
	if err != nil {
		return fmt.Errorf("eingabe ist ein beschädigter gzip-Strom (abgebrochener Dump?): %w", err)
	}
	if n == grenze {
		// Liegt dahinter noch etwas, ist das ein Fehler und kein stilles Abschneiden.
		rest := make([]byte, 1)
		if _, err := leser.Read(rest); err != io.EOF {
			return fmt.Errorf("eingabe ist zu groß: maximale größe (100GB) überschritten")
		}
	}
	if n == 0 {
		return fmt.Errorf("eingabe enthält einen leeren Dump — es gibt nichts zu verschlüsseln")
	}
	return nil
}
