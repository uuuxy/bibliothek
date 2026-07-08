package repository

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strconv"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// DBQueryer interface abstracts pgxpool.Pool and pgx.Tx so that
// the repository can be used both inside and outside of transactions.
type DBQueryer interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
}

// SequenceRepository provides methods to calculate sequential strings,
// such as auto-incrementing barcodes with prefixes (e.g., "B-10001").
type SequenceRepository struct {
	db DBQueryer
}

// NewSequenceRepository initializes a new SequenceRepository.
func NewSequenceRepository(db DBQueryer) *SequenceRepository {
	return &SequenceRepository{db: db}
}

// GetNextSequence fetches the highest existing barcode with a given prefix,
// increments the numeric part by one, and returns it.
// Default fallback starts at 10001 if no prior entry is found.
// Note: Since table and column names cannot be parameterized in Postgres,
// they are securely formatted using Sprintf. We trust the inputs as they are developer-defined constants.
func (r *SequenceRepository) GetNextSequence(ctx context.Context, tableName, colName, prefix string) (int, error) {
	// Construct the dynamic query.
	// This is safe because tableName and colName are never user-provided.
	query := fmt.Sprintf(`
		SELECT %s 
		FROM %s 
		WHERE %s LIKE $1 
		ORDER BY %s DESC 
		LIMIT 1
	`, colName, tableName, colName, colName)

	likePattern := prefix + "%"
	var lastValue string

	err := r.db.QueryRow(ctx, query, likePattern).Scan(&lastValue)

	startNum := 10001

	if err == nil {
		// Regex to extract the numeric part following the prefix
		// e.g. "B-10001" -> "10001"
		re := regexp.MustCompile(fmt.Sprintf(`^%s(\d+)`, regexp.QuoteMeta(prefix)))
		matches := re.FindStringSubmatch(lastValue)
		if len(matches) > 1 {
			if parsed, parseErr := strconv.Atoi(matches[1]); parseErr == nil {
				startNum = parsed + 1
			}
		}
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return 0, fmt.Errorf("failed to query next sequence: %w", err)
	}

	return startNum, nil
}
