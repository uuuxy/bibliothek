package inventur

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDB(t *testing.T) {
	ctx := context.Background()

	t.Run("empty URL", func(t *testing.T) {
		pool, err := NewDB(ctx, "")
		assert.Error(t, err)
		assert.Nil(t, pool)
		assert.Contains(t, err.Error(), "database URL darf nicht leer sein")
	})

	t.Run("invalid URL", func(t *testing.T) {
		// "invalid-url" isn't a valid postgres DSN usually
		pool, err := NewDB(ctx, " ://invalid-url")
		assert.Error(t, err)
		assert.Nil(t, pool)
		assert.Contains(t, err.Error(), "ungültige database URL")
	})

	t.Run("unreachable DB", func(t *testing.T) {
		pool, err := NewDB(ctx, "postgres://user:pass@127.0.0.1:1/nonexistent?sslmode=disable")
		assert.Error(t, err)
		assert.Nil(t, pool)
		assert.Contains(t, err.Error(), "postgres nicht erreichbar")
	})

	t.Run("valid DB", func(t *testing.T) {
		dbURL := os.Getenv("TEST_DB")
		if dbURL == "" {
			t.Skip("TEST_DB not set. Skipping real DB connection test.")
		}

		pool, err := NewDB(ctx, dbURL)
		require.NoError(t, err)
		assert.NotNil(t, pool)
		defer pool.Close()

		err = pool.Ping(ctx)
		assert.NoError(t, err)
	})
}
