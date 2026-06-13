package main

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

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

// nullStr converts an empty string to a typed nil suitable for pgx nullable columns.
func nullStr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

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
