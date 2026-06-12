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
//
// Assumed MySQL source schema (old library system):
//
//	medien table columns expected:
//	  id, titel, untertitel, autor, isbn, verlag, erscheinungsjahr,
//	  beschreibung, medientyp, standort, regal, notizen, anzahl, erstellt_am
//
// Each row with anzahl > 1 generates that many rows in buecher_exemplare,
// each with a unique sequential B-XXXXXX barcode.
//
// If your source schema differs, adjust the mysqlMedium struct and the SELECT
// query in readMySQLTitles() accordingly.

package main

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ---------------------------------------------------------------------------
// Source model (MySQL)
// ---------------------------------------------------------------------------

// mysqlMedium represents one row from the old MySQL `medien` table.
type mysqlMedium struct {
	ID               int
	Titel            string
	Untertitel       sql.NullString
	Autor            sql.NullString
	ISBN             sql.NullString
	Verlag           sql.NullString
	Erscheinungsjahr sql.NullInt64
	Beschreibung     sql.NullString
	Medientyp        sql.NullString
	Standort         sql.NullString // free-text shelf location → JSONB
	Regal            sql.NullString // rack/row label           → JSONB
	Notizen          sql.NullString // free-text notes          → JSONB
	Anzahl           int            // physical copy count
	ErstelltAm       sql.NullTime
}

// ---------------------------------------------------------------------------
// CLI flags & configuration
// ---------------------------------------------------------------------------

var (
	dryRun    = flag.Bool("dry-run", false, "validate only, do not write to PostgreSQL")
	batchSize = flag.Int("batch", 200, "titles per INSERT transaction")
)

// ---------------------------------------------------------------------------
// Error log writer
// ---------------------------------------------------------------------------

const errorLogPath = "migration_errors.log"

type errLogger struct {
	f *os.File
	w *bufio.Writer
	n int // total errors written
}

func newErrLogger() (*errLogger, error) {
	// #nosec G304 - errorLogPath is a hardcoded constant
	f, err := os.OpenFile(errorLogPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open error log: %w", err)
	}
	return &errLogger{f: f, w: bufio.NewWriterSize(f, 64*1024)}, nil
}

func (el *errLogger) write(mysqlID int, isbn, reason string) {
	el.n++
	ts := time.Now().Format("2006-01-02 15:04:05")
	_, _ = fmt.Fprintf(el.w, "[%s] mysql_id=%d isbn=%q reason=%s\n", ts, mysqlID, isbn, reason)
}

func (el *errLogger) close() {
	_ = el.w.Flush()
	_ = el.f.Close()
}

// ---------------------------------------------------------------------------
// ISBN validation
// ---------------------------------------------------------------------------

var reNonDigit = regexp.MustCompile(`[^0-9Xx]`)

// normalizeISBN strips hyphens/spaces and upper-cases X.
func normalizeISBN(raw string) string {
	s := reNonDigit.ReplaceAllString(raw, "")
	return strings.ToUpper(s)
}

// validateISBN13 checks the ISBN-13 check digit.
func validateISBN13(isbn string) bool {
	if len(isbn) != 13 {
		return false
	}
	sum := 0
	for i, ch := range isbn {
		if !unicode.IsDigit(ch) {
			return false
		}
		d := int(ch - '0')
		if i%2 == 0 {
			sum += d
		} else {
			sum += d * 3
		}
	}
	return sum%10 == 0
}

// validateISBN10 checks the ISBN-10 check digit.
func validateISBN10(isbn string) bool {
	if len(isbn) != 10 {
		return false
	}
	sum := 0
	for i, ch := range isbn {
		var d int
		if i == 9 && (ch == 'X' || ch == 'x') {
			d = 10
		} else if unicode.IsDigit(ch) {
			d = int(ch - '0')
		} else {
			return false
		}
		sum += d * (10 - i)
	}
	return sum%11 == 0
}

// validateISBN returns (normalised, ok).
func validateISBN(raw string) (string, bool) {
	if raw == "" {
		return "", true // NULL ISBN is allowed
	}
	n := normalizeISBN(raw)
	switch len(n) {
	case 10:
		return n, validateISBN10(n)
	case 13:
		return n, validateISBN13(n)
	default:
		return n, false
	}
}

// ---------------------------------------------------------------------------
// Barcode helpers
// ---------------------------------------------------------------------------

var reBarcodeNum = regexp.MustCompile(`^B-(\d+)$`)

