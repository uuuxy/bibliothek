package jobs

import (
	"bytes"
	"context"
	"log"
	"os"
	"slices"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// Die Kopie außer Haus: Jede Nachtsicherung geht verschlüsselt in einen S3-Speicher, und dort gilt
// dieselbe Aufbewahrung wie im Backup-Verzeichnis (zuLoeschen). Ohne sie bliebe am Speicher jede
// Sicherung mit allen Personen ihres Stands liegen, und die Auskunft (internal/auskunft,
// dsgvoSicherungen) nennte eine Frist, die dort nicht gilt.

// s3Ziel ist der Speicher außer Haus mit seinem Bucket.
type s3Ziel struct {
	client *minio.Client
	bucket string
}

// leseS3Ziel liest den Speicher außer Haus aus der Umgebung. ok=false heißt: nicht vollständig
// eingestellt, oder der Client entsteht nicht; der Grund steht im Log.
func leseS3Ziel() (s3Ziel, bool) {
	endpunkt := os.Getenv("S3_ENDPOINT")
	zugang := os.Getenv("S3_ACCESS_KEY")
	geheimnis := os.Getenv("S3_SECRET_KEY")
	bucket := os.Getenv("S3_BUCKET")
	mitSSL := os.Getenv("S3_USE_SSL") != "false" // Vorgabe: mit SSL

	if endpunkt == "" || zugang == "" || geheimnis == "" || bucket == "" {
		log.Println("Backup: S3 credentials not fully configured – skipping offsite upload")
		return s3Ziel{}, false
	}
	client, err := minio.New(endpunkt, &minio.Options{
		Creds:  credentials.NewStaticV4(zugang, geheimnis, ""),
		Secure: mitSSL,
	})
	if err != nil {
		log.Printf("Backup: Failed to initialize S3 client: %v", err)
		return s3Ziel{}, false
	}
	return s3Ziel{client: client, bucket: bucket}, true
}

// schreibeBackupNachS3 legt die verschlüsselte Sicherung am Speicher ab; fehlt der Bucket, wird er
// angelegt. Ein Fehler steht im Log und geht nicht weiter: Die Sicherung im Haus gilt schon als
// gelungen.
func schreibeBackupNachS3(ctx context.Context, ziel s3Ziel, objektname string, verschluesselt []byte) bool {
	vorhanden, bucketFehler := ziel.client.BucketExists(ctx, ziel.bucket)
	if bucketFehler == nil && !vorhanden {
		if err := ziel.client.MakeBucket(ctx, ziel.bucket, minio.MakeBucketOptions{}); err != nil {
			log.Printf("Backup: S3-Bucket %q konnte nicht angelegt werden: %v", ziel.bucket, err)
		}
	}

	if _, err := ziel.client.PutObject(ctx, ziel.bucket, objektname, bytes.NewReader(verschluesselt), int64(len(verschluesselt)), minio.PutObjectOptions{
		ContentType: "application/octet-stream",
	}); err != nil {
		log.Printf("Backup: S3 upload failed for %s: %v", objektname, err)
		return false
	}
	log.Printf("Backup: S3 upload successful → s3://%s/%s", ziel.bucket, objektname)
	return true
}

// rotiereBackupsInS3 löscht am Speicher, was die Aufbewahrung nicht hält, nach derselben Regel wie
// rotiereBackups im Verzeichnis. Bricht die Liste ab, wird nichts gelöscht: Aus einer halben Liste
// löschte zuLoeschen Stände, die bleiben sollen.
func rotiereBackupsInS3(ctx context.Context, ziel s3Ziel, naechte, wochen int) {
	dateien, err := leseS3Backups(ctx, ziel)
	if err != nil {
		log.Printf("Backup rotation S3: Liste von %s nicht lesbar, es wird nichts gelöscht: %v", ziel.bucket, err)
		return
	}
	for _, d := range zuLoeschen(dateien, naechte, wochen) {
		// %q, weil die Namen setzt, wer Zugang zum Bucket hat; ein Zeilenumbruch darin fälschte das Log.
		if err := ziel.client.RemoveObject(ctx, ziel.bucket, d.Name, minio.RemoveObjectOptions{}); err != nil {
			log.Printf("Backup rotation S3: failed to delete %q: %v", d.Name, err)
		} else {
			log.Printf("Backup rotation S3: deleted old backup %q", d.Name)
		}
	}
}

// leseS3Backups liest die Sicherungen am Speicher wie leseBackups das Verzeichnis: nur
// backup_*.sql.gz.enc der obersten Ebene, nach Namen sortiert. Ohne Recursive kommt ein
// Unterordner als ein Eintrag, der auf „/" endet, und fällt am Suffix heraus.
func leseS3Backups(ctx context.Context, ziel s3Ziel) ([]BackupDatei, error) {
	var dateien []BackupDatei
	for objekt := range ziel.client.ListObjectsIter(ctx, ziel.bucket, minio.ListObjectsOptions{Prefix: backupPraefix}) {
		if objekt.Err != nil {
			return nil, objekt.Err
		}
		if strings.HasSuffix(objekt.Key, backupEndung) {
			dateien = append(dateien, BackupDatei{Name: objekt.Key, GeaendertAm: objekt.LastModified})
		}
	}
	slices.SortFunc(dateien, func(a, b BackupDatei) int { return strings.Compare(a.Name, b.Name) })
	return dateien, nil
}
