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

	"bibliothek/db"
	"bibliothek/internal/crypto"
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
//
// Die Schüler-IDs werden EINMAL für alle Barcodes geladen (ANY($1)) — das war das N+1 und
// ist seit cdb23e14 erledigt. Die INSERTs laufen bewusst weiter einzeln und NICHT als
// pgx.Batch. Das ist eine Entscheidung, kein Übersehen:
//
// Ein Batch-Umbau (PR #442, abgelehnt am 11.08.2026) kostet die Zuordnung. Am selben
// Datenbestand gegeneinander gefahren — ein Foto mit gültigem Barcode, eins ohne:
//
//	dieser Stand  INFO  "Foto erfolgreich migriert"            barcode=S-abg1-…
//	              WARN  "Kein Schüler für Barcode gefunden"    barcode=S-GIBTESNICHT-999
//	mit Batch     WARN  "Kein Schüler gefunden oder DB-Fehler" (ohne Barcode, ohne Erfolgszeile)
//
// Das „oder" ist der Punkt: Der Batch liefert pro Ergebnis nur ErrNoRows und kann nicht
// mehr sagen, WELCHE Datei es traf und OB es überhaupt ein Fehler war. Bei einer einmaligen
// Übernahme von Schülerfotos ist genau diese Liste das Ergebnis — wer sie nicht hat, weiß
// hinterher nicht, welche Kinder ohne Bild dastehen.
//
// Dazu käme, dass der Vorschlag die Suche wieder pro Zeile in die Anweisung zurückholt
// (INSERT … SELECT id FROM schueler WHERE barcode_id = $1) und damit die Vorablade-Abfrage
// oben entwertet, und dass ein Batch alle verschlüsselten Bilder gleichzeitig im Speicher
// hält statt eines nach dem anderen. Der Gewinn wäre eine Runde statt N — bei einem
// Werkzeug, das genau einmal läuft.
//
// Falls je zehntausende Fotos zu übernehmen sind: dann in Blöcken von ~50 batchen UND eine
// Liste der Barcodes in Batch-Reihenfolge mitführen, damit jedes Ergebnis wieder einer
// Datei zuzuordnen ist. Ohne diese Liste nicht.
func migriereAlleFotos(pool db.PgxPoolIface, root *os.Root, entries []os.DirEntry) (processed, migrated int) {
	// Barcodes aus Dateinamen extrahieren
	var barcodes []string
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ".jpg") {
			continue
		}
		barcodeID := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
		barcodes = append(barcodes, barcodeID)
	}

	if len(barcodes) == 0 {
		return 0, 0
	}

	// Alle Schueler-IDs auf einmal laden
	studentMap := make(map[string]string)
	query := "SELECT barcode_id, id FROM schueler WHERE barcode_id = ANY($1)"
	rows, err := pool.Query(context.Background(), query, barcodes)
	if err != nil {
		slog.Error("Fehler beim Laden der Schueler-IDs", "error", err)
		return 0, 0
	}
	defer rows.Close()

	for rows.Next() {
		var barcode, studentID string
		if err := rows.Scan(&barcode, &studentID); err != nil {
			slog.Error("Fehler beim Scannen der Schueler-ID", "error", err)
			continue
		}
		studentMap[barcode] = studentID
	}
	if rows.Err() != nil {
		slog.Error("Fehler nach dem Scannen der Schueler-IDs", "error", rows.Err())
		return 0, 0
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ".jpg") {
			continue
		}
		processed++

		barcodeID := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
		studentID, ok := studentMap[barcodeID]
		if !ok {
			slog.Warn("Kein Schüler für Barcode gefunden (übersprungen)", "barcode", barcodeID)
			continue
		}

		if migriereFoto(pool, root, entry.Name(), barcodeID, studentID) {
			migrated++
		}
	}
	return processed, migrated
}

// migriereFoto liest, verschlüsselt und speichert das Foto einer Datei (Dateiname =
// "<barcode>.jpg") in schueler_fotos. Liefert true bei Erfolg; Fehler und fehlende
// Schüler werden protokolliert und mit false quittiert.
func migriereFoto(pool db.PgxPoolIface, root *os.Root, name string, barcodeID string, studentID string) bool {
	file, err := root.Open(name)
	if err != nil {
		slog.Error("Konnte Bild nicht öffnen", "file", name, "error", err)
		return false
	}
	defer func() {
		if cerr := file.Close(); cerr != nil {
			slog.Error("Fehler beim Schließen der Datei", "file", name, "error", cerr)
		}
	}()

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
