// Command encrypt-backup verschlüsselt einen komprimierten pg_dump von der
// Standardeingabe in das Backup-Format (`.sql.gz.enc`) auf die Standardausgabe.
//
// Es ist das Gegenstück zu cmd/restore-backup und existiert, weil die beiden
// SHELL-Wege am nächtlichen Job vorbei dumpten: `scripts/backup.sh` und die
// Vorab-Sicherung in `update.sh` legten `.sql.gz` im KLARTEXT ab — jeder Schülername,
// jede Adresse, jede Ausleihe, 7 bzw. 30 Tage lang (Befund A5,
// docs/datenschutz_offene_punkte.md). Ein Skript kann das Format nicht selbst erzeugen;
// hier ist es.
//
// Verwendung (der Schlüssel kommt aus der Umgebung, NICHT aus einem Argument — sonst
// stünde er in der Prozessliste jedes Mitlesenden):
//
//	pg_dump … | gzip | BACKUP_ENCRYPTION_KEY=<passphrase> encrypt-backup > backup.sql.gz.enc
//
// Im Betrieb läuft es im Backend-Container, der den Schlüssel ohnehin gesetzt hat:
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
	if err := run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr, os.Getenv); err != nil {
		fmt.Fprintf(os.Stderr, "encrypt-backup: %v\n", err)
		os.Exit(1)
	}
}

// run ist der ganze Ablauf, mit Ein-/Ausgabe und Umgebung als Parameter — so prüft ihn
// der Test ohne echte Dateien und ohne echte Umgebungsvariablen.
func run(args []string, stdin io.Reader, stdout, stderr io.Writer, getenv func(string) string) error {
	if len(args) > 0 {
		return fmt.Errorf("dieses Werkzeug nimmt keine Argumente; Aufruf: … | BACKUP_ENCRYPTION_KEY=<passphrase> encrypt-backup > <datei>.sql.gz.enc")
	}

	encKey := getenv("BACKUP_ENCRYPTION_KEY")
	if encKey == "" {
		return fmt.Errorf("BACKUP_ENCRYPTION_KEY nicht gesetzt — ohne Schlüssel kann nicht verschlüsselt werden")
	}
	// Warnen, nicht abbrechen: dieselbe Abwägung wie im nächtlichen Job (jobs/backup.go).
	// Ein Backup mit kurzer Passphrase ist deutlich besser als gar keins — und der
	// Aufrufer ist hier ein Deploy-Skript, dem der Abbruch die Sicherung nähme.
	if len(encKey) < mindestLaenge {
		// Fehler beim Warnen bewusst verworfen: Ein nicht schreibbares stderr darf das Backup nicht verhindern.
		fmt.Fprintf(stderr, "encrypt-backup: WARNUNG — BACKUP_ENCRYPTION_KEY ist nur %d Zeichen lang (empfohlen: >= %d)\n", len(encKey), mindestLaenge) //nolint:errcheck
	}

	klartext, err := io.ReadAll(stdin)
	if err != nil {
		return fmt.Errorf("eingabe konnte nicht gelesen werden: %w", err)
	}
	if err := pruefeGzip(klartext); err != nil {
		return err
	}

	verschluesselt, err := backupkrypto.VerschluesseleBackup(encKey, klartext)
	if err != nil {
		return fmt.Errorf("verschlüsselung fehlgeschlagen: %w", err)
	}

	// Der Rückweg wird BEWIESEN, bevor ein einziges Byte herausgeht.
	//
	// Der Aufrufer wirft den Klartext-Dump nach diesem Aufruf weg — genau das ist der
	// Zweck. Wäre die Ausgabe unlesbar (falsch abgeleiteter Schlüssel, halb geschriebener
	// Puffer), stünde am Ende eine Datei, die wie ein Backup aussieht und keines ist;
	// bemerkt würde das erst beim Wiederherstellen, also im schlechtestmöglichen Moment.
	// Die Gegenprobe kostet einen zweiten scrypt-Durchlauf (~100 ms) — gemessen am
	// Risiko ist das nichts.
	zurueck, err := backupkrypto.EntschluesseleBackup(encKey, verschluesselt)
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

// mindestLaenge spiegelt jobs.MinBackupSchluesselLaenge. Bewusst als eigene Konstante:
// das Werkzeug soll ohne das cgo-behaftete jobs-Paket bauen (internal/backupkrypto).
const mindestLaenge = 32

// pruefeGzip lehnt ab, was kein gzip-Strom mit Inhalt ist.
//
// Ohne diese Prüfung wäre der stille Ausfall wieder möglich, den `pipefail` in den
// Skripten schon einmal abgestellt hat: Bricht pg_dump ab, liefert die Pipe entweder
// nichts oder ein gzip mit einer Fehlermeldung darin. Verschlüsselt sähe beides aus wie
// ein Backup — dieselbe Endung, plausible Größe, unlesbar erst im Ernstfall. Der
// vollständige Entpack-Durchlauf (statt nur der Magic-Bytes) fängt zusätzlich den
// abgeschnittenen Strom ab: Bei ihm fehlt am Ende die Prüfsumme.
func pruefeGzip(daten []byte) error {
	if len(daten) == 0 {
		return fmt.Errorf("leere eingabe — es gibt nichts zu verschlüsseln (ist pg_dump fehlgeschlagen?)")
	}
	leser, err := gzip.NewReader(bytes.NewReader(daten))
	if err != nil {
		return fmt.Errorf("eingabe ist kein gültiger gzip-Strom (erwartet: pg_dump | gzip): %w", err)
	}
	defer func() { _ = leser.Close() }() //nolint:errcheck
	n, err := io.Copy(io.Discard, leser)
	if err != nil {
		return fmt.Errorf("eingabe ist ein beschädigter gzip-Strom (abgebrochener Dump?): %w", err)
	}
	if n == 0 {
		return fmt.Errorf("eingabe enthält einen leeren Dump — es gibt nichts zu verschlüsseln")
	}
	return nil
}
