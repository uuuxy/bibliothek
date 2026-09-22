// cmd/seed füllt eine Wegwerf-Datenbank mit Test-Admin, Schülern, Titeln und Exemplaren —
// die Vorstufe zum k6-Lasttest (docs/SCRIPTS.md). Es fragt nicht, bevor es schreibt.
//
// Der Lauf selbst steht in seed(): Umfang und Ausgabe kommen von außen, Fehler kommen
// zurück. main() liefert die Umgebung (DATABASE_URL, JWT_SECRET) und den vollen Umfang;
// main_pg_test.go fährt denselben Lauf klein gegen die Test-Datenbank.
package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"math/rand"
	"os"
	"time"

	"bibliothek/auth"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Umfang ist die Größe eines Laufs. Exemplare werden in Portionen von Chunk eingefügt.
type Umfang struct {
	Schueler, Titel, Exemplare, Chunk int
}

// vollerUmfang ist die Größe für den Lasttest.
var vollerUmfang = Umfang{Schueler: 2000, Titel: 5000, Exemplare: 80000, Chunk: 10000}

const (
	adminEmail   = "scanner@test.local"
	adminBarcode = "ADMIN-SCANNER-TEST"
)

func main() {
	ctx := context.Background()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL environment variable is required")
	}
	jwtSecret := os.Getenv("JWT_SECRET")
	if len(jwtSecret) < 32 {
		log.Fatalf("FATAL: JWT_SECRET environment variable must be at least 32 characters long for security")
	}
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		log.Fatalf("Fehler beim Verbinden mit der Datenbank: %v\n", err)
	}
	defer pool.Close()

	fmt.Println("Starte massiven Daten-Import für den Stresstest...")
	startTime := time.Now()

	adminID, err := seed(ctx, pool, vollerUmfang, os.Stdout)
	if err != nil {
		log.Fatalf("%v\n", err)
	}
	authenticator, err := auth.NewAuthenticator(jwtSecret, pool, 8760*time.Hour) // 1 Jahr gültig
	if err == nil {
		token, _ := authenticator.GenerateToken(adminID, adminBarcode, auth.RoleAdmin) //nolint:errcheck
		fmt.Printf("\n========================================================\n")
		fmt.Printf("🛡️ DAST/SAST Scanner JWT (1 Jahr gültig):\n%s\n", token)
		fmt.Printf("========================================================\n\n")
	}

	fmt.Printf("🎉 Fertig in %v. Die Datenbank ist jetzt voll und bereit für k6.\n", time.Since(startTime))
}

// seed führt den Lauf aus und liefert die Id des Test-Admins — die gespeicherte, auch
// wenn das Konto schon vom vorigen Lauf da war. Jeder Schritt ist wiederholbar: Was es
// schon gibt, wird übersprungen (ON CONFLICT DO NOTHING), nichts entsteht doppelt.
func seed(ctx context.Context, pool *pgxpool.Pool, u Umfang, out io.Writer) (string, error) {
	adminID, err := generateTestAdmin(ctx, pool)
	if err != nil {
		return "", err
	}
	if err := generateStudents(ctx, pool, u.Schueler, out); err != nil {
		return "", err
	}
	titleIDs, err := generateTitles(ctx, pool, u.Titel, out)
	if err != nil {
		return "", err
	}
	if err := generateExemplare(ctx, pool, titleIDs, u.Exemplare, u.Chunk, out); err != nil {
		return "", err
	}
	return adminID, nil
}

