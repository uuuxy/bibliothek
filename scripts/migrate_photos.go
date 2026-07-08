//go:build ignore

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"bibliothek/internal/crypto"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	log.Println("Starte Foto-Migration (Legacy -> Encrypted DB)...")

	// 1. Datenbank-URL aus Umgebungsvariable
	dbUrl := os.Getenv("DATABASE_URL")
	if dbUrl == "" {
		log.Fatal("FEHLER: DATABASE_URL Umgebungsvariable ist nicht gesetzt.")
	}

	// 2. Encryption Key überprüfen
	encKey := os.Getenv("APP_ENCRYPTION_KEY")
	if encKey == "" {
		log.Fatal("FEHLER: APP_ENCRYPTION_KEY Umgebungsvariable ist nicht gesetzt.")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbUrl)
	if err != nil {
		log.Fatalf("Fehler bei der Datenbankverbindung: %v", err)
	}
	defer pool.Close()

	fotosDir := filepath.Join("uploads", "fotos")
	files, err := os.ReadDir(fotosDir)
	if err != nil {
		if os.IsNotExist(err) {
			log.Println("Ordner 'uploads/fotos' existiert nicht. Keine Migration nötig.")
			return
		}
		log.Fatalf("Fehler beim Lesen des Ordners uploads/fotos: %v", err)
	}

	erfolg := 0
	fehler := 0
	uebersprungen := 0

	for _, file := range files {
		if file.IsDir() || filepath.Ext(strings.ToLower(file.Name())) != ".jpg" {
			continue
		}

		barcodeID := strings.TrimSuffix(file.Name(), filepath.Ext(file.Name())) // z.B. "S-10041"

		// 1. Hole UUID des Schülers anhand des Barcodes
		var schuelerID string
		err := pool.QueryRow(ctx, "SELECT id FROM schueler WHERE barcode_id = $1", barcodeID).Scan(&schuelerID)
		if err != nil {
			log.Printf("Überspringe %s: Schüler mit Barcode %s nicht in der DB gefunden.", file.Name(), barcodeID)
			uebersprungen++
			continue
		}

		// 2. Lese das JPEG-Bild
		filePath := filepath.Join(fotosDir, file.Name())
		imgBytes, err := os.ReadFile(filePath)
		if err != nil {
			log.Printf("Fehler beim Lesen der Datei %s: %v", file.Name(), err)
			fehler++
			continue
		}

		// 3. Verschlüssele das Bild
		encryptedData, err := crypto.Encrypt(imgBytes)
		if err != nil {
			log.Printf("Fehler beim Verschlüsseln von %s: %v", file.Name(), err)
			fehler++
			continue
		}

		// 4. In die Datenbank einfügen
		query := `
			INSERT INTO schueler_fotos (schueler_id, foto_encrypted)
			VALUES ($1, $2)
			ON CONFLICT (schueler_id) DO UPDATE SET 
				foto_encrypted = EXCLUDED.foto_encrypted,
				aktualisiert_am = CURRENT_TIMESTAMP
		`
		_, err = pool.Exec(ctx, query, schuelerID, encryptedData)
		if err != nil {
			log.Printf("Fehler beim Speichern von %s in die DB: %v", file.Name(), err)
			fehler++
			continue
		}

		log.Printf("Erfolgreich migriert: %s", file.Name())
		erfolg++

		// Optional: Alte Datei löschen/umbenennen nach erfolgreicher Migration
		// os.Rename(filePath, filePath+".migrated")
	}

	fmt.Println("-------------------------------------------------")
	fmt.Printf("Migration abgeschlossen!\nErfolgreich: %d\nÜbersprungen: %d\nFehler: %d\n", erfolg, uebersprungen, fehler)
	fmt.Println("Du kannst die migrierten Dateien im Ordner uploads/fotos/ nun löschen.")
}
