package auth

import (
	"context"
	"fmt"
	"testing"
	"time"

	"bibliothek/internal/pgtest"
)

// Die Sperrliste an der echten Tabelle revoked_tokens: Ein widerrufenes Token gilt als
// widerrufen, ein zweiter Widerruf desselben Tokens ist kein Fehler, und das Aufräumen nimmt
// nur, was abgelaufen ist. Ein nachgespielter Pool belegt davon nichts: Die Zusagen hängen am
// Schlüssel der Tabelle und am Vergleich mit der Uhr der Datenbank.
func TestTokenBlacklist_WiderrufUndAufraeumenAnDerTabelle(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	b := NewTokenBlacklist(pool)
	t.Cleanup(b.Stop)

	lauf := time.Now().UnixNano()
	gueltig := fmt.Sprintf("token-gueltig-%d", lauf)
	abgelaufen := fmt.Sprintf("token-abgelaufen-%d", lauf)
	fremd := fmt.Sprintf("token-nie-widerrufen-%d", lauf)
	t.Cleanup(func() {
		if _, err := pool.Exec(context.Background(), `DELETE FROM revoked_tokens WHERE token_signature = ANY($1)`,
			[]string{hashToken(gueltig), hashToken(abgelaufen)}); err != nil {
			t.Logf("Aufräumen der Probe: %v", err)
		}
	})

	istWiderrufen := func(token string) bool {
		t.Helper()
		widerrufen, err := b.IsBlacklisted(token)
		if err != nil {
			t.Fatalf("IsBlacklisted: %v", err)
		}
		return widerrufen
	}
	zeilen := func(token string) int {
		t.Helper()
		var n int
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM revoked_tokens WHERE token_signature = $1`,
			hashToken(token)).Scan(&n); err != nil {
			t.Fatalf("Zeilen zählen: %v", err)
		}
		return n
	}

	if istWiderrufen(gueltig) {
		t.Fatal("ein nie widerrufenes Token gilt als widerrufen")
	}
	if err := b.Add(gueltig, time.Now().Add(time.Hour)); err != nil {
		t.Fatalf("Widerruf: %v", err)
	}
	if !istWiderrufen(gueltig) {
		t.Error("das widerrufene Token gilt nicht als widerrufen")
	}
	if istWiderrufen(fremd) {
		t.Error("ein anderes Token gilt nach dem Widerruf als widerrufen")
	}
	// In der Tabelle steht der Hash, nicht das Token.
	var roh int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM revoked_tokens WHERE token_signature = $1`, gueltig).Scan(&roh); err != nil {
		t.Fatalf("Zeilen zählen: %v", err)
	}
	if roh != 0 {
		t.Error("das Token steht im Klartext in revoked_tokens")
	}

	// Ein zweiter Widerruf desselben Tokens, mit anderer Ablaufzeit: kein Fehler, eine Zeile.
	if err := b.Add(gueltig, time.Now().Add(2*time.Hour)); err != nil {
		t.Errorf("zweiter Widerruf desselben Tokens: %v", err)
	}
	if n := zeilen(gueltig); n != 1 {
		t.Errorf("%d Zeilen nach zwei Widerrufen desselben Tokens, erwartet 1", n)
	}

	// Das Aufräumen nimmt das abgelaufene und lässt das gültige.
	if err := b.Add(abgelaufen, time.Now().Add(-time.Minute)); err != nil {
		t.Fatalf("Widerruf des abgelaufenen Tokens: %v", err)
	}
	if n := zeilen(abgelaufen); n != 1 {
		t.Fatalf("%d Zeilen für das abgelaufene Token vor dem Aufräumen, erwartet 1", n)
	}
	b.cleanup()
	if n := zeilen(abgelaufen); n != 0 {
		t.Errorf("das abgelaufene Token steht nach dem Aufräumen noch in der Tabelle")
	}
	if !istWiderrufen(gueltig) {
		t.Error("das Aufräumen hat ein Token genommen, das noch nicht abgelaufen ist")
	}
}
