package main

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"

	"github.com/jackc/pgx/v5/pgxpool"
)

var reBarcodeNum = regexp.MustCompile(`^B-(\d+)$`)

// highestBarcodeSeq liest die höchste bereits vergebene B-XXXXX-Nummer aus PostgreSQL.
//
// Die Nummer wird in SQL NUMERISCH bestimmt, nicht per MAX(barcode_id): barcode_id ist
// VARCHAR, dort gilt die Zeichenfolge-Ordnung. Ab 100.000 Exemplaren ist 'B-99999'
// größer als 'B-100000' (weil '9' > '1'), der Zähler wäre auf 99999 zurückgesprungen und
// hätte anschließend Barcodes vergeben, die längst existieren — 23505 auf
// buecher_exemplare.barcode_id, und zwar für jeden weiteren Datensatz. Für eine
// Schulbibliothek mit rund 80.000 Titeln ist diese Schwelle erreichbar, nicht theoretisch.
//
// Die Regex-Bedingung schützt zugleich die Umwandlung: nur reine Ziffernfolgen bis neun
// Stellen werden überhaupt gelesen. Ein von Hand vergebener Barcode wie 'B-2024/A' bringt
// so weder den CAST zum Scheitern noch setzt er — wie bisher — den Zähler stillschweigend
// auf 0 zurück, was jede folgende Nummer kollidieren ließe.
func highestBarcodeSeq(ctx context.Context, pool *pgxpool.Pool) (int, error) {
	var seq sql.NullInt64
	err := pool.QueryRow(ctx, `
		SELECT MAX(CAST(substring(barcode_id FROM 3) AS BIGINT))
		FROM buecher_exemplare
		WHERE barcode_id ~ '^B-[0-9]{1,9}$'
	`).Scan(&seq)
	if err != nil {
		return 0, fmt.Errorf("höchsten Barcode ermitteln: %w", err)
	}
	if !seq.Valid {
		return 0, nil // noch keine generierten Barcodes vorhanden
	}
	return int(seq.Int64), nil
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
