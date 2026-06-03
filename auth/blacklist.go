package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"sync"
	"time"
)

// TokenBlacklist is a thread-safe in-memory store for invalidated JWTs.
// It stores the SHA-256 hash of the token mapped to its expiration time.
type TokenBlacklist struct {
	mu     sync.RWMutex
	tokens map[string]time.Time
	ctx    context.Context
	cancel context.CancelFunc
}

// NewTokenBlacklist initializes a new TokenBlacklist and starts a background
// cleanup goroutine to prevent memory leaks from expired tokens.
func NewTokenBlacklist() *TokenBlacklist {
	ctx, cancel := context.WithCancel(context.Background())
	b := &TokenBlacklist{
		tokens: make(map[string]time.Time),
		ctx:    ctx,
		cancel: cancel,
	}

	// Start the cleanup routine
	go b.cleanupLoop()

	return b
}

// hashToken computes a SHA-256 hash of the token string.
// We hash the token instead of storing it raw to save memory and avoid
// keeping sensitive tokens in memory longer than necessary.
func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

// Add inserts a token into the blacklist with its expiration time.
func (b *TokenBlacklist) Add(token string, expiresAt time.Time) {
	b.mu.Lock()
	defer b.mu.Unlock()

	hash := hashToken(token)
	b.tokens[hash] = expiresAt
}

// IsBlacklisted checks if a token exists in the blacklist.
func (b *TokenBlacklist) IsBlacklisted(token string) bool {
	b.mu.RLock()
	defer b.mu.RUnlock()

	hash := hashToken(token)
	_, exists := b.tokens[hash]
	return exists
}

// Stop cleanly stops the background cleanup routine.
func (b *TokenBlacklist) Stop() {
	b.cancel()
}

// cleanupLoop periodically removes expired tokens from the map to prevent memory leaks.
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

// cleanup iterates through the map and deletes any tokens that have passed their expiration time.
func (b *TokenBlacklist) cleanup() {
	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()
	for hash, exp := range b.tokens {
		if now.After(exp) {
			delete(b.tokens, hash)
		}
	}
}
