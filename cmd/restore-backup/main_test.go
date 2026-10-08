package main

import (
	"bytes"
	"compress/gzip"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"bibliothek/internal/backupkrypto"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createBackup erzeugt ein gültiges Backup im scrypt-Format über die echte
// Verschlüsselungsfunktion — so testet der Restore genau das Format, das er im Betrieb
// vorfindet, statt es im Test nachzubauen.
func createBackup(t *testing.T, passphrase string, cleartext []byte) []byte {
	t.Helper()
	enc, err := backupkrypto.VerschluesseleBackup(passphrase, cleartext)
	require.NoError(t, err)
	return enc
}

func gzipData(t *testing.T, data []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := gzip.NewWriter(&buf)
	_, err := w.Write(data)
	require.NoError(t, err)
	require.NoError(t, w.Close())
	return buf.Bytes()
}

func setupArgs(t *testing.T, args ...string) {
	t.Helper()
	oldArgs := os.Args
	os.Args = append([]string{"restore-backup"}, args...)
	t.Cleanup(func() {
		os.Args = oldArgs
	})
}

func TestRun_InvalidArgs(t *testing.T) {
	setupArgs(t) // 0 args
	err := run()
	assert.ErrorContains(t, err, "usage:")

	setupArgs(t, "in.enc", "out.sql", "extra") // 3 args
	err = run()
	assert.ErrorContains(t, err, "usage:")
}

func TestRun_NoKey(t *testing.T) {
	setupArgs(t, "in.enc")
	t.Setenv("BACKUP_ENCRYPTION_KEY", "")
	err := run()
	assert.ErrorContains(t, err, "BACKUP_ENCRYPTION_KEY nicht gesetzt")
}

func TestRun_FileNotFound(t *testing.T) {
	setupArgs(t, "does_not_exist.enc")
	t.Setenv("BACKUP_ENCRYPTION_KEY", "secret")
	err := run()
	assert.ErrorContains(t, err, "konnte nicht gelesen werden")
}

func TestRun_DecryptFails(t *testing.T) {
	dir := t.TempDir()
	inFile := filepath.Join(dir, "bad.enc")
	err := os.WriteFile(inFile, []byte("invalid data"), 0644)
	require.NoError(t, err)

	setupArgs(t, inFile)
	t.Setenv("BACKUP_ENCRYPTION_KEY", "secret")
	err = run()
	assert.ErrorContains(t, err, "entschlüsselung fehlgeschlagen")
}

func TestRun_GzipFails(t *testing.T) {
	dir := t.TempDir()
	inFile := filepath.Join(dir, "bad_gzip.enc")

	// Valid encryption, but invalid gzip content
	encData := createBackup(t, "secret", []byte("not a gzip file"))
	err := os.WriteFile(inFile, encData, 0644)
	require.NoError(t, err)

	setupArgs(t, inFile)
	t.Setenv("BACKUP_ENCRYPTION_KEY", "secret")
	err = run()
	assert.ErrorContains(t, err, "gzip-header ungültig")
}

func TestRun_SuccessStdout(t *testing.T) {
	dir := t.TempDir()
	inFile := filepath.Join(dir, "backup.enc")

	sqlData := []byte("SELECT * FROM books;")
	gzipped := gzipData(t, sqlData)
	encData := createBackup(t, "secret", gzipped)

	err := os.WriteFile(inFile, encData, 0644)
	require.NoError(t, err)

	setupArgs(t, inFile)
	t.Setenv("BACKUP_ENCRYPTION_KEY", "secret")

	// Redirect stdout
	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w
	t.Cleanup(func() {
		os.Stdout = oldStdout
	})

	err = run()
	require.NoError(t, err)

	require.NoError(t, w.Close())

	outData, err := io.ReadAll(r)
	require.NoError(t, err)
	assert.Equal(t, sqlData, outData)
}

func TestRun_SuccessFile(t *testing.T) {
	dir := t.TempDir()
	inFile := filepath.Join(dir, "backup.enc")
	outFile := filepath.Join(dir, "out.sql")

	sqlData := []byte("SELECT * FROM books;")
	gzipped := gzipData(t, sqlData)
	encData := createBackup(t, "secret", gzipped)

	err := os.WriteFile(inFile, encData, 0644)
	require.NoError(t, err)

	setupArgs(t, inFile, outFile)
	t.Setenv("BACKUP_ENCRYPTION_KEY", "secret")

	err = run()
	require.NoError(t, err)

	outData, err := os.ReadFile(outFile)
	require.NoError(t, err)
	assert.Equal(t, sqlData, outData)
}

// Nach einem Wechsel des BACKUP_ENCRYPTION_KEY öffnet jede Sicherung nur der Schlüssel, der bei
// ihrem Entstehen galt (docs/SECURITY.md, „BACKUP_ENCRYPTION_KEY wechseln"). Mit dem anderen
// nennt die Meldung den falschen Schlüssel, und es entsteht keine Ausgabedatei.
func TestRun_NachSchluesselwechsel(t *testing.T) {
	const (
		alt = "alter-schluessel-mit-mehr-als-32-zeichen-0001"
		neu = "neuer-schluessel-mit-mehr-als-32-zeichen-0002"
	)
	dir := t.TempDir()
	standVorher, standNachher := []byte("-- Stand vor dem Wechsel\n"), []byte("-- Stand nach dem Wechsel\n")
	vorher := filepath.Join(dir, "vorher.sql.gz.enc")
	nachher := filepath.Join(dir, "nachher.sql.gz.enc")
	require.NoError(t, os.WriteFile(vorher, createBackup(t, alt, gzipData(t, standVorher)), 0o600))
	require.NoError(t, os.WriteFile(nachher, createBackup(t, neu, gzipData(t, standNachher)), 0o600))

	faelle := []struct {
		name       string
		schluessel string
		datei      string
		inhalt     []byte // nil: Das Werkzeug lehnt ab.
	}{
		{"laufender Schlüssel, Sicherung von vor dem Wechsel", neu, vorher, nil},
		{"alter Schlüssel, Sicherung von vor dem Wechsel", alt, vorher, standVorher},
		{"laufender Schlüssel, Sicherung nach dem Wechsel", neu, nachher, standNachher},
		{"alter Schlüssel, Sicherung nach dem Wechsel", alt, nachher, nil},
	}
	for _, fall := range faelle {
		t.Run(fall.name, func(t *testing.T) {
			ausgabe := filepath.Join(t.TempDir(), "wiederherstellung.sql")
			setupArgs(t, fall.datei, ausgabe)
			t.Setenv("BACKUP_ENCRYPTION_KEY", fall.schluessel)

			err := run()
			if fall.inhalt == nil {
				assert.ErrorContains(t, err, "falscher Schlüssel")
				assert.NoFileExists(t, ausgabe)
				return
			}
			require.NoError(t, err)
			inhalt, err := os.ReadFile(ausgabe)
			require.NoError(t, err)
			assert.Equal(t, fall.inhalt, inhalt)
		})
	}
}

func TestRun_FileCreateFails(t *testing.T) {
	dir := t.TempDir()
	inFile := filepath.Join(dir, "backup.enc")

	// Use a directory as the output file to trigger an open error
	outFile := filepath.Join(dir, "out_dir")
	err := os.Mkdir(outFile, 0755)
	require.NoError(t, err)

	sqlData := []byte("SELECT * FROM books;")
	gzipped := gzipData(t, sqlData)
	encData := createBackup(t, "secret", gzipped)

	err = os.WriteFile(inFile, encData, 0644)
	require.NoError(t, err)

	setupArgs(t, inFile, outFile)
	t.Setenv("BACKUP_ENCRYPTION_KEY", "secret")

	err = run()
	assert.ErrorContains(t, err, "ausgabedatei")
}

func TestMain_Failure(t *testing.T) {
	// Execute the test binary with a special environment variable
	if os.Getenv("BE_CRASHER") == "1" {
		setupArgs(t)
		main()
		return
	}

	cmd := os.Args[0]
	// run the test binary but with a specific test name and env var set
	execCmd := exec.Command(cmd, "-test.run=TestMain_Failure")
	execCmd.Env = append(os.Environ(), "BE_CRASHER=1")
	err := execCmd.Run()
	// main() should exit with 1, which means execCmd.Run() returns an error
	if e, ok := err.(*exec.ExitError); ok && !e.Success() {
		return // Success
	}
	t.Fatalf("process ran with err %v, want exit status 1", err)
}
