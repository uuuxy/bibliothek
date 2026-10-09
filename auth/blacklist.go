package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"log"
	"time"

	"bibliothek/repository"
)

// DatabasePool ist der Zugang zur Datenbank, den auth/ an die Abfragen in repository/ reicht.
type DatabasePool = repository.DBQueryer

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

// Add trägt ein Token mit seiner Ablaufzeit in revoked_tokens ein.
//
// Der Fehler geht an den Aufrufer zurück, statt nur in einer Logzeile zu enden: Ein
// gescheiterter Widerruf heißt, dass das Token bis zu seinem natürlichen Ablauf gültig
// bleibt (bis zu zwölf Stunden). Wer danach „abgemeldet" meldet, meldet etwas, das nicht
// stattgefunden hat (Register 12.09.2026; Frage 5 in docs/sweeps.md).
//
// Null betroffene Zeilen sind kein Fehlschlag: Dasselbe Token ist dann schon widerrufen, das
// Ziel ist erreicht (repository.WiderrufeToken).
func (b *TokenBlacklist) Add(token string, expiresAt time.Time) error {
	hash := hashToken(token)
	// We use a short timeout for the DB operation, since this is called on logout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := repository.WiderrufeToken(ctx, b.pool, hash, expiresAt); err != nil {
		// Security-relevant: a failed revocation means the token stays valid until expiry.
		log.Printf("token-blacklist: WARN Token konnte nicht widerrufen werden: %v", err)
		return err
	}
	return nil
}

// IsBlacklisted prüft, ob das Token in revoked_tokens steht. Ein Datenbankfehler kommt als
// Fehler zurück und nicht als „widerrufen": Der Aufrufer lehnt die Anfrage in beiden Fällen
// ab (fail-closed), antwortet aber verschieden — widerrufen = 401, nicht prüfbar = 503.
func (b *TokenBlacklist) IsBlacklisted(token string) (bool, error) {
	hash := hashToken(token)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	exists, err := repository.TokenWiderrufen(ctx, b.pool, hash)
	if err != nil {
		return false, err
	}
	return exists, nil
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

	if err := repository.LoescheAbgelaufeneWiderrufe(ctx, b.pool); err != nil {
		log.Printf("token-blacklist: Aufräumen abgelaufener Tokens fehlgeschlagen: %v", err)
	}
}
