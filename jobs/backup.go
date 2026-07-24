package jobs

import (
	"bytes"
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

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"github.com/jackc/pgx/v5/pgconn"
)

// escapePgPass escapes backslashes and colons as required by PostgreSQL .pgpass format.
func escapePgPass(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	return strings.ReplaceAll(s, ":", "\\:")
}

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
	encKey, dsn, backupDir, ok := resolveBackupEnv()
	if !ok {
		return
	}

	// Leitet einen stabilen 32-Byte AES-Schlüssel via SHA-256 aus der Passphrase ab.
	// In der Produktion durch eine saubere KDF (argon2id/scrypt) ersetzen, falls Schlüsselrotation nötig ist.
	keyBytes := sha256.Sum256([]byte(encKey))

	timestamp := time.Now().UTC().Format("2006-01-02T150405Z")
	outFilename := filepath.Join(backupDir, fmt.Sprintf("backup_%s.sql.gz.enc", timestamp))

	log.Printf("Backup: starting daily PostgreSQL backup → %s", outFilename)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	compressedData, ok := dumpAndCompress(ctx, dsn)
	if !ok {
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

	// S3 Offsite Upload
	uploadBackupToS3(ctx, outFilename, encrypted)

	// Rotation: Nur die letzten 14 täglichen Backups behalten, um Speicherplatzmangel zu vermeiden
	rotateBackups(backupDir, 14)
}

// resolveBackupEnv liest die Backup-relevanten Umgebungsvariablen und legt bei
// Bedarf das Zielverzeichnis an. ok=false signalisiert einen (protokollierten)
// Abbruchgrund (fehlender Key/DSN oder nicht anlegbares Verzeichnis).
func resolveBackupEnv() (encKey, dsn, backupDir string, ok bool) {
	encKey = os.Getenv("BACKUP_ENCRYPTION_KEY")
	if encKey == "" {
		log.Println("Backup: BACKUP_ENCRYPTION_KEY not set – skipping encrypted backup")
		return "", "", "", false
	}

	dsn = os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Println("Backup: DATABASE_URL not set – skipping backup")
		return "", "", "", false
	}

	backupDir = os.Getenv("BACKUP_DIR")
	if backupDir == "" {
		backupDir = "./backups"
	}
	if err := os.MkdirAll(backupDir, 0750); err != nil {
		// #nosec G706
		log.Printf("Backup: cannot create backup directory: %v", err)
		return "", "", "", false
	}
	return encKey, dsn, backupDir, true
}

func createPgPassFile(config *pgconn.Config) (string, error) {
	passFile, err := os.CreateTemp("", "pgpass-*")
	if err != nil {
		return "", fmt.Errorf("konnte pgpass-Datei nicht erstellen: %w", err)
	}
	defer func() { _ = passFile.Close() }() //nolint:errcheck

	port := fmt.Sprintf("%d", config.Port)
	if port == "0" {
		port = "5432"
	}

	pgPassLine := fmt.Sprintf("%s:%s:%s:%s:%s\n",
		escapePgPass(config.Host),
		escapePgPass(port),
		escapePgPass(config.Database),
		escapePgPass(config.User),
		escapePgPass(config.Password),
	)
	if _, err := passFile.WriteString(pgPassLine); err != nil {
		return "", fmt.Errorf("konnte in pgpass-Datei nicht schreiben: %w", err)
	}

	return passFile.Name(), nil
}

