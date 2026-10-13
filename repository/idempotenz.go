package repository

// Die Idempotenz-Schlüssel der Theke (Tabelle idempotency_keys). Der Schlüssel wird vor der
// Arbeit reserviert (Status 0, Platzhalter als Antwort), und die Antwort wird danach in die
// Reservierung geschrieben. Würde erst nach der Arbeit eingefügt, fände eine zweite Anfrage mit
// demselben Schlüssel, die zwischen dem Festschreiben der ersten und dem Ablegen ihrer Antwort
// eintrifft, nichts und buchte neu: Das Buch liegt schon beim Kind, also nähme sie es zurück
// (api/theke/idempotenz_pg_test.go).

import (
	"context"
	"errors"
	"time"

	"bibliothek/db"

	"github.com/jackc/pgx/v5"
)

// IdempotenzReservierungsfrist ist die Frist, nach der eine Reservierung (Status 0) als
// verwaist gilt; die nächste Anfrage mit demselben Schlüssel übernimmt sie. Verwaist heißt: Der
// Server ist zwischen Reservierung und Antwort gestorben, nicht: Er arbeitet noch. Das stimmt
// nur, solange jede Arbeit unter dem Schlüssel vor Ablauf dieser Frist abgebrochen und ihre
// Antwort abgelegt ist: Die TimeoutMiddleware (api.StandardBearbeitungsfrist) beendet die
// Arbeit, saveToCache (api/theke, IdempotenzSpeicherfrist) schreibt danach noch.
// api/idempotenz_fristen_test.go hält die drei Werte aneinander.
const IdempotenzReservierungsfrist = 60 * time.Second

// IdempotenzAntwort ist, was unter einem Schlüssel steht: die gespeicherte Antwort, oder bei
// Status 0 eine Reservierung, deren Arbeit noch läuft.
type IdempotenzAntwort struct {
	Daten  []byte
	Status int
}

// IstInArbeit meldet, ob der Schlüssel reserviert, aber noch ohne Antwort ist.
func (a IdempotenzAntwort) IstInArbeit() bool { return a.Status == 0 }

// errIdempotenzNichtReserviert meldet, dass sich die Antwort nicht ablegen ließ, weil der
// Schlüssel dieser Anfrage nicht oder nicht mehr gehört: keine Reservierung, oder schon
// beantwortet.
var errIdempotenzNichtReserviert = errors.New("idempotenz: schlüssel nicht reserviert")

// ReserviereIdempotenzSchluessel versucht, den Schlüssel zu reservieren. Mit reserviert = true
// darf der Rufer arbeiten und legt danach die Antwort ab. Sonst gehört der Schlüssel schon
// jemandem, und antwort sagt, ob dessen Antwort da ist (Status ab 200) oder noch kommt (Status
// 0). Eine Reservierung, die älter ist als IdempotenzReservierungsfrist, gilt als verwaist und
// wird übernommen; unter der Zeilensperre von ON CONFLICT nimmt sie genau einer.
func ReserviereIdempotenzSchluessel(ctx context.Context, pool db.PgxPoolIface, schluessel string) (reserviert bool, antwort *IdempotenzAntwort, err error) {
	var k string
	err = pool.QueryRow(ctx, `
		INSERT INTO idempotency_keys (idempotency_key, response_data, status_code)
		VALUES ($1, '{"in_arbeit": true}', 0)
		ON CONFLICT (idempotency_key) DO UPDATE SET created_at = CURRENT_TIMESTAMP
		WHERE idempotency_keys.status_code = 0
		  AND idempotency_keys.created_at < CURRENT_TIMESTAMP - make_interval(secs => $2)
		RETURNING idempotency_key
	`, schluessel, IdempotenzReservierungsfrist.Seconds()).Scan(&k)
	if err == nil {
		return true, nil, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return false, nil, err
	}
	antwort, err = LadeIdempotenzAntwort(ctx, pool, schluessel)
	if err != nil {
		return false, nil, err
	}
	return false, antwort, nil
}

