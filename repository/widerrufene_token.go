package repository

// widerrufene_token.go — die Anweisungen zur Sperrliste revoked_tokens. In der Tabelle steht
// der Hash eines Tokens, den auth/ bildet, nicht das Token selbst.

import (
	"context"
	"time"
)

// WiderrufeToken trägt die Signatur mit der Ablaufzeit des Tokens ein. Steht sie schon in der
// Liste, ändert sich nichts: Das Ziel ist dann erreicht.
func WiderrufeToken(ctx context.Context, db DBQueryer, signatur string, laeuftAb time.Time) error {
	_, err := db.Exec(ctx, `
		INSERT INTO revoked_tokens (token_signature, expires_at)
		VALUES ($1, $2)
		ON CONFLICT (token_signature) DO NOTHING
	`, signatur, laeuftAb)
	return err
}

// TokenWiderrufen sagt, ob die Signatur in der Sperrliste steht.
func TokenWiderrufen(ctx context.Context, db DBQueryer, signatur string) (bool, error) {
	var widerrufen bool
	err := db.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM revoked_tokens WHERE token_signature = $1)
	`, signatur).Scan(&widerrufen)
	return widerrufen, err
}

// LoescheAbgelaufeneWiderrufe entfernt die Einträge, deren Token ohnehin abgelaufen ist.
func LoescheAbgelaufeneWiderrufe(ctx context.Context, db DBQueryer) error {
	_, err := db.Exec(ctx, `DELETE FROM revoked_tokens WHERE expires_at < NOW()`)
	return err
}
