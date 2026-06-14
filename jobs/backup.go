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

// BackupJob führt tägliche verschlüsselte PostgreSQL-Datenbank-Backups durch.
// Es ruft pg_dump auf (muss im PATH sein), komprimiert mit gzip und
// verschlüsselt mit AES-256-GCM unter Verwendung eines Schlüssels, der aus BACKUP_ENCRYPTION_KEY abgeleitet wird.
//
// Erforderliche Umgebungsvariablen:
//   - DATABASE_URL          – PostgreSQL DSN (im Produktivbetrieb bereits gesetzt)
//   - BACKUP_ENCRYPTION_KEY – 32+ Zeichen Passphrase zur AES-256 Schlüsselableitung
//   - BACKUP_DIR            – Zielverzeichnis (Standard: ./backups)
type BackupJob struct{}

// RunDatabaseBackup führt die komplette Backup-Pipeline aus: Dump → gzip → AES-256-GCM Verschlüsselung.
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
		// #nosec G706
		log.Printf("Backup: cannot create backup directory: %v", err)
		return
	}

	// Leitet einen stabilen 32-Byte AES-Schlüssel via SHA-256 aus der Passphrase ab.
	// In der Produktion durch eine saubere KDF (argon2id/scrypt) ersetzen, falls Schlüsselrotation nötig ist.
	keyBytes := sha256.Sum256([]byte(encKey))

	timestamp := time.Now().UTC().Format("2006-01-02T150405Z")
	outFilename := filepath.Join(backupDir, fmt.Sprintf("backup_%s.sql.gz.enc", timestamp))

	// #nosec G706

	log.Printf("Backup: starting daily PostgreSQL backup → %s", outFilename)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	// pg_dump schreibt SQL nach stdout; wir pipen es durch gzip → AES-GCM Verschlüsselung.
	// Parse den DSN, um die Verbindungsparameter für pg_dump zu extrahieren.
	pgDumpArgs := dsnToPgDumpArgs(dsn)
	pgDump := exec.CommandContext(ctx, "pg_dump", pgDumpArgs...) //nolint:gosec

	sqlReader, sqlWriter := io.Pipe()
	pgDump.Stdout = sqlWriter
	pgDump.Stderr = os.Stderr

	if err := pgDump.Start(); err != nil {
		// #nosec G706
		log.Printf("Backup: pg_dump start failed: %v", err)
		return
	}

	// Pipeline: Gzip-Komprimierung des SQL-Streams
	var compressedBuf strings.Builder
	pr, pw := io.Pipe()

	// Goroutine: Gzip-Komprimierung der pg_dump-Ausgabe
	go func() {
		gz := gzip.NewWriter(pw)
		if _, err := io.Copy(gz, sqlReader); err != nil {
			pw.CloseWithError(err)
			return
		}
		_ = gz.Close()
		_ = pw.Close()
	}()

	// Alle komprimierten Daten lesen
	compressedData, err := io.ReadAll(pr)
	_ = compressedBuf // silence unused var
	if err != nil {
		// #nosec G706
		log.Printf("Backup: compression failed: %v", err)
		return
	}

	if err := pgDump.Wait(); err != nil {
		// #nosec G706
		log.Printf("Backup: pg_dump finished with error: %v", err)
		return
	}

	// AES-256-GCM Verschlüsselung des komprimierten Dumps
	encrypted, err := encryptAESGCM(keyBytes[:], compressedData)
	if err != nil {
		// #nosec G706
		log.Printf("Backup: encryption failed: %v", err)
		return
	}

	// Verschlüsseltes Backup mit restriktiven Berechtigungen auf die Festplatte schreiben (Eigentümer nur lesen)
	// #nosec G304 - outFilename is safely constructed using a timestamp
	if err := os.WriteFile(outFilename, encrypted, 0600); err != nil {
		// #nosec G706
		log.Printf("Backup: writing backup file failed: %v", err)
		return
	}

	sizeMB := float64(len(encrypted)) / 1024 / 1024
	// #nosec G706
	log.Printf("Backup: completed successfully → %s (%.2f MB)", outFilename, sizeMB)

	// Rotation: Nur die letzten 14 täglichen Backups behalten, um Speicherplatzmangel zu vermeiden
	rotateBackups(backupDir, 14)
}

// encryptAESGCM verschlüsselt Klartext mit AES-256-GCM.
// Ausgabeformat: [12-Byte Nonce][Ciphertext+Tag].
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

// DecryptBackup ist eine Hilfsfunktion für Disaster-Recovery-Wiederherstellungen.
// Verwendung: .enc Datei lesen, DecryptBackup(key, data) aufrufen, gunzip, Wiederherstellung über psql.
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

// rotateBackups löscht die ältesten Backup-Dateien, wenn es mehr als maxKeep gibt.
func rotateBackups(dir string, maxKeep int) {
	entries, err := filepath.Glob(filepath.Join(dir, "backup_*.sql.gz.enc"))
	if err != nil || len(entries) <= maxKeep {
		return
	}
	// Dateien sind nach Zeitstempel benannt; lexikographische Sortierung = chronologische Sortierung
	// Einträge aus Glob sind bereits alphabetisch sortiert
	toDelete := entries[:len(entries)-maxKeep]
	for _, f := range toDelete {
		// #nosec G304 - f is derived from filepath.Glob
		if err := os.Remove(f); err != nil {
			// #nosec G706
			log.Printf("Backup rotation: failed to delete %s: %v", f, err)
		} else {
			// #nosec G706
			log.Printf("Backup rotation: deleted old backup %s", f)
		}
	}
}

// dsnToPgDumpArgs konvertiert einen PostgreSQL-DSN/Verbindungsstring in CLI-Argumente für pg_dump.
// Unterstützt sowohl das postgres:// URL-Format als auch das key=value-Format.
func dsnToPgDumpArgs(dsn string) []string {
	// DSN direkt über PGPASSWORD ENV übergeben, separat gesetzt.
	// pg_dump akzeptiert --dbname mit einer vollständigen Verbindungs-URI.
	return []string{
		"--dbname=" + dsn,
		"--no-password",
		"--format=plain",
		"--encoding=UTF8",
		"--verbose",
	}
}

// BackupKeyFingerprint gibt einen kurzen Hex-Fingerabdruck des Verschlüsselungsschlüssels für Protokollierungs-/Audit-Zwecke zurück.
// Protokolliert NIEMALS den tatsächlichen Schlüssel – nur einen SHA-256-Fingerabdruck davon.
func BackupKeyFingerprint(encKey string) string {
	h := sha256.Sum256([]byte(encKey))
	return hex.EncodeToString(h[:4]) // 8 hex chars = 32 bits, sufficient for audit identity
}
