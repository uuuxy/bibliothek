package main

import (
	"bytes"
	"compress/gzip"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createOldFormatBackup erzeugt ein Backup im alten SHA-256 Format.
func createOldFormatBackup(t *testing.T, passphrase string, cleartext []byte) []byte {
	t.Helper()
	keyBytes := sha256.Sum256([]byte(passphrase))
	block, err := aes.NewCipher(keyBytes[:])
	require.NoError(t, err)

	gcm, err := cipher.NewGCM(block)
	require.NoError(t, err)

	nonce := make([]byte, gcm.NonceSize())
	_, err = rand.Read(nonce)
	require.NoError(t, err)

	return gcm.Seal(nonce, nonce, cleartext, nil)
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
	encData := createOldFormatBackup(t, "secret", []byte("not a gzip file"))
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
	encData := createOldFormatBackup(t, "secret", gzipped)

	err := os.WriteFile(inFile, encData, 0644)
	require.NoError(t, err)

	setupArgs(t, inFile)
	t.Setenv("BACKUP_ENCRYPTION_KEY", "secret")

	// Redirect stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	t.Cleanup(func() {
		os.Stdout = oldStdout
	})

	err = run()
	require.NoError(t, err)

	w.Close()

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
	encData := createOldFormatBackup(t, "secret", gzipped)

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

func TestRun_FileCreateFails(t *testing.T) {
	dir := t.TempDir()
	inFile := filepath.Join(dir, "backup.enc")

	// Use a directory as the output file to trigger an open error
	outFile := filepath.Join(dir, "out_dir")
	err := os.Mkdir(outFile, 0755)
	require.NoError(t, err)

	sqlData := []byte("SELECT * FROM books;")
	gzipped := gzipData(t, sqlData)
	encData := createOldFormatBackup(t, "secret", gzipped)

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
