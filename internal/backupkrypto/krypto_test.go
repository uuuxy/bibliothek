package backupkrypto

import (
	"bytes"
	"compress/gzip"
	"io"
	"testing"
)

// gzipBytes komprimiert wie die Backup-Pipeline in RunDatabaseBackup.
func gzipBytes(t *testing.T, data []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	if _, err := gz.Write(data); err != nil {
		t.Fatalf("gzip write: %v", err)
	}
	if err := gz.Close(); err != nil {
		t.Fatalf("gzip close: %v", err)
	}
	return buf.Bytes()
}

func gunzipBytes(t *testing.T, data []byte) []byte {
	t.Helper()
	gz, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("gzip reader: %v", err)
	}
	out, err := io.ReadAll(gz)
	if err != nil {
		t.Fatalf("gunzip read: %v", err)
	}
	return out
}

// TestBackupRestoreRoundTrip beweist, dass Verschlüsselung (jobs.RunDatabaseBackup und
// cmd/encrypt-backup, jetzt scrypt) und Entschlüsselung (cmd/restore-backup) ein
// zusammenpassendes Paar sind. Schlägt
// er fehl, sind die automatischen .sql.gz.enc-Backups NICHT wiederherstellbar.
func TestBackupRestoreRoundTrip(t *testing.T) {
	passphrase := "produktions-passphrase-mind-32-zeichen-xx"
	originalSQL := []byte("-- pg_dump\nCREATE TABLE schueler (id uuid PRIMARY KEY);\nINSERT INTO schueler VALUES ('00000000-0000-0000-0000-000000000001');\n")

	compressed := gzipBytes(t, originalSQL)
	encrypted, err := VerschluesseleBackup(passphrase, compressed)
	if err != nil {
		t.Fatalf("VerschluesseleBackup: %v", err)
	}

	// Das neue Format MUSS das Magic tragen — sonst würde es als Altformat gelesen.
	if !IstScryptFormat(encrypted) {
		t.Fatal("neues Backup trägt nicht das scrypt-Magic")
	}

	decrypted, err := EntschluesseleBackup(passphrase, encrypted)
	if err != nil {
		t.Fatalf("DecryptBackup: %v", err)
	}
	if !bytes.Equal(gunzipBytes(t, decrypted), originalSQL) {
		t.Errorf("wiederhergestelltes SQL weicht ab")
	}
}

// TestBackupRestore_AltformatWirdAbgelehnt hält fest, dass der schwache SHA-256-Weg
// GANZ entfernt ist: Eine Datei ohne die scrypt-Format-Kennung (so begannen die alten
// Backups: direkt mit [Nonce][Ciphertext]) wird abgelehnt, nicht mehr schwach
// entschlüsselt. Im Pilotbetrieb gibt es keine schützenswerten Altbackups; ein
// mitgeschleppter SHA-256-Pfad wäre nur ein dauerhafter Schwachpunkt.
func TestBackupRestore_AltformatWirdAbgelehnt(t *testing.T) {
	// Beliebige Bytes ohne "BKDF"-Magic stehen für ein Backup im alten Format.
	altbackup := bytes.Repeat([]byte{0x01}, 64)
	if IstScryptFormat(altbackup) {
		t.Fatal("Testdaten dürfen das scrypt-Magic nicht tragen")
	}
	if _, err := EntschluesseleBackup("egal-passphrase-mind-32-zeichen-xxxxxxxx", altbackup); err == nil {
		t.Fatal("ein Backup ohne scrypt-Format-Kennung muss abgelehnt werden (kein schwacher SHA-256-Fallback)")
	}
}

func TestBackupRestore_WrongKeyFails(t *testing.T) {
	encrypted, err := VerschluesseleBackup("richtige-passphrase-mind-32-zeichen-xx", gzipBytes(t, []byte("geheime daten")))
	if err != nil {
		t.Fatalf("VerschluesseleBackup: %v", err)
	}
	// Falscher Schlüssel muss an der GCM-Authentifizierung scheitern (kein stiller Müll).
	if _, err := EntschluesseleBackup("FALSCHE-passphrase-mind-32-zeichen-xxxx", encrypted); err == nil {
		t.Error("Entschlüsselung mit falschem Schlüssel soll fehlschlagen, war aber erfolgreich")
	}
}

func TestBackupRestore_TruncatedCiphertextFails(t *testing.T) {
	if _, err := EntschluesseleBackup("egal-passphrase-mind-32-zeichen-xxxxxxxx", []byte{0x01, 0x02}); err == nil {
		t.Error("zu kurzer Ciphertext (< Nonce) soll Fehler liefern")
	}
}

func TestBackupRestore_TamperedCiphertextFails(t *testing.T) {
	encrypted, err := VerschluesseleBackup("passphrase-mind-32-zeichen-xxxxxxxxxxxx", gzipBytes(t, []byte("integritaet")))
	if err != nil {
		t.Fatalf("VerschluesseleBackup: %v", err)
	}
	// Letztes Byte (im GCM-Tag-Bereich) kippen → Manipulation muss erkannt werden.
	encrypted[len(encrypted)-1] ^= 0xFF
	if _, err := EntschluesseleBackup("passphrase-mind-32-zeichen-xxxxxxxxxxxx", encrypted); err == nil {
		t.Error("manipulierter Ciphertext soll an der GCM-Integritätsprüfung scheitern")
	}
}
