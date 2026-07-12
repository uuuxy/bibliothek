package main

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"strconv"
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
	n, _ := strconv.Atoi(m[1])  //nolint:errcheck
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
