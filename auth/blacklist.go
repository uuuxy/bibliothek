package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// DatabasePool defines the interface for database operations needed by the blacklist.
type DatabasePool interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// TokenBlacklist is a database-backed store for invalidated JWTs.
// It stores the SHA-256 hash of the token mapped to its expiration time.
type TokenBlacklist struct {
	pool   DatabasePool
	ctx    context.Context
	cancel context.CancelFunc
}

// NewTokenBlacklist initializes a new TokenBlacklist and starts a background
// cleanup goroutine to prevent the table from growing indefinitely with expired tokens.
func NewTokenBlacklist(pool DatabasePool) *TokenBlacklist {
	ctx, cancel := context.WithCancel(context.Background())
	b := &TokenBlacklist{
		pool:   pool,
		ctx:    ctx,
		cancel: cancel,
	}

	// Start the cleanup routine
	go b.cleanupLoop()

	return b
}

// hashToken computes a SHA-256 hash of the token string.
// We hash the token instead of storing it raw to save space and avoid
// keeping sensitive tokens in the database longer than necessary.
func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

// Add inserts a token into the revoked_tokens table with its expiration time.
func (b *TokenBlacklist) Add(token string, expiresAt time.Time) {
	hash := hashToken(token)
	// We use a short timeout for the DB operation, since this is called on logout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, _ = b.pool.Exec(ctx, `
		INSERT INTO revoked_tokens (token_signature, expires_at)
		VALUES ($1, $2)
		ON CONFLICT (token_signature) DO NOTHING
	`, hash, expiresAt)
}

// IsBlacklisted checks if a token exists in the revoked_tokens table.
func (b *TokenBlacklist) IsBlacklisted(token string) bool {
	hash := hashToken(token)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	var exists bool
	err := b.pool.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM revoked_tokens WHERE token_signature = $1)
	`, hash).Scan(&exists)
	
	if err != nil {
		// Fail-closed: if the DB is down and we can't verify the token isn't revoked,
		// deny access. This is the safer security posture for a school system.
		return true
	}
	return exists
}

// Stop cleanly stops the background cleanup routine.
func (b *TokenBlacklist) Stop() {
	b.cancel()
}

// cleanupLoop periodically removes expired tokens from the DB.
func (b *TokenBlacklist) cleanupLoop() {
	ticker := time.NewTicker(15 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-b.ctx.Done():
			return
		case <-ticker.C:
			b.cleanup()
		}
	}
}

// cleanup deletes any tokens from revoked_tokens that have passed their expiration time.
func (b *TokenBlacklist) cleanup() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, _ = b.pool.Exec(ctx, `DELETE FROM revoked_tokens WHERE expires_at < NOW()`)
}
