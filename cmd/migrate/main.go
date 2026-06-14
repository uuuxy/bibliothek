// cmd/migrate/main.go
//
// One-shot MySQL → PostgreSQL migration tool for the Bibliothek system.
//
// Usage:
//
//	MYSQL_DSN="user:pass@tcp(host:3306)/olddb?parseTime=true" \
//	PG_DSN="postgres://user:pass@host:5432/newdb" \
//	go run ./cmd/migrate [--dry-run] [--batch 500]
//
// Required environment variables:
//
//	MYSQL_DSN   – MySQL DSN in go-sql-driver/mysql format
//	PG_DSN      – PostgreSQL connection string (URL or DSN)
//
// Optional flags:
//
//	--dry-run   – Validate and log only; make no writes to PostgreSQL
//	--batch N   – Number of titles to insert per transaction (default: 200)

package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	dryRun    = flag.Bool("dry-run", false, "validate only, do not write to PostgreSQL")
	batchSize = flag.Int("batch", 200, "titles per INSERT transaction")
)

func main() {
	flag.Parse()

	mysqlDSN, pgDSN := parseEnv()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	el, err := newErrLogger()
	if err != nil {
		log.Fatalf("cannot open error log: %v", err)
	}
	defer el.close()

	mysqlDB := connectMySQL(ctx, mysqlDSN)
	defer func() { _ = mysqlDB.Close() }()

	pgPool := connectPostgres(ctx, pgDSN)
	defer pgPool.Close()

	log.Println("Lese Quelldaten aus MySQL …")
	titles, err := readMySQLTitles(mysqlDB)
	if err != nil {
		log.Fatalf("mysql read: %v", err)
	}
	// #nosec G706
	log.Printf("Gelesen: %d Titel aus MySQL", len(titles))

	if *dryRun {
		doDryRun(titles, el)
		return
	}

	barcodeSeq, totalTitles, totalCopies := doImport(ctx, pgPool, titles, el)

	printSummary(len(titles), totalTitles, totalCopies, el.n, barcodeSeq)

	// Cleanly close MySQL before exit (pgPool closed via defer above).
	if err := mysqlDB.Close(); err != nil {
		// #nosec G706
		log.Printf("mysql close: %v", err)
	}
}

func parseEnv() (mysqlDSN, pgDSN string) {
	mysqlDSN = os.Getenv("MYSQL_DSN")
	if mysqlDSN == "" {
		log.Fatal("MYSQL_DSN environment variable is required")
	}
	pgDSN = os.Getenv("PG_DSN")
	if pgDSN == "" {
		log.Fatal("PG_DSN environment variable is required")
	}
	return mysqlDSN, pgDSN
}

func connectMySQL(ctx context.Context, dsn string) *sql.DB {
	log.Println("Verbinde mit MySQL …")
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("mysql open: %v", err)
	}
	db.SetMaxOpenConns(4)
	db.SetConnMaxLifetime(5 * time.Minute)
	if err := db.PingContext(ctx); err != nil {
		log.Fatalf("mysql ping: %v", err)
	}
	log.Println("MySQL Verbindung OK")
	return db
}

func connectPostgres(ctx context.Context, dsn string) *pgxpool.Pool {
	log.Println("Verbinde mit PostgreSQL …")
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		log.Fatalf("pgx pool new: %v", err)
	}
	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("pg ping: %v", err)
	}
	log.Println("PostgreSQL Verbindung OK")
	return pool
}

func doDryRun(titles []mysqlMedium, el *errLogger) {
	log.Println("DRY-RUN: Validiere Datensätze ohne Schreibzugriff auf PostgreSQL …")
	seenISBNs := make(map[string]int)
	validationErrors := 0
	for _, m := range titles {
		isbnRaw := ""
		if m.ISBN.Valid {
			isbnRaw = m.ISBN.String
		}
		normISBN, ok := validateISBN(isbnRaw)
		if isbnRaw != "" && !ok {
			el.write(m.ID, isbnRaw, "ungültige ISBN-Prüfziffer")
			validationErrors++
		}
		if normISBN != "" {
			if prevID, exists := seenISBNs[normISBN]; exists {
				el.write(m.ID, normISBN, fmt.Sprintf("doppelte ISBN – Kollision mit mysql_id=%d", prevID))
				validationErrors++
			} else {
				seenISBNs[normISBN] = m.ID
			}
		}
	}
	// #nosec G706
	log.Printf("DRY-RUN abgeschlossen: %d Validierungsfehler → %s", validationErrors, errorLogPath)
}

func doImport(ctx context.Context, pgPool *pgxpool.Pool, titles []mysqlMedium, el *errLogger) (barcodeSeq, totalTitles, totalCopies int) {
	barcodeSeq, err := highestBarcodeSeq(ctx, pgPool)
	if err != nil {
		log.Fatalf("barcode seq lookup: %v", err)
	}
	// #nosec G706
	log.Printf("Höchster vorhandener Barcode: B-%05d → Nächster: B-%05d", barcodeSeq, barcodeSeq+1)

	seenISBNs := make(map[string]int)
	totalBatches := int(math.Ceil(float64(len(titles)) / float64(*batchSize)))

	for i := 0; i < len(titles); i += *batchSize {
		end := i + *batchSize
		if end > len(titles) {
			end = len(titles)
		}
		batch := titles[i:end]
		batchNum := (i / *batchSize) + 1

		// #nosec G706
		log.Printf("Batch %d/%d: importiere %d Titel …", batchNum, totalBatches, len(batch))
		t, c := insertBatch(ctx, pgPool, batch, seenISBNs, el, &barcodeSeq)
		totalTitles += t
		totalCopies += c
		// #nosec G706
		log.Printf("  → %d Titel, %d Exemplare eingetragen", t, c)
	}
	return barcodeSeq, totalTitles, totalCopies
}

func printSummary(numTitles, totalTitles, totalCopies, errorCount, barcodeSeq int) {
	// #nosec G706
	log.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	// #nosec G706
	log.Printf("Migration abgeschlossen:")
	// #nosec G706
	log.Printf("  Quell-Titel (MySQL):      %d", numTitles)
	// #nosec G706
	log.Printf("  Importierte Titel:        %d", totalTitles)
	// #nosec G706
	log.Printf("  Importierte Exemplare:    %d", totalCopies)
	// #nosec G706
	log.Printf("  Fehler / Warnungen:       %d → %s", errorCount, errorLogPath)
	// #nosec G706
	log.Printf("  Letzter Barcode:          B-%05d", barcodeSeq)
	// #nosec G706
	log.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	if errorCount > 0 {
		// #nosec G706
		log.Printf("⚠  %d Einträge konnten nicht migriert werden. Details: %s", errorCount, errorLogPath)
		os.Exit(2) // non-zero but not fatal: partial success
	}
}

// Ensure pgx is used (satisfies unused import check when pgx is only used via pgxpool).
var _ pgx.Tx