// LadeIdempotenzAntwort liest, was unter dem Schlüssel steht; pgx.ErrNoRows, wenn nichts.
func LadeIdempotenzAntwort(ctx context.Context, pool db.PgxPoolIface, schluessel string) (*IdempotenzAntwort, error) {
	var a IdempotenzAntwort
	if err := pool.QueryRow(ctx,
		`SELECT response_data, status_code FROM idempotency_keys WHERE idempotency_key = $1`,
		schluessel).Scan(&a.Daten, &a.Status); err != nil {
		return nil, err
	}
	return &a, nil
}

// SchreibeIdempotenzAntwort schreibt die Antwort in die eigene Reservierung. Eine fremde oder
// schon beantwortete Zeile (Status ungleich 0) bleibt unberührt.
func SchreibeIdempotenzAntwort(ctx context.Context, pool db.PgxPoolIface, schluessel string, daten []byte, status int) error {
	tag, err := pool.Exec(ctx,
		`UPDATE idempotency_keys SET response_data = $2, status_code = $3 WHERE idempotency_key = $1 AND status_code = 0`,
		schluessel, daten, status)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errIdempotenzNichtReserviert
	}
	return nil
}

// UebernimmIdempotenzAntwort macht eine abgelegte Antwort wieder zur Reservierung. Das
// Nachbuchen braucht das, wenn unter dem Schlüssel eine Antwort des Scans an der Theke steht
// und trotzdem noch gebucht werden muss (Fehlerantwort, oder nur die Fremdrückgabe gebucht).
// Nur wer genau die gelesene Antwort vorfindet, Status und Inhalt, übernimmt; ein zweiter
// Aufruf mit derselben Portion findet danach die Reservierung und bucht nicht daneben.
// created_at rückt vor, damit die Übernahme als verwaist ab jetzt zählt.
func UebernimmIdempotenzAntwort(ctx context.Context, pool db.PgxPoolIface, schluessel string, alt IdempotenzAntwort) (bool, error) {
	tag, err := pool.Exec(ctx, `
		UPDATE idempotency_keys
		SET status_code = 0, response_data = '{"in_arbeit": true}', created_at = CURRENT_TIMESTAMP
		WHERE idempotency_key = $1 AND status_code = $2 AND response_data = $3::jsonb`,
		schluessel, alt.Status, alt.Daten)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}

// StelleIdempotenzAntwortWiederHer legt eine übernommene Antwort zurück in die eigene
// Reservierung, nach einem Serverfehler des Nachbuchens: Die nächste Runde findet die Antwort
// des Scans wieder vor.
func StelleIdempotenzAntwortWiederHer(ctx context.Context, pool db.PgxPoolIface, schluessel string, alt IdempotenzAntwort) error {
	tag, err := pool.Exec(ctx,
		`UPDATE idempotency_keys SET response_data = $2, status_code = $3 WHERE idempotency_key = $1 AND status_code = 0`,
		schluessel, alt.Daten, alt.Status)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errIdempotenzNichtReserviert
	}
	return nil
}

// GibIdempotenzSchluesselFrei löscht die eigene Reservierung nach einem Serverfehler, damit die
// Wiederholung neu bucht und nicht den Fehler zurückbekommt. freigegeben = false ohne Fehler
// heißt: Es gab keine Reservierung mehr zu löschen.
func GibIdempotenzSchluesselFrei(ctx context.Context, pool db.PgxPoolIface, schluessel string) (freigegeben bool, err error) {
	tag, err := pool.Exec(ctx,
		`DELETE FROM idempotency_keys WHERE idempotency_key = $1 AND status_code = 0`, schluessel)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

// LoescheAlteIdempotenzSchluessel entfernt Schlüssel, die älter sind als 24 Stunden, und
// liefert ihre Zahl. Die Wiederholung eines Scans kommt nach Sekunden oder Minuten; ohne das
// Aufräumen wüchse die Tabelle mit jeder Buchung.
func LoescheAlteIdempotenzSchluessel(ctx context.Context, pool db.PgxPoolIface) (int64, error) {
	tag, err := pool.Exec(ctx, "DELETE FROM idempotency_keys WHERE created_at < NOW() - INTERVAL '24 hours'")
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}
