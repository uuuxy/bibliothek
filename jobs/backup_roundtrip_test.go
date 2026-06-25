package jobs

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
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

// TestBackupRestoreRoundTrip beweist, dass die Verschlüsselung beim Backup und die
// Entschlüsselung bei der Wiederherstellung ein zusammenpassendes Paar sind — durch
// dieselbe SHA-256-Schlüsselableitung wie in der Produktion. Schlägt dieser Test fehl,
// sind die automatischen .sql.gz.enc-Backups NICHT wiederherstellbar.
func TestBackupRestoreRoundTrip(t *testing.T) {
	passphrase := "produktions-passphrase-mind-32-zeichen-xx"
	originalSQL := []byte("-- pg_dump\nCREATE TABLE schueler (id uuid PRIMARY KEY);\nINSERT INTO schueler VALUES ('00000000-0000-0000-0000-000000000001');\n")

	// --- Backup-Seite: exakt wie RunDatabaseBackup ---
	keyBytes := sha256.Sum256([]byte(passphrase))
	compressed := gzipBytes(t, originalSQL)
	encrypted, err := encryptAESGCM(keyBytes[:], compressed)
	if err != nil {
		t.Fatalf("encryptAESGCM: %v", err)
	}

	// --- Restore-Seite: exakt wie DecryptBackup (Disaster Recovery) ---
	decrypted, err := DecryptBackup(passphrase, encrypted)
	if err != nil {
		t.Fatalf("DecryptBackup: %v", err)
	}
	recoveredSQL := gunzipBytes(t, decrypted)

	if !bytes.Equal(recoveredSQL, originalSQL) {
		t.Errorf("wiederhergestelltes SQL weicht ab:\n got: %q\nwant: %q", recoveredSQL, originalSQL)
	}
}

func TestBackupRestore_WrongKeyFails(t *testing.T) {
	keyBytes := sha256.Sum256([]byte("richtige-passphrase-mind-32-zeichen-xx"))
	encrypted, err := encryptAESGCM(keyBytes[:], gzipBytes(t, []byte("geheime daten")))
	if err != nil {
		t.Fatalf("encryptAESGCM: %v", err)
	}

	// Falscher Schlüssel muss an der GCM-Authentifizierung scheitern (kein stiller Müll).
	if _, err := DecryptBackup("FALSCHE-passphrase-mind-32-zeichen-xxxx", encrypted); err == nil {
		t.Error("Entschlüsselung mit falschem Schlüssel soll fehlschlagen, war aber erfolgreich")
	}
}

func TestBackupRestore_TruncatedCiphertextFails(t *testing.T) {
	if _, err := DecryptBackup("egal-passphrase-mind-32-zeichen-xxxxxxxx", []byte{0x01, 0x02}); err == nil {
		t.Error("zu kurzer Ciphertext (< Nonce) soll Fehler liefern")
	}
}

func TestBackupRestore_TamperedCiphertextFails(t *testing.T) {
	keyBytes := sha256.Sum256([]byte("passphrase-mind-32-zeichen-xxxxxxxxxxxx"))
	encrypted, err := encryptAESGCM(keyBytes[:], gzipBytes(t, []byte("integritaet")))
	if err != nil {
		t.Fatalf("encryptAESGCM: %v", err)
	}
	// Letztes Byte (im GCM-Tag-Bereich) kippen → Manipulation muss erkannt werden.
	encrypted[len(encrypted)-1] ^= 0xFF
	if _, err := DecryptBackup("passphrase-mind-32-zeichen-xxxxxxxxxxxx", encrypted); err == nil {
		t.Error("manipulierter Ciphertext soll an der GCM-Integritätsprüfung scheitern")
	}
}
