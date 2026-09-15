package repository

// Idempotenz-Schlüssel der Theke (Tabelle idempotency_keys). Seit dem 15.09.2026 wird der
// Schlüssel VOR der Arbeit reserviert (Status 0, Platzhalter als Antwort) und die Antwort
// danach in die Reservierung geschrieben. Vorher wurde nur nach der Arbeit eingefügt: Eine
// zweite Anfrage mit demselben Schlüssel, die nach dem Commit der ersten und vor dem
// Speichern ihrer Antwort eintraf, fand nichts und buchte neu — das Buch lag schon beim Kind,
// also wurde es zurückgenommen (api/idempotenz_pg_test.go).

import (
	"context"
	"errors"

	"bibliothek/db"

	"github.com/jackc/pgx/v5"
)

// IdempotenzAntwort ist, was unter einem Schlüssel steht: die gespeicherte Antwort, oder bei
// Status 0 eine Reservierung, deren Arbeit noch läuft.
type IdempotenzAntwort struct {
	Daten  []byte
	Status int
}

// InArbeit sagt, ob der Schlüssel reserviert, aber noch ohne Antwort ist.
func (a IdempotenzAntwort) InArbeit() bool { return a.Status == 0 }

// ErrIdempotenzNichtReserviert heißt: Die Antwort ließ sich nicht ablegen, weil der Schlüssel
// nicht (mehr) dieser Anfrage gehört — keine Reservierung, oder schon beantwortet.
var ErrIdempotenzNichtReserviert = errors.New("idempotenz: schlüssel nicht reserviert")

// ReserviereIdempotenzSchluessel versucht, den Schlüssel zu reservieren. reserviert=true: der
// Aufrufer darf arbeiten und speichert danach die Antwort. Sonst gehört der Schlüssel schon
// jemandem, und antwort sagt, ob dessen Antwort da ist (Status ≥ 200) oder noch kommt
// (Status 0). Eine Reservierung, die älter als 60 Sekunden ist, gilt als verwaist (der
// Server ist zwischen Reservierung und Antwort gestorben) und wird übernommen — unter dem
// Zeilen-Lock von ON CONFLICT nimmt sie genau einer.
func ReserviereIdempotenzSchluessel(ctx context.Context, pool db.PgxPoolIface, schluessel string) (reserviert bool, antwort *IdempotenzAntwort, err error) {
	var k string
	err = pool.QueryRow(ctx, `
		INSERT INTO idempotency_keys (idempotency_key, response_data, status_code)
		VALUES ($1, '{"in_arbeit": true}', 0)
		ON CONFLICT (idempotency_key) DO UPDATE SET created_at = CURRENT_TIMESTAMP
		WHERE idempotency_keys.status_code = 0
		  AND idempotency_keys.created_at < CURRENT_TIMESTAMP - INTERVAL '60 seconds'
		RETURNING idempotency_key
	`, schluessel).Scan(&k)
	if err == nil {
		return true, nil, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return false, nil, err
	}
	antwort, err = LiesIdempotenzAntwort(ctx, pool, schluessel)
	if err != nil {
		return false, nil, err
	}
	return false, antwort, nil
}

// LiesIdempotenzAntwort liest, was unter dem Schlüssel steht; pgx.ErrNoRows, wenn nichts.
func LiesIdempotenzAntwort(ctx context.Context, pool db.PgxPoolIface, schluessel string) (*IdempotenzAntwort, error) {
	var a IdempotenzAntwort
	if err := pool.QueryRow(ctx,
		`SELECT response_data, status_code FROM idempotency_keys WHERE idempotency_key = $1`,
		schluessel).Scan(&a.Daten, &a.Status); err != nil {
		return nil, err
	}
	return &a, nil
}

// SpeichereIdempotenzAntwort schreibt die Antwort in die eigene Reservierung. Eine fremde
// oder schon beantwortete Zeile (Status ≠ 0) bleibt unberührt.
func SpeichereIdempotenzAntwort(ctx context.Context, pool db.PgxPoolIface, schluessel string, daten []byte, status int) error {
	tag, err := pool.Exec(ctx,
		`UPDATE idempotency_keys SET response_data = $2, status_code = $3 WHERE idempotency_key = $1 AND status_code = 0`,
		schluessel, daten, status)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrIdempotenzNichtReserviert
	}
	return nil
}

// GibIdempotenzSchluesselFrei löscht die eigene Reservierung — nach einem Serverfehler, damit
// die Wiederholung neu bucht statt den Fehler zurückzubekommen. freigegeben=false ohne
// Fehler: Es gab keine Reservierung (mehr) zu löschen.
func GibIdempotenzSchluesselFrei(ctx context.Context, pool db.PgxPoolIface, schluessel string) (freigegeben bool, err error) {
	tag, err := pool.Exec(ctx,
		`DELETE FROM idempotency_keys WHERE idempotency_key = $1 AND status_code = 0`, schluessel)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}
