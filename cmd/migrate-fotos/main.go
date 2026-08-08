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
	defer func() {
		if cerr := root.Close(); cerr != nil {
			slog.Error("Fehler beim Schließen des Root-Verzeichnisses", "error", cerr)
		}
	}()

	dir, err := root.Open(".")
	if err != nil {
		slog.Error("Fehler beim Lesen des Foto-Verzeichnisses", "error", err)
		os.Exit(1)
	}
	entries, err := dir.ReadDir(-1)
	if cerr := dir.Close(); cerr != nil {
		slog.Error("Fehler beim Schließen des Foto-Verzeichnisses", "error", cerr)
	}
	if err != nil {
		slog.Error("Fehler beim Lesen des Foto-Verzeichnisses", "error", err)
		os.Exit(1)
	}

	processed, migrated := migriereAlleFotos(pool, root, entries)

	fmt.Printf("Migration abgeschlossen. %d Fotos gefunden, %d erfolgreich migriert und verschlüsselt.\n", processed, migrated)
	fmt.Println("Du kannst das Verzeichnis 'uploads/fotos' jetzt sicher löschen.")
}

// migriereAlleFotos verschlüsselt alle .jpg-Dateien im Verzeichnis und liefert die Zahl
// gefundener und erfolgreich migrierter Fotos. Ausgelagert aus main, damit main flach bleibt.
func migriereAlleFotos(pool *pgxpool.Pool, root *os.Root, entries []os.DirEntry) (processed, migrated int) {
	batch := &pgx.Batch{}

	query := `
		INSERT INTO schueler_fotos (schueler_id, foto_encrypted)
		SELECT id, $2
		FROM schueler WHERE barcode_id = $1
		ON CONFLICT (schueler_id) DO UPDATE SET
			foto_encrypted = EXCLUDED.foto_encrypted,
			aktualisiert_am = CURRENT_TIMESTAMP
		RETURNING schueler_id
	`

	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(strings.ToLower(name), ".jpg") {
			continue
		}
		processed++

		barcodeID := strings.TrimSuffix(name, filepath.Ext(name))

		file, err := root.Open(name)
		if err != nil {
			slog.Error("Konnte Bild nicht öffnen", "file", name, "error", err)
			continue
		}
		imgBytes, err := io.ReadAll(file)
		if cerr := file.Close(); cerr != nil {
			slog.Error("Fehler beim Schließen der Datei", "file", name, "error", cerr)
		}
		if err != nil {
			slog.Error("Konnte Bild nicht lesen", "file", name, "error", err)
			continue
		}

		encryptedData, err := crypto.Encrypt(imgBytes)
		if err != nil {
			slog.Error("Konnte Bild nicht verschlüsseln", "file", name, "error", err)
			continue
		}

		batch.Queue(query, barcodeID, encryptedData)
	}

	if batch.Len() > 0 {
		ctx := context.Background()
		br := pool.SendBatch(ctx, batch)

		for i := 0; i < batch.Len(); i++ {
			var studentID string
			err := br.QueryRow().Scan(&studentID)
			if err != nil {
				if err == pgx.ErrNoRows {
					// Dies passiert, wenn das SELECT id ... WHERE barcode_id = $1 kein Ergebnis geliefert hat.
					// Wir können nicht genau sagen, welcher Barcode das war, aber der Batch wird fortgesetzt.
					slog.Warn("Foto übersprungen: Kein Schüler gefunden oder DB-Fehler")
				} else {
					slog.Error("Fehler beim Einfügen in die Datenbank", "error", err)
				}
			} else {
				migrated++
			}
		}
		if cerr := br.Close(); cerr != nil {
			slog.Error("Fehler beim Schließen des Batches", "error", cerr)
		}
	}

	return processed, migrated
}
