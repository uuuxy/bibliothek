package jobs

import (
	"compress/gzip"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// BackupJob performs daily encrypted PostgreSQL database backups.
// It calls pg_dump (must be on PATH), compresses with gzip, and
// encrypts with AES-256-GCM using a key derived from BACKUP_ENCRYPTION_KEY.
//
// Required environment variables:
//   - DATABASE_URL          – PostgreSQL DSN (already set in production)
//   - BACKUP_ENCRYPTION_KEY – 32+ character passphrase for AES-256 key derivation
//   - BACKUP_DIR            – destination directory (default: ./backups)
type BackupJob struct{}

// RunDatabaseBackup executes the full backup pipeline: dump → gzip → AES-256-GCM encrypt.
func (b *BackupJob) RunDatabaseBackup() {
	encKey := os.Getenv("BACKUP_ENCRYPTION_KEY")
	if encKey == "" {
		log.Println("Backup: BACKUP_ENCRYPTION_KEY not set – skipping encrypted backup")
		return
	}

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Println("Backup: DATABASE_URL not set – skipping backup")
		return
	}

	backupDir := os.Getenv("BACKUP_DIR")
	if backupDir == "" {
		backupDir = "./backups"
	}
	if err := os.MkdirAll(backupDir, 0750); err != nil {
		log.Printf("Backup: cannot create backup directory: %v", err)
		return
	}

	// Derive a stable 32-byte AES key via SHA-256 of the passphrase.
	// In production, replace with a proper KDF (argon2id/scrypt) if key rotation is needed.
	keyBytes := sha256.Sum256([]byte(encKey))

	timestamp := time.Now().UTC().Format("2006-01-02T150405Z")
	outFilename := filepath.Join(backupDir, fmt.Sprintf("backup_%s.sql.gz.enc", timestamp))

	log.Printf("Backup: starting daily PostgreSQL backup → %s", outFilename)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	// pg_dump writes SQL to stdout; we pipe it through gzip → AES-GCM encryption.
	// Parse the DSN to extract connection parameters for pg_dump.
	pgDumpArgs := dsnToPgDumpArgs(dsn)
	pgDump := exec.CommandContext(ctx, "pg_dump", pgDumpArgs...) //nolint:gosec

	sqlReader, sqlWriter := io.Pipe()
	pgDump.Stdout = sqlWriter
	pgDump.Stderr = os.Stderr

	if err := pgDump.Start(); err != nil {
		log.Printf("Backup: pg_dump start failed: %v", err)
		return
	}

	// Pipeline: gzip compress the SQL stream
	var compressedBuf strings.Builder
	pr, pw := io.Pipe()

	// Goroutine: gzip-compress the pg_dump output
	go func() {
		gz := gzip.NewWriter(pw)
		if _, err := io.Copy(gz, sqlReader); err != nil {
			pw.CloseWithError(err)
			return
		}
		gz.Close()
		pw.Close()
	}()

	// Read all compressed data
	compressedData, err := io.ReadAll(pr)
	_ = compressedBuf // silence unused var
	if err != nil {
		log.Printf("Backup: compression failed: %v", err)
		return
	}

	if err := pgDump.Wait(); err != nil {
		log.Printf("Backup: pg_dump finished with error: %v", err)
		return
	}

	// AES-256-GCM encrypt the compressed dump
	encrypted, err := encryptAESGCM(keyBytes[:], compressedData)
	if err != nil {
		log.Printf("Backup: encryption failed: %v", err)
		return
	}

	// Write encrypted backup to disk with restrictive permissions (owner read-only)
	// #nosec G304 - outFilename is safely constructed using a timestamp
	if err := os.WriteFile(outFilename, encrypted, 0600); err != nil {
		log.Printf("Backup: writing backup file failed: %v", err)
		return
	}

	sizeMB := float64(len(encrypted)) / 1024 / 1024
	log.Printf("Backup: completed successfully → %s (%.2f MB)", outFilename, sizeMB)

	// Rotate: keep only the last 14 daily backups to avoid disk exhaustion
	rotateBackups(backupDir, 14)
}

// encryptAESGCM encrypts plaintext using AES-256-GCM.
// Output format: [12-byte nonce][ciphertext+tag].
func encryptAESGCM(key, plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("AES cipher init: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("GCM init: %w", err)
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("nonce generation: %w", err)
	}
	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// DecryptBackup is a utility function for disaster-recovery restore operations.
// Usage: read the .enc file, call DecryptBackup(key, data), gunzip, restore via psql.
func DecryptBackup(encKey string, ciphertext []byte) ([]byte, error) {
	keyBytes := sha256.Sum256([]byte(encKey))
	block, err := aes.NewCipher(keyBytes[:])
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}
	nonce, ct := ciphertext[:nonceSize], ciphertext[nonceSize:]
	return gcm.Open(nil, nonce, ct, nil)
}

// rotateBackups deletes the oldest backup files if there are more than maxKeep.
func rotateBackups(dir string, maxKeep int) {
	entries, err := filepath.Glob(filepath.Join(dir, "backup_*.sql.gz.enc"))
	if err != nil || len(entries) <= maxKeep {
		return
	}
	// Files are named with timestamps; lexicographic sort = chronological sort
	// Entries from Glob are already sorted alphabetically
	toDelete := entries[:len(entries)-maxKeep]
	for _, f := range toDelete {
		// #nosec G304 - f is derived from filepath.Glob
		if err := os.Remove(f); err != nil {
			log.Printf("Backup rotation: failed to delete %s: %v", f, err)
		} else {
			log.Printf("Backup rotation: deleted old backup %s", f)
		}
	}
}

// dsnToPgDumpArgs converts a PostgreSQL DSN/connection string to pg_dump CLI arguments.
// Supports both postgres:// URL format and key=value format.
func dsnToPgDumpArgs(dsn string) []string {
	// Pass the DSN directly via PGPASSWORD env is set separately.
	// pg_dump accepts --dbname with a full connection URI.
	return []string{
		"--dbname=" + dsn,
		"--no-password",
		"--format=plain",
		"--encoding=UTF8",
		"--verbose",
	}
}

// BackupKeyFingerprint returns a short hex fingerprint of the encryption key for logging/audit.
// NEVER logs the actual key – only a SHA-256 fingerprint of it.
func BackupKeyFingerprint(encKey string) string {
	h := sha256.Sum256([]byte(encKey))
	return hex.EncodeToString(h[:4]) // 8 hex chars = 32 bits, sufficient for audit identity
}
