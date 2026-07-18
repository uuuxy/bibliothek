package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	"bibliothek/auth"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	// 1. Datenbankverbindung aufbauen (Zugangsdaten ausschließlich aus der Umgebung)
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL environment variable is required")
	}
	ctx := context.Background()
	
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		log.Fatalf("Fehler beim Verbinden mit der Datenbank: %v\n", err)
	}
	defer pool.Close()

	fmt.Println("Starte massiven Daten-Import für den Stresstest...")
	startTime := time.Now()

	// 1.1 Test-Admin für Security-Scans generieren
	adminID := uuid.New()
	adminBarcode := "ADMIN-SCANNER-TEST"
	_, err = pool.Exec(ctx, `
		INSERT INTO benutzer (id, barcode_id, vorname, nachname, email, rolle, aktiv)
		VALUES ($1, $2, 'Scanner', 'TestAdmin', 'scanner@test.local', 'admin', true)
		ON CONFLICT (email) DO NOTHING
	`, adminID, adminBarcode)
	if err != nil {
		log.Printf("Warnung: Konnte Test-Admin nicht anlegen: %v\n", err)
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if len(jwtSecret) < 32 {
		log.Fatalf("FATAL: JWT_SECRET environment variable must be at least 32 characters long for security")
	}
	authenticator, err := auth.NewAuthenticator(jwtSecret, pool, 8760*time.Hour) // 1 Jahr gültig
	if err == nil {
		token, _ := authenticator.GenerateToken(adminID.String(), adminBarcode, auth.RoleAdmin)  //nolint:errcheck
		fmt.Printf("\n========================================================\n")
		fmt.Printf("🛡️ DAST/SAST Scanner JWT (1 Jahr gültig):\n%s\n", token)
		fmt.Printf("========================================================\n\n")
	}

	// 2. 2.000 Dummy-Schüler generieren
	studentBatch := &pgx.Batch{}
	for i := 1; i <= 2000; i++ {
		barcodeID := fmt.Sprintf("S%06d", i)
		vorname := fmt.Sprintf("Vorname%d", i)
		nachname := fmt.Sprintf("Nachname%d", i)
		klasse := fmt.Sprintf("%d%s", rand.Intn(8)+5, string(rune('A'+rand.Intn(4)))) // z.B. 7B
		abgaengerJahr := time.Now().Year() + rand.Intn(5) + 1
		
		studentBatch.Queue(`
			INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr, ist_gesperrt)
			VALUES ($1, $2, $3, $4, $5, false)
			ON CONFLICT (barcode_id) WHERE deleted_at IS NULL DO NOTHING
		`, barcodeID, vorname, nachname, klasse, abgaengerJahr)
	}

	br := pool.SendBatch(ctx, studentBatch)
	if err := br.Close(); err != nil {
		log.Fatalf("Fehler beim Einfügen der Schüler: %v\n", err)
	}
	fmt.Printf("✅ 2.000 Schüler erfolgreich generiert.\n")

	// 3. 5.000 Dummy-Titel generieren
	titleBatch := &pgx.Batch{}
	titleIDs := make([]uuid.UUID, 5000)
	for i := 0; i < 5000; i++ {
		titleIDs[i] = uuid.New()
		titelName := fmt.Sprintf("Titel %d", i+1)
		isbn := fmt.Sprintf("ISBN-%010d", i+1)
		
		titleBatch.Queue(`
			INSERT INTO buecher_titel (id, titel, isbn) 
			VALUES ($1, $2, $3)
			ON CONFLICT (isbn) DO NOTHING
		`, titleIDs[i], titelName, isbn)
	}
	tRes := pool.SendBatch(ctx, titleBatch)
	if err := tRes.Close(); err != nil {
		log.Fatalf("Fehler beim Einfügen der Titel: %v\n", err)
	}
	fmt.Printf("✅ 5.000 Titel erfolgreich generiert.\n")

	// 4. 80.000 Dummy-Exemplare (Bücher) generieren
	chunkSize := 10000
	totalExemplare := 80000

	for i := 0; i < totalExemplare; i += chunkSize {
		bookBatch := &pgx.Batch{}
		for j := 1; j <= chunkSize; j++ {
			barcode := fmt.Sprintf("B%07d", i+j) // Generiert Barcodes wie B0000001
			titelID := titleIDs[rand.Intn(5000)] // Wähle zufälligen Titel
			
			bookBatch.Queue(`
				INSERT INTO buecher_exemplare (barcode_id, titel_id, ist_ausleihbar) 
				VALUES ($1, $2, true)
				ON CONFLICT (barcode_id) DO NOTHING
			`, barcode, titelID)
		}

		bRes := pool.SendBatch(ctx, bookBatch)
		if err := bRes.Close(); err != nil {
			log.Fatalf("Fehler beim Einfügen der Bücher (Chunk %d): %v\n", i, err)
		}
		fmt.Printf("⏳ %d / %d Exemplare eingefügt...\n", i+chunkSize, totalExemplare)
	}

	fmt.Printf("✅ Alle %d Exemplare erfolgreich generiert.\n", totalExemplare)
	fmt.Printf("🎉 Fertig in %v. Die Datenbank ist jetzt voll und bereit für k6.\n", time.Since(startTime))
}
