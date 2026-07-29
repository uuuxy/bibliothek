package main

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"github.com/jackc/pgx/v5/pgxpool"
)

var reNonDigit = regexp.MustCompile(`[^0-9Xx]`)
var reBarcodeNum = regexp.MustCompile(`^B-(\d+)$`)

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