// generateTestAdmin legt das Scanner-Konto an oder findet es. Bis zum 22.09.2026 stand
// hier eine frische UUID mit ON CONFLICT DO NOTHING: Beim zweiten Lauf blieb die Zeile
// unberührt, der Ausweis fand keine Leserzeile, und das JWT trug eine Id, die es nicht
// gibt — still, mit einer Warnung im Log (TestSeed_ZweiterLaufIstDerselbeAdmin).
func generateTestAdmin(ctx context.Context, pool *pgxpool.Pool) (string, error) {
	// ON CONFLICT … DO UPDATE statt DO NOTHING: Nur so liefert RETURNING auch die
	// vorhandene Zeile. Geändert wird nichts (aktiv bleibt, was es ist).
	var adminID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO benutzer (vorname, nachname, email, rolle, aktiv)
		VALUES ('Scanner', 'TestAdmin', $1, 'admin', true)
		ON CONFLICT (email) DO UPDATE SET aktiv = benutzer.aktiv
		RETURNING id
	`, adminEmail).Scan(&adminID); err != nil {
		return "", fmt.Errorf("test-Admin anlegen: %w", err)
	}
	// Die Ausweisnummer steht an der Leserzeile (Migration 125), die der Trigger beim
	// Anlegen des Kontos erzeugt. ZWEI Anweisungen und keine schreibende CTE: Eine Zeile,
	// die dieselbe Anweisung gerade eingefügt hat, liegt außerhalb des Schnappschusses
	// des äußeren UPDATE — es fände sie nicht und würde still 0 Zeilen ändern.
	tag, err := pool.Exec(ctx, `
		UPDATE leser SET barcode_id = $2
		WHERE id = (SELECT leser_id FROM benutzer WHERE id = $1)
	`, adminID, adminBarcode)
	if err != nil {
		return "", fmt.Errorf("ausweis des Test-Admins eintragen: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return "", fmt.Errorf("ausweis des Test-Admins eintragen: %d Leserzeilen statt 1", tag.RowsAffected())
	}
	return adminID, nil
}

func generateStudents(ctx context.Context, pool *pgxpool.Pool, n int, out io.Writer) error {
	studentBatch := &pgx.Batch{}
	for i := 1; i <= n; i++ {
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
		return fmt.Errorf("schüler einfügen: %w", err)
	}
	fmt.Fprintf(out, "✅ %d Schüler erfolgreich generiert.\n", n) //nolint:errcheck // Fortschrittszeile
	return nil
}

// generateTitles liefert die Ids der Titel, an die die Exemplare gehängt werden. Titel,
// die es schon gab (zweiter Lauf), behalten ihre gespeicherte Id — die frische aus
// dem INSERT wäre eine Id ohne Zeile, und jedes Exemplar daran ein Fremdschlüssel-Fehler.
func generateTitles(ctx context.Context, pool *pgxpool.Pool, n int, out io.Writer) ([]uuid.UUID, error) {
	titleBatch := &pgx.Batch{}
	for i := 0; i < n; i++ {
		titleBatch.Queue(`
			INSERT INTO buecher_titel (id, titel, isbn)
			VALUES ($1, $2, $3)
			ON CONFLICT (isbn) DO NOTHING
		`, uuid.New(), fmt.Sprintf("Titel %d", i+1), fmt.Sprintf("ISBN-%010d", i+1))
	}
	tRes := pool.SendBatch(ctx, titleBatch)
	if err := tRes.Close(); err != nil {
		return nil, fmt.Errorf("titel einfügen: %w", err)
	}
	rows, err := pool.Query(ctx, `SELECT id FROM buecher_titel WHERE isbn LIKE 'ISBN-%' ORDER BY isbn LIMIT $1`, n)
	if err != nil {
		return nil, fmt.Errorf("titel-Ids lesen: %w", err)
	}
	titleIDs, err := pgx.CollectRows(rows, pgx.RowTo[uuid.UUID])
	if err != nil {
		return nil, fmt.Errorf("titel-Ids lesen: %w", err)
	}
	if len(titleIDs) != n {
		return nil, fmt.Errorf("%d Titel gespeichert, %d erwartet", len(titleIDs), n)
	}
	fmt.Fprintf(out, "✅ %d Titel erfolgreich generiert.\n", n) //nolint:errcheck // Fortschrittszeile
	return titleIDs, nil
}

func generateExemplare(ctx context.Context, pool *pgxpool.Pool, titleIDs []uuid.UUID, total, chunk int, out io.Writer) error {
	for i := 0; i < total; i += chunk {
		// Die letzte Portion ist kürzer, wenn total kein Vielfaches von chunk ist —
		// sonst entstünden mehr Exemplare als bestellt (TestSeed_LegtGenauDenUmfangAn).
		bis := min(chunk, total-i)
		bookBatch := &pgx.Batch{}
		for j := 1; j <= bis; j++ {
			barcode := fmt.Sprintf("B%07d", i+j) // Generiert Barcodes wie B0000001
			titelID := titleIDs[rand.Intn(len(titleIDs))]

			bookBatch.Queue(`
				INSERT INTO buecher_exemplare (barcode_id, titel_id, ist_ausleihbar)
				VALUES ($1, $2, true)
				ON CONFLICT (barcode_id) DO NOTHING
			`, barcode, titelID)
		}

		bRes := pool.SendBatch(ctx, bookBatch)
		if err := bRes.Close(); err != nil {
			return fmt.Errorf("exemplare einfügen (Portion ab %d): %w", i, err)
		}
		fmt.Fprintf(out, "⏳ %d / %d Exemplare eingefügt...\n", i+bis, total) //nolint:errcheck // Fortschrittszeile
	}

	fmt.Fprintf(out, "✅ Alle %d Exemplare erfolgreich generiert.\n", total) //nolint:errcheck // Fortschrittszeile
	return nil
}
