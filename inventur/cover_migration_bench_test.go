package inventur

import (
	"context"
	"fmt"
	"testing"
	"time"

	"bibliothek/db"

	"github.com/jackc/pgx/v5/pgconn"
)

// mockDB is a simple mock database connection to measure update overhead.
type mockDB struct {
	db.PgxPoolIface
	updates  int
	queryErr error
	execErr  error
}

func (m *mockDB) Exec(ctx context.Context, sql string, arguments ...interface{}) (pgconn.CommandTag, error) {
	// Simulate some DB latency
	time.Sleep(5 * time.Millisecond)
	m.updates++
	return pgconn.CommandTag{}, m.execErr
}

func BenchmarkSequentialUpdate(b *testing.B) {
	mdb := &mockDB{}
	ctx := context.Background()
	ids := make([]string, 100)
	urls := make([]string, 100)
	for i := 0; i < 100; i++ {
		ids[i] = fmt.Sprintf("id-%d", i)
		urls[i] = fmt.Sprintf("/uploads/cover-%d.jpg", i)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for j := 0; j < 100; j++ {
			_, _ = mdb.Exec(ctx, "UPDATE buecher_titel SET cover_url = $1 WHERE id = $2", urls[j], ids[j])
		}
	}
}

func BenchmarkBatchUpdateUNNEST(b *testing.B) {
	mdb := &mockDB{}
	ctx := context.Background()
	ids := make([]string, 100)
	urls := make([]string, 100)
	for i := 0; i < 100; i++ {
		ids[i] = fmt.Sprintf("id-%d", i)
		urls[i] = fmt.Sprintf("/uploads/cover-%d.jpg", i)
	}

	query := `
		UPDATE buecher_titel
		SET cover_url = data.cover_url
		FROM (SELECT UNNEST($1::uuid[]) AS id, UNNEST($2::text[]) AS cover_url) AS data
		WHERE buecher_titel.id = data.id
	`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = mdb.Exec(ctx, query, ids, urls)
	}
}