// dumpAndCompress ruft pg_dump auf (Verbindungsdaten via temporärer .pgpass-Datei)
// und liefert den gzip-komprimierten SQL-Dump. Jeder Fehler wird mit seiner Ursache
// protokolliert; ok=false bedeutet Abbruch.
func dumpAndCompress(ctx context.Context, dsn string) (compressedData []byte, ok bool) {
	// Parse den DSN, um die Verbindungsparameter für pg_dump zu extrahieren.
	config, err := pgconn.ParseConfig(dsn)
	if err != nil {
		log.Printf("Backup: failed to parse DSN: %v", err)
		return nil, false
	}

	passFileName, err := createPgPassFile(config)
	if err != nil {
		log.Printf("Backup: %v", err)
		return nil, false
	}
	defer func() { _ = os.Remove(passFileName) }() //nolint:errcheck

	port := fmt.Sprintf("%d", config.Port)
	if port == "0" {
		port = "5432"
	}

	// #nosec G204 - arguments are derived from securely parsed DSN configuration
	pgDump := exec.CommandContext(ctx, "pg_dump",
		"--host="+config.Host,
		"--port="+port,
		"--username="+config.User,
		"--dbname="+config.Database,
		"--no-password",
		"--format=plain",
		"--encoding=UTF8",
		"--verbose",
	)
	pgDump.Env = append(os.Environ(), "PGPASSFILE="+passFileName)
	sqlReader, sqlWriter := io.Pipe()
	pgDump.Stdout = sqlWriter
	pgDump.Stderr = os.Stderr

	if err := pgDump.Start(); err != nil {
		// #nosec G706
		log.Printf("Backup: pg_dump start failed: %v", err)
		return nil, false
	}

	// Start Wait in a goroutine to close the writer when process exits
	waitErrCh := make(chan error, 1)
	go func() {
		err := pgDump.Wait()
		_ = sqlWriter.Close() // Ensure io.Copy gets EOF
		waitErrCh <- err
	}()

	// Pipeline: Gzip-Komprimierung des SQL-Streams
	pr, pw := io.Pipe()

	// Goroutine: Gzip-Komprimierung der pg_dump-Ausgabe
	go func() {
		gz := gzip.NewWriter(pw)
		if _, err := io.Copy(gz, sqlReader); err != nil {
			pw.CloseWithError(err)
			return
		}
		// gz.Close flushes the gzip footer; propagate a failure to the reader so it
		// does not consume a truncated, invalid archive.
		if err := gz.Close(); err != nil {
			pw.CloseWithError(err)
			return
		}
		pw.CloseWithError(nil) // signals clean EOF to the reading side
	}()

	// Alle komprimierten Daten lesen
	data, err := io.ReadAll(pr)
	if err != nil {
		// #nosec G706
		log.Printf("Backup: compression failed: %v", err)
		return nil, false
	}

	if err := <-waitErrCh; err != nil {
		// #nosec G706
		log.Printf("Backup: pg_dump finished with error: %v", err)
		return nil, false
	}

	return data, true
}

// uploadBackupToS3 lädt das verschlüsselte Backup offsite zu S3 hoch, sofern die
// S3-Zugangsdaten vollständig konfiguriert sind. Fehler werden protokolliert, aber
// nicht weitergereicht – das lokale Backup gilt bereits als erfolgreich.
func uploadBackupToS3(ctx context.Context, outFilename string, encrypted []byte) {
	s3Endpoint := os.Getenv("S3_ENDPOINT")
	s3AccessKey := os.Getenv("S3_ACCESS_KEY")
	s3SecretKey := os.Getenv("S3_SECRET_KEY")
	s3Bucket := os.Getenv("S3_BUCKET")
	s3UseSSL := os.Getenv("S3_USE_SSL") != "false" // Default to true

	if s3Endpoint == "" || s3AccessKey == "" || s3SecretKey == "" || s3Bucket == "" {
		log.Println("Backup: S3 credentials not fully configured – skipping offsite upload")
		return
	}

	minioClient, err := minio.New(s3Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(s3AccessKey, s3SecretKey, ""),
		Secure: s3UseSSL,
	})
	if err != nil {
		log.Printf("Backup: Failed to initialize S3 client: %v", err)
		return
	}

	objectName := filepath.Base(outFilename)
	reader := bytes.NewReader(encrypted)

	// Optional: Make bucket if not exists
	exists, errBucketExists := minioClient.BucketExists(ctx, s3Bucket)
	if errBucketExists == nil && !exists {
		if err := minioClient.MakeBucket(ctx, s3Bucket, minio.MakeBucketOptions{}); err != nil {
			log.Printf("Backup: S3-Bucket %q konnte nicht angelegt werden: %v", s3Bucket, err)
		}
	}

	if _, err := minioClient.PutObject(ctx, s3Bucket, objectName, reader, int64(len(encrypted)), minio.PutObjectOptions{
		ContentType: "application/octet-stream",
	}); err != nil {
		log.Printf("Backup: S3 upload failed for %s: %v", objectName, err)
		return
	}
	log.Printf("Backup: S3 upload successful → s3://%s/%s", s3Bucket, objectName)
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

// BackupKeyFingerprint gibt einen kurzen Hex-Fingerabdruck des Verschlüsselungsschlüssels für Protokollierungs-/Audit-Zwecke zurück.
// Protokolliert NIEMALS den tatsächlichen Schlüssel – nur einen SHA-256-Fingerabdruck davon.
func BackupKeyFingerprint(encKey string) string {
	h := sha256.Sum256([]byte(encKey))
	return hex.EncodeToString(h[:4]) // 8 hex chars = 32 bits, sufficient for audit identity
}
