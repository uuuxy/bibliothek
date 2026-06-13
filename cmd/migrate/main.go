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

	// ── Environment variables ────────────────────────────────────────────
	mysqlDSN := os.Getenv("MYSQL_DSN")
	if mysqlDSN == "" {
		log.Fatal("MYSQL_DSN environment variable is required")
	}
	pgDSN := os.Getenv("PG_DSN")
	if pgDSN == "" {
		log.Fatal("PG_DSN environment variable is required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	// ── Error log ────────────────────────────────────────────────────────
	el, err := newErrLogger()
	if err != nil {
		log.Fatalf("cannot open error log: %v", err)
	}
	defer el.close()

	// ── MySQL connection ─────────────────────────────────────────────────
	log.Println("Verbinde mit MySQL …")
	mysqlDB, err := sql.Open("mysql", mysqlDSN)
	if err != nil {
		log.Fatalf("mysql open: %v", err)
	}
	defer func() { _ = mysqlDB.Close() }()
	mysqlDB.SetMaxOpenConns(4)
	mysqlDB.SetConnMaxLifetime(5 * time.Minute)
	if err := mysqlDB.PingContext(ctx); err != nil {
		log.Fatalf("mysql ping: %v", err)
	}
	log.Println("MySQL Verbindung OK")

	// ── PostgreSQL connection pool ────────────────────────────────────────
	log.Println("Verbinde mit PostgreSQL …")
	pgPool, err := pgxpool.New(ctx, pgDSN)
	if err != nil {
		log.Fatalf("pgx pool new: %v", err)
	}
	defer pgPool.Close()
	if err := pgPool.Ping(ctx); err != nil {
		log.Fatalf("pg ping: %v", err)
	}
	log.Println("PostgreSQL Verbindung OK")

	// ── Read all MySQL titles ─────────────────────────────────────────────
	log.Println("Lese Quelldaten aus MySQL …")
	titles, err := readMySQLTitles(mysqlDB)
	if err != nil {
		log.Fatalf("mysql read: %v", err)
	}
	// #nosec G706
	log.Printf("Gelesen: %d Titel aus MySQL", len(titles))

	if *dryRun {
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
		return
	}

	// ── Resolve starting barcode sequence ────────────────────────────────
	barcodeSeq, err := highestBarcodeSeq(ctx, pgPool)
	if err != nil {
		log.Fatalf("barcode seq lookup: %v", err)
	}
	// #nosec G706
	log.Printf("Höchster vorhandener Barcode: B-%05d → Nächster: B-%05d", barcodeSeq, barcodeSeq+1)

	// ── Batch import ─────────────────────────────────────────────────────
	seenISBNs := make(map[string]int)
	totalTitles, totalCopies := 0, 0
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

	// ── Summary ───────────────────────────────────────────────────────────
	// #nosec G706
	log.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	// #nosec G706
	log.Printf("Migration abgeschlossen:")
	// #nosec G706
	log.Printf("  Quell-Titel (MySQL):      %d", len(titles))
	// #nosec G706
	log.Printf("  Importierte Titel:        %d", totalTitles)
	// #nosec G706
	log.Printf("  Importierte Exemplare:    %d", totalCopies)
	// #nosec G706
	log.Printf("  Fehler / Warnungen:       %d → %s", el.n, errorLogPath)
	// #nosec G706
	log.Printf("  Letzter Barcode:          B-%05d", barcodeSeq)
	// #nosec G706
	log.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	if el.n > 0 {
		// #nosec G706
		log.Printf("⚠  %d Einträge konnten nicht migriert werden. Details: %s", el.n, errorLogPath)
		os.Exit(2) // non-zero but not fatal: partial success
	}

	// Cleanly close MySQL before exit (pgPool closed via defer above).
	if err := mysqlDB.Close(); err != nil {
		// #nosec G706
		log.Printf("mysql close: %v", err)
	}
}

// Ensure pgx is used (satisfies unused import check when pgx is only used via pgxpool).
var _ pgx.Tx
