package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"log"
	"time"

	"bibliothek/repository"
)

// Datenbank ist der Zugang zur Datenbank, den auth/ an die Abfragen in repository/ reicht.
type Datenbank = repository.Datenbank

// Sperrliste führt die widerrufenen Tokens in der Tabelle revoked_tokens, als Hash mit ihrer
// Ablaufzeit.
type Sperrliste struct {
	pool Datenbank
}

// newSperrliste baut die Sperrliste. Abgelaufene Tokens räumt Anmeldung.RaeumeAuf ab.
func newSperrliste(pool Datenbank) *Sperrliste {
	return &Sperrliste{pool: pool}
}

// hashDesTokens liefert den SHA-256-Hash des Tokens. In der Tabelle steht nur der Hash: Er ist
// kürzer, und ein Token selbst liegt nicht länger als nötig in der Datenbank.
func hashDesTokens(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

// Widerrufe trägt ein Token mit seiner Ablaufzeit in revoked_tokens ein.
//
// Der Fehler geht an den Aufrufer zurück, statt nur in einer Logzeile zu enden: Ein
// gescheiterter Widerruf heißt, dass das Token bis zu seinem natürlichen Ablauf gültig bleibt
// (bis zu zwölf Stunden). Wer danach „abgemeldet" meldet, meldet etwas, das nicht stattgefunden
// hat (Frage 5 in docs/sweeps.md).
//
// Null betroffene Zeilen sind kein Fehlschlag: Dasselbe Token ist dann schon widerrufen, das
// Ziel ist erreicht (repository.WiderrufeToken).
func (s *Sperrliste) Widerrufe(ctx context.Context, token string, laeuftAb time.Time) error {
	hash := hashDesTokens(token)
	// Kurze Frist, weil die Abmeldung darauf wartet. Der Widerruf läuft zu Ende, auch wenn der
	// Browser die Verbindung trennt: Abgebrochen bliebe das Token bis zu seinem Ablauf gültig.
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()

	if err := repository.WiderrufeToken(ctx, s.pool, hash, laeuftAb); err != nil {
		log.Printf("token-blacklist: WARN Token konnte nicht widerrufen werden: %v", err)
		return err
	}
	return nil
}

// istWiderrufen prüft, ob das Token in revoked_tokens steht. Ein Fehler der Datenbank kommt als
// Fehler zurück und nicht als „widerrufen": Der Aufrufer lehnt die Anfrage in beiden Fällen ab,
// antwortet aber verschieden, widerrufen mit 401, nicht prüfbar mit 503.
func (s *Sperrliste) istWiderrufen(ctx context.Context, token string) (bool, error) {
	hash := hashDesTokens(token)
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	widerrufen, err := repository.IstTokenWiderrufen(ctx, s.pool, hash)
	if err != nil {
		return false, err
	}
	return widerrufen, nil
}

// raeumeAb löscht die Tokens aus revoked_tokens, deren Ablaufzeit vorbei ist.
func (s *Sperrliste) raeumeAb(ctx context.Context) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	if err := repository.LoescheAbgelaufeneWiderrufe(ctx, s.pool); err != nil {
		log.Printf("token-blacklist: Aufräumen abgelaufener Tokens fehlgeschlagen: %v", err)
	}
}
