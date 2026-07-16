package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"bibliothek/internal/crypto"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	// Setup strukturiertes Logging
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		slog.Error("DATABASE_URL environment variable is not set")
		os.Exit(1)
	}

	key := os.Getenv("APP_ENCRYPTION_KEY")
	if key == "" {
		slog.Error("APP_ENCRYPTION_KEY environment variable is not set")
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		slog.Error("Failed to connect to DB", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	uploadDir := filepath.Join("uploads", "fotos")
	root, err := os.OpenRoot(uploadDir)
	if err != nil {
		if os.IsNotExist(err) {
			slog.Info("Kein uploads/fotos Verzeichnis gefunden. Nichts zu migrieren.")
			return
		}
		slog.Error("Fehler beim Öffnen des Foto-Verzeichnisses", "error", err)
		os.Exit(1)
	}
	defer root.Close()

	dir, err := root.Open(".")
	if err != nil {
		slog.Error("Fehler beim Lesen des Foto-Verzeichnisses", "error", err)
		os.Exit(1)
	}
	entries, err := dir.ReadDir(-1)
	dir.Close()
	if err != nil {
		slog.Error("Fehler beim Lesen des Foto-Verzeichnisses", "error", err)
		os.Exit(1)
	}

	var processed, migrated int
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ".jpg") {
			continue
		}
		processed++
		if migriereFoto(pool, root, entry.Name()) {
			migrated++
		}
	}

	fmt.Printf("Migration abgeschlossen. %d Fotos gefunden, %d erfolgreich migriert und verschlüsselt.\n", processed, migrated)
	fmt.Println("Du kannst das Verzeichnis 'uploads/fotos' jetzt sicher löschen.")
}

// migriereFoto liest, verschlüsselt und speichert das Foto einer Datei (Dateiname =
// "<barcode>.jpg") in schueler_fotos. Liefert true bei Erfolg; Fehler und fehlende
// Schüler werden protokolliert und mit false quittiert.
func migriereFoto(pool *pgxpool.Pool, root *os.Root, name string) bool {
	barcodeID := strings.TrimSuffix(name, filepath.Ext(name))

	file, err := root.Open(name)
	if err != nil {
		slog.Error("Konnte Bild nicht öffnen", "file", name, "error", err)
		return false
	}
	defer file.Close()

	imgBytes, err := io.ReadAll(file)
	if err != nil {
		slog.Error("Konnte Bild nicht lesen", "file", name, "error", err)
		return false
	}

	encryptedData, err := crypto.Encrypt(imgBytes)
	if err != nil {
		slog.Error("Konnte Bild nicht verschlüsseln", "file", name, "error", err)
		return false
	}

	var studentID string
	err = pool.QueryRow(context.Background(), "SELECT id FROM schueler WHERE barcode_id = $1", barcodeID).Scan(&studentID)
	if err != nil {
		if errorsIs(err, pgx.ErrNoRows) {
			slog.Warn("Kein Schüler für Barcode gefunden (übersprungen)", "barcode", barcodeID)
		} else {
			slog.Error("DB Fehler beim Suchen des Schülers", "barcode", barcodeID, "error", err)
		}
		return false
	}

	query := `
		INSERT INTO schueler_fotos (schueler_id, foto_encrypted)
		VALUES ($1, $2)
		ON CONFLICT (schueler_id) DO UPDATE SET
			foto_encrypted = EXCLUDED.foto_encrypted,
			aktualisiert_am = CURRENT_TIMESTAMP
	`
	_, err = pool.Exec(context.Background(), query, studentID, encryptedData)
	if err != nil {
		slog.Error("Fehler beim Einfügen in die Datenbank", "student_id", studentID, "error", err)
		return false
	}

	slog.Info("Foto erfolgreich migriert", "barcode", barcodeID)
	return true
}

func errorsIs(err error, target error) bool {
	return err.Error() == target.Error()
}