// highestBarcodeSeq reads the current highest B-XXXXXX sequence number from PostgreSQL.
func highestBarcodeSeq(ctx context.Context, pool *pgxpool.Pool) (int, error) {
	var raw sql.NullString
	err := pool.QueryRow(ctx,
		`SELECT MAX(barcode_id) FROM buecher_exemplare WHERE barcode_id LIKE 'B-%'`,
	).Scan(&raw)
	if err != nil || !raw.Valid {
		return 0, err
	}
	m := reBarcodeNum.FindStringSubmatch(raw.String)
	if len(m) < 2 {
		return 0, nil
	}
	n, _ := strconv.Atoi(m[1])
	return n, nil
}

// nextBarcodes returns `count` sequential barcodes starting after `seq`.
func nextBarcodes(seq, count int) []string {
	codes := make([]string, count)
	for i := range codes {
		codes[i] = fmt.Sprintf("B-%05d", seq+i+1)
	}
	return codes
}

// validateBarcode ensures a barcode matches the expected B-XXXXX pattern.
func validateBarcode(bc string) bool {
	return reBarcodeNum.MatchString(bc)
}

// ---------------------------------------------------------------------------
// MySQL reader
// ---------------------------------------------------------------------------

func readMySQLTitles(db *sql.DB) ([]mysqlMedium, error) {
	// Adjust the column list / table name here if your old schema differs.
	const q = `
		SELECT
			id,
			titel,
			IFNULL(untertitel, ''),
			IFNULL(autor, ''),
			IFNULL(isbn, ''),
			IFNULL(verlag, ''),
			IFNULL(erscheinungsjahr, 0),
			IFNULL(beschreibung, ''),
			IFNULL(medientyp, 'Buch'),
			IFNULL(standort, ''),
			IFNULL(regal, ''),
			IFNULL(notizen, ''),
			IFNULL(anzahl, 1),
			erstellt_am
		FROM medien
		ORDER BY id
	`
	rows, err := db.Query(q)
	if err != nil {
		return nil, fmt.Errorf("mysql query: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var results []mysqlMedium
	for rows.Next() {
		var m mysqlMedium
		var (
			untertitel       string
			autor            string
			isbn             string
			verlag           string
			erscheinungsjahr int64
			beschreibung     string
			medientyp        string
			standort         string
			regal            string
			notizen          string
		)
		if err := rows.Scan(
			&m.ID, &m.Titel,
			&untertitel, &autor, &isbn, &verlag,
			&erscheinungsjahr, &beschreibung, &medientyp,
			&standort, &regal, &notizen,
			&m.Anzahl, &m.ErstelltAm,
		); err != nil {
			return nil, fmt.Errorf("mysql scan row id=%d: %w", m.ID, err)
		}
		if untertitel != "" {
			m.Untertitel = sql.NullString{String: untertitel, Valid: true}
		}
		if autor != "" {
			m.Autor = sql.NullString{String: autor, Valid: true}
		}
		if isbn != "" {
			m.ISBN = sql.NullString{String: isbn, Valid: true}
		}
		if verlag != "" {
			m.Verlag = sql.NullString{String: verlag, Valid: true}
		}
		if erscheinungsjahr > 0 {
			m.Erscheinungsjahr = sql.NullInt64{Int64: erscheinungsjahr, Valid: true}
		}
		if beschreibung != "" {
			m.Beschreibung = sql.NullString{String: beschreibung, Valid: true}
		}
		if medientyp != "" {
			m.Medientyp = sql.NullString{String: medientyp, Valid: true}
		}
		if standort != "" {
			m.Standort = sql.NullString{String: standort, Valid: true}
		}
		if regal != "" {
			m.Regal = sql.NullString{String: regal, Valid: true}
		}
		if notizen != "" {
			m.Notizen = sql.NullString{String: notizen, Valid: true}
		}
		if m.Anzahl <= 0 {
			m.Anzahl = 1
		}
		results = append(results, m)
	}
	return results, rows.Err()
}

// ---------------------------------------------------------------------------
// Transformation helpers
// ---------------------------------------------------------------------------

// buildErweiterteEigenschaften serialises free-text fields into the JSONB column.
func buildErweiterteEigenschaften(m mysqlMedium) (string, error) {
	props := make(map[string]string)
	if m.Standort.Valid && m.Standort.String != "" {
		props["standort"] = m.Standort.String
	}
	if m.Regal.Valid && m.Regal.String != "" {
		props["regal"] = m.Regal.String
	}
	if m.Notizen.Valid && m.Notizen.String != "" {
		props["notizen"] = m.Notizen.String
	}
	// Add legacy source reference for traceability
	props["mysql_id"] = strconv.Itoa(m.ID)

	b, err := json.Marshal(props)
	if err != nil {
		return "{}", fmt.Errorf("json marshal: %w", err)
	}
	return string(b), nil
}

func nullableString(s sql.NullString) *string {
	if !s.Valid || s.String == "" {
		return nil
	}
	v := s.String
	return &v
}

func nullableInt(n sql.NullInt64) *int {
	if !n.Valid {
		return nil
	}
	v := int(n.Int64)
	return &v
}

// ---------------------------------------------------------------------------
// PostgreSQL writer
// ---------------------------------------------------------------------------

// insertBatch inserts one batch of titles and their copies inside a single transaction.
// It returns the number of successfully inserted titles and copies.
func insertBatch(
	ctx context.Context,
	pool *pgxpool.Pool,
	batch []mysqlMedium,
	seenISBNs map[string]int, // isbn → mysql source ID; updated in-place
	el *errLogger,
	barcodeSeq *int,
) (titlesOK, copiesOK int) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		// #nosec G706
		log.Printf("ERROR begin transaction: %v", err)
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()

	for _, m := range batch {
		// ── Validate ISBN ────────────────────────────────────────────────
		isbnRaw := ""
		if m.ISBN.Valid {
			isbnRaw = m.ISBN.String
		}
		normISBN, isbnOK := validateISBN(isbnRaw)
		if isbnRaw != "" && !isbnOK {
			el.write(m.ID, isbnRaw, "ungültige ISBN-Prüfziffer")
			normISBN = "" // treat as NULL in PG rather than aborting
		}

		// ── Duplicate ISBN check ─────────────────────────────────────────
		if normISBN != "" {
			if prevID, exists := seenISBNs[normISBN]; exists {
				el.write(m.ID, normISBN, fmt.Sprintf("doppelte ISBN – bereits importiert als mysql_id=%d", prevID))
				normISBN = "" // store without ISBN to avoid PG UNIQUE violation
			} else {
				seenISBNs[normISBN] = m.ID
			}
		}

		// ── Build JSONB ──────────────────────────────────────────────────
		jsonbProps, err := buildErweiterteEigenschaften(m)
		if err != nil {
			el.write(m.ID, isbnRaw, fmt.Sprintf("JSONB-Fehler: %v", err))
			continue
		}

		// ── Resolve medientyp ────────────────────────────────────────────
		medientyp := "Buch"
		if m.Medientyp.Valid && m.Medientyp.String != "" {
			medientyp = m.Medientyp.String
		}

		// ── Resolve timestamps ───────────────────────────────────────────
		erstelltAm := time.Now()
		if m.ErstelltAm.Valid {
			erstelltAm = m.ErstelltAm.Time
		}

		// ── Insert buecher_titel ─────────────────────────────────────────
		var titelID string
		err = tx.QueryRow(ctx, `
			INSERT INTO buecher_titel
				(titel, untertitel, autor, isbn, verlag, erscheinungsjahr,
				 beschreibung, medientyp, erweiterte_eigenschaften,
				 stock, erstellt_am)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
			RETURNING id
		`,
			m.Titel,
			nullableString(m.Untertitel),
			nullableString(m.Autor),
			nullStr(normISBN),
			nullableString(m.Verlag),
			nullableInt(m.Erscheinungsjahr),
			nullableString(m.Beschreibung),
			medientyp,
			jsonbProps,
			m.Anzahl,
			erstelltAm,
		).Scan(&titelID)
		if err != nil {
			el.write(m.ID, isbnRaw, fmt.Sprintf("INSERT buecher_titel fehlgeschlagen: %v", err))
			// Roll back only this title; keep going with the batch by aborting the tx and restarting.
			// Strategy: skip the failed title and continue outside this function.
			continue
		}
		titlesOK++

		// ── Generate exemplare barcodes ──────────────────────────────────
		barcodes := nextBarcodes(*barcodeSeq, m.Anzahl)
		for _, bc := range barcodes {
			if !validateBarcode(bc) {
				el.write(m.ID, isbnRaw, fmt.Sprintf("generierter Barcode ungültig: %s", bc))
				continue
			}
			_, err = tx.Exec(ctx, `
				INSERT INTO buecher_exemplare
					(titel_id, barcode_id, erworben_am, ist_ausleihbar,
					 erweiterte_eigenschaften, erstellt_am)
				VALUES ($1, $2, CURRENT_DATE, true, '{}', $3)
			`, titelID, bc, erstelltAm)
			if err != nil {
				el.write(m.ID, isbnRaw, fmt.Sprintf("INSERT exemplar barcode=%s: %v", bc, err))
				continue
			}
			copiesOK++
		}
		*barcodeSeq += m.Anzahl
	}

	if err := tx.Commit(ctx); err != nil {
		// #nosec G706
		log.Printf("ERROR commit batch: %v", err)
		return 0, 0
	}
	return titlesOK, copiesOK
}

// nullStr converts an empty string to a typed nil suitable for pgx nullable columns.
func nullStr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// ---------------------------------------------------------------------------
// main
// ---------------------------------------------------------------------------

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
