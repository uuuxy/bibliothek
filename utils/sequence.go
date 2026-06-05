package utils

import (
	"context"
	"fmt"
	"regexp"
	"strconv"

	"github.com/jackc/pgx/v5"
)

// GetNextBarcodeSequence fetches the next available sequence number for barcodes
// (e.g. S-XXXXX or B-XXXXX) from the given table.
func GetNextBarcodeSequence(ctx context.Context, tx pgx.Tx, table, prefix string, forUpdate bool) (int, error) {
	// Prevent SQL injection by validating the table name
	switch table {
	case "buecher_exemplare", "schueler":
		// Allowed tables
	default:
		return 0, fmt.Errorf("invalid table name for sequence generation: %s", table)
	}

	query := fmt.Sprintf(`
		SELECT barcode_id
		FROM %s
		WHERE barcode_id LIKE $1
		ORDER BY barcode_id DESC
		LIMIT 1
	`, table)

	if forUpdate {
		query += " FOR UPDATE"
	}

	var lastBarcode string
	err := tx.QueryRow(ctx, query, prefix+"-%").Scan(&lastBarcode)

	startNum := 10001
	if err == nil {
		re := regexp.MustCompile(prefix + `-(\d+)`)
		matches := re.FindStringSubmatch(lastBarcode)
		if len(matches) > 1 {
			if parsed, parseErr := strconv.Atoi(matches[1]); parseErr == nil {
				startNum = parsed + 1
			}
		}
	} else if err != pgx.ErrNoRows {
		return 0, err
	}

	return startNum, nil
}
