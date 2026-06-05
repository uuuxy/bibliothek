package db

import (
	"context"
	"os"
	"testing"
	"github.com/jackc/pgx/v5/pgxpool"
)

func BenchmarkInitPermissionsPg(b *testing.B) {
	dbUrl := os.Getenv("DATABASE_URL")
	if dbUrl == "" {
		b.Skip("DATABASE_URL not set")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbUrl)
	if err != nil {
		b.Fatalf("failed to connect to db: %v", err)
	}
	defer pool.Close()

	database := &Database{Pool: pool}

	// Make sure the schema is created and initialized before benching the whole thing
	err = database.InitPermissions(ctx)
	if err != nil {
		b.Fatalf("failed initial init: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := database.InitPermissions(ctx)
		if err != nil {
			b.Fatalf("failed to init permissions: %v", err)
		}
	}
}
