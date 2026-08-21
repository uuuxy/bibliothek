package jobs

import (
	"bytes"
	"compress/gzip"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
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

// altformatVerschluesseln erzeugt ein Backup im ALTEN Format (SHA-256(passphrase),
// [Nonce][Ciphertext]) — so, wie RunDatabaseBackup es bis zum 21.08.2026 schrieb.
// Nur für den Rückwärtskompatibilitäts-Test: Ein auf Produktion liegendes Altbackup
// muss weiter öffenbar sein, sonst wäre der Format-Wechsel ein Datenverlust.
func altformatVerschluesseln(t *testing.T, passphrase string, klartext []byte) []byte {
	t.Helper()
	key := sha256.Sum256([]byte(passphrase))
	block, err := aes.NewCipher(key[:])
	if err != nil {
		t.Fatalf("aes: %v", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		t.Fatalf("gcm: %v", err)
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		t.Fatalf("nonce: %v", err)
	}
	return gcm.Seal(nonce, nonce, klartext, nil)
}

// TestBackupRestoreRoundTrip beweist, dass Verschlüsselung (RunDatabaseBackup, jetzt
// scrypt) und Entschlüsselung (DecryptBackup) ein zusammenpassendes Paar sind. Schlägt
// er fehl, sind die automatischen .sql.gz.enc-Backups NICHT wiederherstellbar.
func TestBackupRestoreRoundTrip(t *testing.T) {
	passphrase := "produktions-passphrase-mind-32-zeichen-xx"
	originalSQL := []byte("-- pg_dump\nCREATE TABLE schueler (id uuid PRIMARY KEY);\nINSERT INTO schueler VALUES ('00000000-0000-0000-0000-000000000001');\n")

	compressed := gzipBytes(t, originalSQL)
	encrypted, err := verschluesseleBackup(passphrase, compressed)
	if err != nil {
		t.Fatalf("verschluesseleBackup: %v", err)
	}

	// Das neue Format MUSS das Magic tragen — sonst würde es als Altformat gelesen.
	if !istScryptFormat(encrypted) {
		t.Fatal("neues Backup trägt nicht das scrypt-Magic")
	}

	decrypted, err := DecryptBackup(passphrase, encrypted)
	if err != nil {
		t.Fatalf("DecryptBackup: %v", err)
	}
	if !bytes.Equal(gunzipBytes(t, decrypted), originalSQL) {
		t.Errorf("wiederhergestelltes SQL weicht ab")
	}
}

// TestBackupRestore_AltformatBleibtLesbar ist der Grund, warum der Format-Wechsel
// gefahrlos ist: Ein vor der Umstellung geschriebenes SHA-256-Backup öffnet DecryptBackup
// weiterhin. Ohne diesen Test wäre die Rückwärtskompatibilität nur eine Behauptung.
func TestBackupRestore_AltformatBleibtLesbar(t *testing.T) {
	passphrase := "alt-passphrase-mind-32-zeichen-xxxxxxxxxx"
	originalSQL := []byte("-- altes pg_dump\nCREATE TABLE alt (x int);\n")

	altbackup := altformatVerschluesseln(t, passphrase, gzipBytes(t, originalSQL))
	if istScryptFormat(altbackup) {
		t.Fatal("Altformat darf das scrypt-Magic NICHT tragen")
	}
	decrypted, err := DecryptBackup(passphrase, altbackup)
	if err != nil {
		t.Fatalf("Altbackup nicht mehr lesbar — Datenverlust: %v", err)
	}
	if !bytes.Equal(gunzipBytes(t, decrypted), originalSQL) {
		t.Error("Altbackup falsch wiederhergestellt")
	}
}

func TestBackupRestore_WrongKeyFails(t *testing.T) {
	encrypted, err := verschluesseleBackup("richtige-passphrase-mind-32-zeichen-xx", gzipBytes(t, []byte("geheime daten")))
	if err != nil {
		t.Fatalf("verschluesseleBackup: %v", err)
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
	encrypted, err := verschluesseleBackup("passphrase-mind-32-zeichen-xxxxxxxxxxxx", gzipBytes(t, []byte("integritaet")))
	if err != nil {
		t.Fatalf("verschluesseleBackup: %v", err)
	}
	// Letztes Byte (im GCM-Tag-Bereich) kippen → Manipulation muss erkannt werden.
	encrypted[len(encrypted)-1] ^= 0xFF
	if _, err := DecryptBackup("passphrase-mind-32-zeichen-xxxxxxxxxxxx", encrypted); err == nil {
		t.Error("manipulierter Ciphertext soll an der GCM-Integritätsprüfung scheitern")
	}
}
